// TODO ! count, for_each(parent nodes from config, individual in changes) - parse new nodes and parent nodes, parse edges, parse deleted nodes, parse deleted edges
// TODO ! check secrets
// TODO dynamic
// TODO edge attributes to have granular dependency shown
package graph

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	tfjson "github.com/hashicorp/terraform-json"
)

const (
	HCL_PATH_DELIMITER      = "."
	MAX_EXPECTED_PATH_PARTS = 3
)

type NodeKind string

const (
	ResourceNode NodeKind = "resource"
	ModuleNode   NodeKind = "module"
	VariableNode NodeKind = "variable"
	OutputNode   NodeKind = "output"
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
	EdgeTransitive EdgeType = "transitive" // potentially
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
// which are future parents for individual indexed instances(discovered from resource_changes)
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

		g.Nodes[absoluteAddress] = Node{
			Kind:       ResourceNode,
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

// Discovers deleted nodes from Resource changes.
// Modules are not real resources so modules need to be inferred if missing.
// ModuleAddress is full address of module.
func (g *Graph) discoverDeletedNodes(p *tfjson.Plan) {
	for _, c := range p.ResourceChanges {
		if c.Change.Actions.Delete() {
			g.Nodes[c.Address] = Node{
				Kind:    ResourceNode,
				Name:    c.Name,
				Type:    c.Type,
				Mode:    c.Mode,
				Address: c.Address,
				Parent:  c.ModuleAddress,
			}

			currentModule := c.ModuleAddress

			for currentModule != "" {
				if _, ok := g.Nodes[currentModule]; ok {
					break
				}

				name, parent, err := getModuleParts(currentModule)

				if err != nil {
					// TODO handle err
					break
				}

				g.Nodes[c.ModuleAddress] = Node{
					Kind:    ModuleNode,
					State:   StateDelete,
					Name:    name,
					Address: currentModule,
					Parent:  parent,
				}

				currentModule = parent
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

func (g *Graph) processStateModule(m *tfjson.StateModule) error {
	var errs []error
	for _, r := range m.Resources {
		node, ok := g.Nodes[r.Address]
		if !ok {
			errs = append(errs, fmt.Errorf("discover deleted edges(root): no node with address %v", r.Address))
		}

		if node.State == StateDelete {
			for _, dep := range r.DependsOn {
				g.AddEdge(Edge{
					From: dep,
					To:   node.Address,
				})
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
			g.AddEdge(Edge{
				From: tfJoin(cfg.Address, from),
				To:   absoluteAddress,
			})
		})

		// explicit dependencies
		for _, dep := range r.DependsOn {
			g.AddEdge(Edge{
				From: tfJoin(cfg.Address, dep),
				To:   absoluteAddress,
			})
		}
	}

	// process module variables
	// Expressions are nil for root module
	walkExpressions(mc.Expressions, func(to, from string) {
		g.AddEdge(Edge{
			From: tfJoin(cfg.Parent, from),
			To:   tfJoin(cfg.Address, "var", to),
		})
	})

	for name, o := range mc.Module.Outputs {
		fullname := tfJoin(cfg.Address, name)

		expressions := make(map[string]*tfjson.Expression)
		expressions[name] = o.Expression // TODO meh

		walkExpressions(expressions, func(_, from string) {
			g.AddEdge(Edge{
				From: tfJoin(cfg.Address, from),
				To:   fullname,
			})
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

// Enhances *existing* resource nodes with their state
// and changes in form of json objects
func (g *Graph) discoverResourceChanges(p *tfjson.Plan) error {
	var errs []error

	for _, c := range p.ResourceChanges {
		node, ok := g.Nodes[c.Address]

		if !ok {
			errs = append(errs, fmt.Errorf("Node %v not found", c.Address))
			continue
		}

		switch {
		case c.Change.Actions.Create():
			node.State = StateCreate
		case c.Change.Actions.Replace():
			node.State = StateReplace
		case c.Change.Actions.Delete():
			node.State = StateDelete
		case c.Change.Actions.NoOp():
			node.State = StateNoop
		case c.Change.Actions.Update():
			node.State = StateModify
		}

		// TODO handle moving with PreviousAddress
		// TODO process change and store only before/after values
		// with stripped secrets
		changes, err := stripChange(c.Change)

		if err != nil {
			return err
		}

		node.Changes = changes

		g.Nodes[c.Address] = node
	}

	return errors.Join(errs...)
}

func stripChange(c *tfjson.Change) (*NodeChange, error) {
	var stripped NodeChange

	after, err := stripSensitive(c.After, c.AfterSensitive)
	fmt.Printf("STRIPPED, %v, %v", after, err)

	if err != nil {
		return nil, err
	}

	before, err := stripSensitive(c.Before, c.BeforeSensitive)

	if err != nil {
		return nil, err
	}

	stripped.After = after
	stripped.Before = before

	return &stripped, nil
}

func stripSensitive(changePart any, sensitivePart any) (map[string]any, error) {
	var newPart map[string]any
	var oldPart map[string]any
	var sensitive map[string]any

	switch v := changePart.(type) {
	case map[string]any:
		oldPart = v
		if v != nil {
			newPart = make(map[string]any, len(v))
		}
	case nil:
		oldPart = nil
		newPart = nil
	default:
		return nil, fmt.Errorf("Unexpected value of change part, not map or null")
	}

	switch v := sensitivePart.(type) {
	case map[string]any:
		sensitive = v
	case bool:
		if v {
			return nil, fmt.Errorf("Unexpected TRUE for sensitive map")
		}
		sensitive = nil
	default:
		return nil, fmt.Errorf("Unexpected value of sensitive part, not map or bool")
	}

	for k, v := range oldPart {
		isSensitive, ok := sensitive[k].(bool)
		if ok && isSensitive {
			newPart[k] = "<sensitive>"
		} else {
			newPart[k] = v
		}
	}

	return newPart, nil
}

// Safely add edge with deduplication
func (g *Graph) AddEdge(e Edge) error {
	to, okTo := g.Nodes[e.To]
	from, okFrom := g.Nodes[e.From]

	if !okTo || !okFrom {
		return fmt.Errorf("Unhandled edge between %v -> %v", to, from)
	}

	key := edgeKey{from: e.From, to: e.To}
	if _, ok := g.seen[key]; !ok {
		g.seen[key] = e
		g.Edges = append(g.Edges, e)
	}

	return nil
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

func processParams(r *tfjson.ConfigResource) []string {
	parameters := make([]string, 0, len(r.Expressions))
	for key := range r.Expressions {
		parameters = append(parameters, key)
	}
	return parameters
}

func processInputs(m *tfjson.ModuleCall) []string {
	parameters := make([]string, 0, len(m.Expressions))
	for key := range m.Expressions {
		parameters = append(parameters, key)
	}
	return parameters
}

// Gets module name and its parent address
func getModuleParts(addr string) (string, string, error) {
	parts := strings.Split(addr, HCL_PATH_DELIMITER)
	if len(parts) < 2 {
		return "", "", fmt.Errorf("Could not get name from %v", addr)
	}

	name := parts[len(parts)-1]

	if len(parts) == 2 {
		return name, "", nil
	}

	parent := strings.Join(parts[:len(parts)-2], HCL_PATH_DELIMITER)

	return name, parent, nil
}

// Joins HCL address parts avoiding accidental path separators
func tfJoin(parts ...string) string {
	filtered := make([]string, 0, MAX_EXPECTED_PATH_PARTS)
	for _, part := range parts {
		if part != "" {
			filtered = append(filtered, part)
		}
	}
	return strings.Join(filtered, HCL_PATH_DELIMITER)
}

func New(r io.Reader) (*Graph, error) {
	var plan tfjson.Plan

	decoder := json.NewDecoder(r)
	if err := decoder.Decode(&plan); err != nil {
		return nil, err
	}

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
	graph.discoverDeletedNodes(&plan)
	// TODO module state(e.g. fully)

	// Enrich nodes
	graph.discoverResourceChanges(&plan)

	// Discover edges
	graph.discoverEdges(&root, ModuleConfig{})
	graph.discoverDeletedResourceEdges(&plan) // call only after determined which resources to be deleted

	// make sure all parents exist

	return &graph, nil
}
