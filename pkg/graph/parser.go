// TODO handle moving with PreviousAddress
// TODO better data shaping for ease of manipulation
// TODO count/foreach for modules
// TODO dynamic
// TODO edge attributes to have granular dependency shown
// TODO drift
package graph

import (
	"errors"
	"fmt"

	tfjson "github.com/hashicorp/terraform-json"
)

type NodeKind string

const (
	ResourceNode     NodeKind = "resource"
	DataNode         NodeKind = "data"
	LoopResourceNode NodeKind = "resource_loop"
	ModuleNode       NodeKind = "module"
	LoopModuleNode   NodeKind = "module_loop"
	VariableNode     NodeKind = "variable"
	OutputNode       NodeKind = "output"
)

type ResourceState string

const (
	StateModify  ResourceState = "modify"
	StateDelete  ResourceState = "delete"
	StateCreate  ResourceState = "create"
	StateNoop    ResourceState = "noop"
	StateReplace ResourceState = "replace"
)

type EdgeType string

const (
	EdgeDirect     EdgeType = "direct"
	EdgeTransitive EdgeType = "transitive"
	EdgeInstance   EdgeType = "instance"
)

type Graph struct {
	Nodes map[string]Node `json:"nodes"`
	Edges []Edge          `json:"edges"`
	seen  map[edgeKey]Edge
}

type Node struct {
	Kind       NodeKind            `json:"kind"`
	Name       string              `json:"name"`
	Type       string              `json:"type,omitempty"`
	Mode       tfjson.ResourceMode `json:"mode,omitempty"`
	State      ResourceState       `json:"state,omitempty"`
	Address    string              `json:"address"`
	Parent     string              `json:"parent,omitempty"`
	Parameters []string            `json:"parameters"`
	Changes    *NodeChange         `json:"changes"`
	isInstance bool
}

type Edge struct {
	From               string   `json:"from"`
	To                 string   `json:"to"`
	Type               EdgeType `json:"type"`
	RelevantAttributes []string `json:"attributes,omitempty"`
}

type edgeKey struct {
	from string
	to   string
}

type ModuleConfig struct {
	Name    string
	Address string
	Parent  string
}

type NodeChange struct {
	Before map[string]any `json:"before"`
	After  map[string]any `json:"after"`
}

// Discovers nodes for: resources, modules, variables, outputs.
// Works on configuration section, needs mock of a module call for root configuration.
// DOES NOT handle deleted nodes as they do not exist in configuration when removed.
// Also creates a node for count/for_each resources,
// which are future parents for individual indexed instances(discovered from resource_changes).
// TODO should move parameters to enrichment phase?
func (g *Graph) discoverNodes(mc *tfjson.ModuleCall, cfg ModuleConfig) {
	if cfg.Address != "" {
		// module address is non-empty only for child modules
		g.Nodes[cfg.Address] = Node{
			Kind:       ModuleNode,
			Name:       cfg.Name,
			Address:    cfg.Address,
			Parent:     cfg.Parent,
			Parameters: processInputs(mc),
		}
	}

	for _, r := range mc.Module.Resources {
		absoluteAddress := tfJoin(cfg.Address, r.Address)

		kind := ResourceNode
		if r.CountExpression != nil || r.ForEachExpression != nil {
			kind = LoopResourceNode
		}

		g.Nodes[absoluteAddress] = Node{
			Kind:       kind,
			Name:       r.Name,
			Type:       r.Type,
			Mode:       r.Mode,
			Address:    absoluteAddress,
			Parent:     cfg.Address,
			Parameters: processParams(r),
		}
	}

	// process module variables
	for varname := range mc.Module.Variables {
		absoluteAddress := tfJoin(cfg.Address, "var", varname)
		g.Nodes[absoluteAddress] = Node{
			Kind:       VariableNode,
			Name:       varname,
			Address:    absoluteAddress,
			Parent:     cfg.Address,
			Parameters: []string{},
		}
	}

	// process module outputs
	for name := range mc.Module.Outputs {

		fullname := tfJoin(cfg.Address, name)

		g.Nodes[fullname] = Node{
			Kind:       OutputNode,
			Name:       name,
			Address:    fullname,
			Parent:     cfg.Address,
			Parameters: []string{},
		}
	}

	// recurse into child modules
	for name, m := range mc.Module.ModuleCalls {
		moduleAbsoluteAddress := tfJoin(cfg.Address, "module", name)

		g.discoverNodes(m, ModuleConfig{
			Name:    name,
			Address: moduleAbsoluteAddress,
			Parent:  cfg.Address,
		})
	}
}

// Discovers child nodes for loop nodes
func (g *Graph) discoverInstanceNodes(p *tfjson.Plan) {
	for _, c := range p.ResourceChanges {
		if c.Index != nil {
			// this node is instance node, as it links to its count/for_each index
			var name string
			switch v := c.Index.(type) {
			case float64:
				// FOR SOME REASON it is float not int
				name = fmt.Sprintf("%s[%d]", c.Name, int(v))
			case string:
				name = fmt.Sprintf("%s[\"%s\"]", c.Name, v)
			}

			parent := tfJoin(c.ModuleAddress, c.Type, c.Name)
			g.Nodes[c.Address] = Node{
				Kind:       ResourceNode,
				Name:       name,
				Mode:       c.Mode,
				Type:       c.Type,
				Address:    c.Address,
				Parent:     parent,
				isInstance: true,
				Parameters: []string{},
			}

			if _, ok := g.Nodes[parent]; !ok {
				// TODO: parent does not exist, so whole loop node was
				g.Nodes[parent] = Node{
					Kind:       LoopResourceNode,
					Name:       c.Name,
					Mode:       c.Mode,
					Type:       c.Type,
					Address:    parent,
					Parent:     c.ModuleAddress,
					Parameters: []string{},
				}
			}

		}
	}
}

func (g *Graph) discoverInstanceEdges(p *tfjson.Plan) error {
	if p.PriorState == nil || p.PriorState.Values == nil {
		return fmt.Errorf("no prior state")
	}

	err := g.discoverInstanceEdgesRecurse(p.PriorState.Values.RootModule)

	return err
}

func (g *Graph) discoverInstanceEdgesRecurse(m *tfjson.StateModule) error {
	var errs []error
	for _, r := range m.Resources {
		node, ok := g.Nodes[r.Address]
		if !ok {
			logger.Warn("instance edge: no node for address", "address", r.Address)
			errs = append(errs, fmt.Errorf("discover instance edges(root): no node with address %v", r.Address))
		} else if node.isInstance {
			for _, dep := range r.DependsOn {
				if err := g.AddEdge(Edge{
					From: dep,
					To:   node.Address,
					Type: EdgeInstance,
				}); err != nil {
					logger.Warn("instance edge: failed to add edge", "from", dep, "to", node.Address, "error", err)
				}
			}
		}
	}

	for _, child := range m.ChildModules {
		errs = append(errs, g.discoverInstanceEdgesRecurse(child))
	}

	return errors.Join(errs...)
}

// Discovers deleted nodes from Resource changes.
// Modules are not real resources so modules need to be inferred from their past children if missing.
// ModuleAddress is full address of module.
func (g *Graph) discoverDeletedResourceNodes(p *tfjson.Plan) {
	for _, c := range p.ResourceChanges {
		if c.Change.Actions.Delete() && c.Index == nil {
			g.Nodes[c.Address] = Node{
				Kind:    ResourceNode,
				Name:    c.Name,
				Type:    c.Type,
				Mode:    c.Mode,
				Address: c.Address,
				Parent:  c.ModuleAddress,
			}
		}
	}
}

func (g *Graph) discoverDeletedResourceEdges(p *tfjson.Plan) error {
	if p.PriorState == nil || p.PriorState.Values == nil {
		return fmt.Errorf("no prior state")
	}

	err := g.processStateModule(p.PriorState.Values.RootModule)

	return err
}

func (g *Graph) discoverDeletedLoopEdges() {
	for _, e := range g.Edges {
		if e.Type != EdgeInstance {
			continue
		}

		if to, ok := g.Nodes[e.To]; ok && to.isInstance && to.Parent != "" {
			container := g.Nodes[to.Parent]
			if container.Kind == LoopResourceNode && container.State == StateDelete {
				g.AddEdge(Edge{
					From: e.From,
					To:   to.Parent,
					Type: EdgeTransitive,
				})
			}
		}
	}
}

// Recursive helper to process prior state to find deleted edges with node check
func (g *Graph) processStateModule(m *tfjson.StateModule) error {
	var errs []error
	for _, r := range m.Resources {
		node, ok := g.Nodes[r.Address]
		if !ok {
			logger.Warn("deleted edge: no node for address", "address", r.Address)
			errs = append(errs, fmt.Errorf("discover deleted edges(root): no node with address %v", r.Address))
		} else if node.State == StateDelete && !node.isInstance {
			for _, dep := range r.DependsOn {
				if err := g.AddEdge(Edge{
					From: dep,
					To:   node.Address,
					Type: EdgeTransitive,
				}); err != nil {
					logger.Warn("deleted edge: failed to add edge", "from", dep, "to", node.Address, "error", err)
				}
			}
		}
	}

	for _, child := range m.ChildModules {
		errs = append(errs, g.processStateModule(child))
	}

	return errors.Join(errs...)
}

func (g *Graph) discoverEdges(mc *tfjson.ModuleCall, cfg ModuleConfig) {
	for _, r := range mc.Module.Resources {
		absoluteAddress := tfJoin(cfg.Address, r.Address)

		// implicit dependencies
		walkExpressions(r.Expressions, func(_, from string) {
			if err := g.AddEdge(Edge{
				From: tfJoin(cfg.Address, from),
				To:   absoluteAddress,
				Type: EdgeDirect,
			}); err != nil {
				logger.Warn("edge discovery: failed to add implicit dependency edge", "from", from, "to", absoluteAddress, "error", err)
			}
		})

		// explicit dependencies
		for _, dep := range r.DependsOn {
			if err := g.AddEdge(Edge{
				From: tfJoin(cfg.Address, dep),
				To:   absoluteAddress,
				Type: EdgeDirect,
			}); err != nil {
				logger.Warn("edge discovery: failed to add explicit dependency edge", "from", dep, "to", absoluteAddress, "error", err)
			}
		}

		// implicit loop dependencies
		if r.CountExpression != nil {
			countExpressions := make(map[string]*tfjson.Expression)
			countExpressions["count_expression"] = r.CountExpression
			walkExpressions(countExpressions, func(_, from string) {
				if err := g.AddEdge(Edge{
					From: tfJoin(cfg.Address, from),
					To:   absoluteAddress,
					Type: EdgeDirect,
				}); err != nil {
					logger.Warn("edge discovery: failed to add count expression edge", "from", from, "to", absoluteAddress, "error", err)
				}
			})
		}

		if r.ForEachExpression != nil {
			foreachExpressions := make(map[string]*tfjson.Expression)
			foreachExpressions["for_each_expression"] = r.ForEachExpression
			walkExpressions(foreachExpressions, func(_, from string) {
				if err := g.AddEdge(Edge{
					From: tfJoin(cfg.Address, from),
					To:   absoluteAddress,
					Type: EdgeDirect,
				}); err != nil {
					logger.Warn("edge discovery: failed to add for_each expression edge", "from", from, "to", absoluteAddress, "error", err)
				}
			})
		}

	}

	// process module variables
	// Expressions are nil for root module
	walkExpressions(mc.Expressions, func(to, from string) {
		if err := g.AddEdge(Edge{
			From: tfJoin(cfg.Parent, from),
			To:   tfJoin(cfg.Address, "var", to),
			Type: EdgeDirect,
		}); err != nil {
			logger.Warn("edge discovery: failed to add variable edge", "from", from, "to", to, "error", err)
		}
	})

	for name, o := range mc.Module.Outputs {
		fullname := tfJoin(cfg.Address, name)

		expressions := make(map[string]*tfjson.Expression)
		expressions[name] = o.Expression // TODO meh

		walkExpressions(expressions, func(_, from string) {
			if err := g.AddEdge(Edge{
				From: tfJoin(cfg.Address, from),
				To:   fullname,
				Type: EdgeDirect,
			}); err != nil {
				logger.Warn("edge discovery: failed to add output edge", "from", from, "to", fullname, "error", err)
			}
		})
	}

	// recurse into child modules
	for name, m := range mc.Module.ModuleCalls {
		moduleAbsoluteAddress := tfJoin(cfg.Address, "module", name)

		g.discoverEdges(m, ModuleConfig{
			Name:    name,
			Address: moduleAbsoluteAddress,
			Parent:  cfg.Address,
		})
	}
}

func (g *Graph) ensureModules() {
	for _, node := range g.Nodes {
		current := node.Parent
		if node.isInstance {
			continue
		}
		for current != "" {
			if _, ok := g.Nodes[current]; ok {
				break
			}
			name, parent, err := getModuleParts(current)

			if err != nil {
				logger.Warn("ensure modules: failed to parse module address", "address", current, "error", err)
				break
			}

			g.Nodes[current] = Node{
				Kind:    ModuleNode,
				Name:    name,
				Address: current,
				Parent:  parent,
			}
			current = parent
		}
	}
}

// Safely add edge with deduplication
func (g *Graph) AddEdge(e Edge) error {
	to, okTo := g.Nodes[e.To]
	from, okFrom := g.Nodes[e.From]

	if !okTo || !okFrom {
		return fmt.Errorf("unhandled edge between %v -> %v", from.Address, to.Address)
	}

	if from.Kind == ModuleNode && to.Kind == OutputNode {
		return fmt.Errorf("edge module->output was tried to create(%v -> %v)", from.Address, to.Address)
	}

	if to.isInstance || from.isInstance {
		e.Type = EdgeInstance
	}

	key := edgeKey{from: e.From, to: e.To}
	if _, ok := g.seen[key]; !ok {
		g.seen[key] = e
		g.Edges = append(g.Edges, e)
	}

	return nil
}

// converts transitive edges to direct edges
// when there is no other path between the same endpoints.
// Reachability is checked against all other edges (direct + transitive),
// excluding the edge being evaluated, so that redundant transitive edges
// stay transitive while indispensable ones become direct.
func (g *Graph) promoteTransitiveEdges() {
	for i := range g.Edges {
		e := &g.Edges[i]
		if e.Type != EdgeTransitive {
			continue
		}
		// Build adjacency from all OTHER edges, skipping the current one
		adj := make(map[string][]string)
		for j := range g.Edges {
			if j == i {
				continue
			}
			other := &g.Edges[j]
			adj[other.From] = append(adj[other.From], other.To)
		}
		if !g.hasPath(e.From, e.To, adj) {
			e.Type = EdgeDirect
		}
	}
}

// hasPath returns true if `to` is reachable from `from`
// using the given adjacency list (BFS).
func (g *Graph) hasPath(from, to string, adj map[string][]string) bool {
	visited := make(map[string]bool)
	queue := []string{from}
	visited[from] = true

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		if current == to {
			return true
		}
		for _, neighbor := range adj[current] {
			if !visited[neighbor] {
				visited[neighbor] = true
				queue = append(queue, neighbor)
			}
		}
	}
	return false
}

// Walks expression maps and handles references.
// callback accepts expression key which is expression name
// and reference itself(the reference address)
func walkExpressions(expressions map[string]*tfjson.Expression, handleReference func(key, ref string)) {
	for key, e := range expressions {
		if e.ConstantValue != tfjson.UnknownConstantValue {
			// this is just static value
			continue
		}
		for _, ref := range e.References {
			handleReference(key, ref)
		}

		for _, nested := range e.NestedBlocks {
			walkExpressions(nested, handleReference) // TODO have to test with nested blocks
		}
	}
}

// should really just take plan
func New(plan *tfjson.Plan) (*Graph, error) {

	root := tfjson.ModuleCall{
		Module: plan.Config.RootModule,
	}

	var graph = Graph{
		Nodes: make(map[string]Node, 100),
		Edges: make([]Edge, 0, 100),
		seen:  make(map[edgeKey]Edge, 100),
	}

	// Discover Nodes
	graph.discoverNodes(&root, ModuleConfig{})
	graph.discoverDeletedResourceNodes(plan)
	graph.discoverInstanceNodes(plan)

	// make sure all parents exist
	graph.ensureModules()

	// Enrich nodes
	// TODO Move ProcessParams passes here
	if err := graph.discoverResourceChanges(plan); err != nil {
		logger.Warn("graph enrichment: some resource changes had no matching nodes", "error", err)
	}
	graph.discoverParentState()

	// Discover edges
	graph.discoverEdges(&root, ModuleConfig{})
	if err := graph.discoverDeletedResourceEdges(plan); err != nil {
		logger.Warn("graph: errors during deleted resource edge discovery", "error", err)
	} // call only after enrichments
	if err := graph.discoverInstanceEdges(plan); err != nil {
		logger.Warn("graph: errors during instance edge discovery", "error", err)
	}
	graph.discoverDeletedLoopEdges()
	graph.promoteTransitiveEdges()

	return &graph, nil
}
