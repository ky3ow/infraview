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
	seen  map[Edge]struct{}
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
	From string `json:"from"`
	To   string `json:"to"`
}

type ModuleConfig struct {
	Name    string
	Address string
	Parent  string
}

func (g *Graph) WalkModule(mc *tfjson.ModuleCall, cfg ModuleConfig) {
	// TODO variables, outputs as nodes, especially for child modules
	// TODO count, for_each(parent nodes from config, individual in changes)
	// TODO clean references(dots should be an even number to be not attributes?)
	// TODO check secrets
	// TODO dynamic
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
		g.addEdgesForExpressions(absoluteAddress, prefix, r.Expressions)

		// explicit resource deps
		for _, dep := range r.DependsOn {
			g.AddEdges(Edge{
				From: prefix + dep,
				To:   absoluteAddress,
			})
		}
	}

	// process module deps
	g.addEdgesForModuleVars(cfg.Address, cfg.Parent, mc.Expressions)

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
		if _, ok := g.seen[e]; !ok {
			g.seen[e] = struct{}{}
			g.Edges = append(g.Edges, e)
		}
	}
}

func (g *Graph) addEdgesForExpressions(resource string, prefix string, expressions map[string]*tfjson.Expression) {
	for _, e := range expressions {
		if e.ConstantValue != tfjson.UnknownConstantValue {
			// this is just static value
			continue
		}

		for _, ref := range e.References {
			g.AddEdges(Edge{
				From: prefix + ref,
				To:   resource,
			})
		}

		for _, nested := range e.NestedBlocks {
			g.addEdgesForExpressions(resource, prefix, nested)
		}
	}
}

func (g *Graph) addEdgesForModuleVars(module string, parent string, expressions map[string]*tfjson.Expression) {
	var prefix string
	if parent != "" {
		prefix = parent + "."
	}
	for input_varname, e := range expressions {
		if e.ConstantValue != tfjson.UnknownConstantValue {
			// this is just static value
			continue
		}

		absolute_varname := module + ".var." + input_varname

		for _, ref := range e.References {
			g.AddEdges(Edge{
				From: prefix + ref,
				To:   absolute_varname,
			})
		}

		for _, nested := range e.NestedBlocks {
			g.addEdgesForModuleVars(module, parent, nested)
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
		seen: make(map[Edge]struct{}, 100),
	}

	graph.WalkModule(root, ModuleConfig{})
	graph.ChangesFrom(&plan)

	return &graph, nil
}
