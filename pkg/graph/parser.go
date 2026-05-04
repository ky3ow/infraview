package graph

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	tfjson "github.com/hashicorp/terraform-json"
)

type Graph struct {
	Nodes map[string]Node `json:"nodes"`
	Edges []Edge          `json:"edges"`
	seen  map[edgeKey]Edge
}

type Node struct {
	Name       string              `json:"name"`
	Type       string              `json:"type"`
	Mode       tfjson.ResourceMode `json:"mode"`
	Address    string              `json:"address"`
	Parent     string              `json:"parent,omitempty"`
	Parameters []string            `json:"parameters"`
	Changes    *tfjson.Change      `json:"changes"`
}

type Edge struct {
	From               string   `json:"from"`
	To                 string   `json:"to"`
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

func (g *Graph) WalkModule(mc *tfjson.ModuleCall, cfg ModuleConfig) {
	// TODO make this Two-pass, Nodes and only then Edges, became too convoluted
	// TODO edge attributes to have granular dependency shown
	// TODO count, for_each(parent nodes from config, individual in changes)
	// TODO check secrets
	// TODO dynamic

	// TODO clean references(dots should be an odd number to be not attributes?)
	// vars: odd
	// var.abc 1
	// module.something.var.abc 3
	// module.something.module.another.var.abc 5

	// outputs: even
	// module.something.its_output 2

	// regular resources: odd
	// some_resource 1
	// module.something.resource.name 3
	// module.something.module.another.resource.name 5

	// resource parameters: even
	// module.something.module.another.resource.name.float 6
	// some_resource.name.integer 2
	// module.something.resource.name.integer 4
	var prefix string

	if cfg.Address != "" {
		// case for child modules
		prefix = cfg.Address + "."
		g.Nodes[cfg.Address] = Node{
			Name:       cfg.Name,
			Type:       "module",
			Mode:       tfjson.ManagedResourceMode,
			Address:    cfg.Address,
			Parent:     cfg.Parent,
			Parameters: processInputs(mc),
		}
	} else {
		// this is a root module
		prefix = ""
	}

	// process resources
	for _, r := range mc.Module.Resources {
		absoluteAddress := prefix + r.Address

		g.Nodes[absoluteAddress] = Node{
			Name:       r.Name,
			Type:       r.Type,
			Mode:       r.Mode,
			Address:    absoluteAddress,
			Parent:     cfg.Address,
			Parameters: processParams(r),
		}

		// implicit resource deps
		g.addEdgesForResourceExpressions(absoluteAddress, prefix, r.Expressions)

		// explicit resource deps
		for _, dep := range r.DependsOn {
			g.AddEdges(Edge{
				From: prefix + dep,
				To:   absoluteAddress,
			})
		}
	}

	// process module deps
	for varname, _ := range mc.Module.Variables {
		absoluteAddress := prefix + "var." + varname
		g.Nodes[absoluteAddress] = Node{
			Name:       varname,
			Type:       "variable",
			Mode:       tfjson.ManagedResourceMode,
			Address:    absoluteAddress,
			Parent:     cfg.Address,
			Parameters: []string{},
		}
	}

	g.addEdgesForModuleVars(cfg.Address, cfg.Parent, mc.Expressions)
	g.addModuleOutputs(cfg.Address, mc.Module.Outputs)

	// recurse into child modules
	for name, m := range mc.Module.ModuleCalls {
		moduleAbsoluteAddress := prefix + "module." + name

		g.WalkModule(m, ModuleConfig{
			Name:    name,
			Address: moduleAbsoluteAddress,
			Parent:  cfg.Address,
		})
	}
}

func (g *Graph) ChangesFrom(p *tfjson.Plan) {
	for _, c := range p.ResourceChanges {
		entry, ok := g.Nodes[c.Address]
		// TODO infer edges from past when deleting
		if !ok && c.Change.Actions.Delete() {
			entry = guessDeletedNode(c)
		}

		// TODO handle moving with PreviousAddress
		entry.Changes = c.Change
		g.Nodes[c.Address] = entry
	}
}

func (g *Graph) AddEdges(edges ...Edge) {
	for _, e := range edges {
		key := edgeKey{from: e.From, to: e.To}
		if _, ok := g.seen[key]; !ok {
			g.seen[key] = e
			g.Edges = append(g.Edges, e)
		}
	}
}

func (g *Graph) addEdgesForModuleVars(module string, parent string, expressions map[string]*tfjson.Expression) {
	var prefix string
	if parent != "" {
		prefix = parent + "."
	}

	walkExpressions(expressions, func(to, from string) {
		g.AddEdges(Edge{
			From: prefix + from,
			To:   module + ".var." + to,
		})
	})
}

func (g *Graph) addEdgesForResourceExpressions(resource string, prefix string, expressions map[string]*tfjson.Expression) {
	walkExpressions(expressions, func(_, from string) {
		g.AddEdges(Edge{
			From: prefix + from,
			To:   resource,
		})
	})
}

func (g *Graph) addModuleOutputs(module string, outputs map[string]*tfjson.ConfigOutput) {
	prefix := module
	if module != "" {
		prefix += "."
	}

	for name, o := range outputs {

		fullname := prefix + name

		g.Nodes[fullname] = Node{
			Name:       name,
			Type:       "output",
			Mode:       tfjson.ManagedResourceMode,
			Address:    fullname,
			Parent:     module,
			Parameters: []string{},
		}

		expressions := make(map[string]*tfjson.Expression)
		expressions[name] = o.Expression

		walkExpressions(expressions, func(_, from string) {
			g.AddEdges(Edge{
				From: prefix + from,
				To:   fullname,
			})
		})
	}
}

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
			walkExpressions(nested, handleReference)
		}
	}
}

func guessDeletedNode(c *tfjson.ResourceChange) Node {
	// TODO test for modules
	fromChange := Node{
		Name:    c.Name,
		Type:    c.Type,
		Mode:    c.Mode,
		Address: c.Address,
	}

	changes, ok := c.Change.Before.(map[string]any)
	if !ok {
		panic("I am bad person")
	}

	parameters := make([]string, 0, len(changes))
	for key, _ := range changes {
		parameters = append(parameters, key)
	}

	fromChange.Parameters = parameters

	parent, err := parentFromString(fromChange.Address, fromChange.Type, fromChange.Name)
	if err != nil {
		panic("Deleted node suffix is messed up completely")
	}

	fromChange.Parent = parent

	return fromChange
}

func parentFromString(addr, t, name string) (string, error) {
	suffix := t + "." + name

	idx := strings.LastIndex(addr, suffix)
	if idx == -1 {
		return "", fmt.Errorf("Could not get parent from address: %s, type,name: %s,%s", addr, t, name)
	}

	parent := addr[:idx]

	return strings.TrimSuffix(parent, "."), nil
}

func processParams(r *tfjson.ConfigResource) []string {
	parameters := make([]string, 0, len(r.Expressions))
	for key, _ := range r.Expressions {
		parameters = append(parameters, key)
	}
	return parameters
}

func processInputs(m *tfjson.ModuleCall) []string {
	parameters := make([]string, 0, len(m.Expressions))
	for key, _ := range m.Expressions {
		parameters = append(parameters, key)
	}
	return parameters
}

func New(r io.Reader) (*Graph, error) {
	var plan tfjson.Plan

	decoder := json.NewDecoder(r)
	if err := decoder.Decode(&plan); err != nil {
		return nil, err
	}

	root := &tfjson.ModuleCall{
		Module: plan.Config.RootModule,
	}

	var graph = Graph{
		Nodes: make(map[string]Node, 100),
		Edges: make([]Edge, 0, 100),
		seen:  make(map[edgeKey]Edge, 100),
	}

	graph.WalkModule(root, ModuleConfig{})
	graph.ChangesFrom(&plan)

	return &graph, nil
}
