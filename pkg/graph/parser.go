// TODO count, for_each(parent nodes from config, individual in changes)
// TODO edge attributes to have granular dependency shown
// TODO check secrets
// TODO dynamic
package graph

import (
	"encoding/json"
	"errors"
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

func (g *Graph) discoverPresentNodes(mc *tfjson.ModuleCall, cfg ModuleConfig) {
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
	}

	// process module variables
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

	// process module outputs
	for name := range mc.Module.Outputs {

		fullname := prefix + name

		g.Nodes[fullname] = Node{
			Name:       name,
			Type:       "output",
			Mode:       tfjson.ManagedResourceMode,
			Address:    fullname,
			Parent:     cfg.Address,
			Parameters: []string{},
		}
	}

	// recurse into child modules
	for name, m := range mc.Module.ModuleCalls {
		moduleAbsoluteAddress := prefix + "module." + name

		g.discoverPresentNodes(m, ModuleConfig{
			Name:    name,
			Address: moduleAbsoluteAddress,
			Parent:  cfg.Address,
		})
	}
}

func (g *Graph) discoverPriorNodes(p *tfjson.Plan) {
	for _, c := range p.ResourceChanges {
		if c.Change.Actions.Delete() {
			g.Nodes[c.Address] = Node{
				Name:    c.Name,
				Type:    c.Type,
				Mode:    c.Mode,
				Address: c.Address,
			}
		}
	}
}

func (g *Graph) discoverEdges(mc *tfjson.ModuleCall, cfg ModuleConfig) {
	var prefix string

	if cfg.Address != "" {
		// case for child modules
		prefix = cfg.Address + "."
	} else {
		// this is a root module
		prefix = ""
	}

	for _, r := range mc.Module.Resources {
		absoluteAddress := prefix + r.Address

		// implicit dependencies
		walkExpressions(r.Expressions, func(_, from string) {
			g.AddEdge(Edge{
				From: prefix + from,
				To:   absoluteAddress,
			})
		})

		// explicit dependencies
		for _, dep := range r.DependsOn {
			g.AddEdge(Edge{
				From: prefix + dep,
				To:   absoluteAddress,
			})
		}
	}

	// process module variables
	// Expressions are nil for root module
	parentPrefix := cfg.Parent
	if parentPrefix != "" {
		parentPrefix += "."
	}

	walkExpressions(mc.Expressions, func(to, from string) {
		g.AddEdge(Edge{
			From: parentPrefix + from,
			To:   prefix + "var." + to,
		})
	})

	for name, o := range mc.Module.Outputs {
		fullname := prefix + name

		expressions := make(map[string]*tfjson.Expression)
		expressions[name] = o.Expression // TODO meh

		walkExpressions(expressions, func(_, from string) {
			g.AddEdge(Edge{
				From: prefix + from,
				To:   fullname,
			})
		})
	}

	// recurse into child modules
	for name, m := range mc.Module.ModuleCalls {
		moduleAbsoluteAddress := prefix + "module." + name

		g.discoverEdges(m, ModuleConfig{
			Name:    name,
			Address: moduleAbsoluteAddress,
			Parent:  cfg.Address,
		})
	}
}

func (g *Graph) changesFrom(p *tfjson.Plan) error {
	var errs []error

	for _, c := range p.ResourceChanges {
		node, ok := g.Nodes[c.Address]

		if !ok {
			errs = append(errs, fmt.Errorf("Node %v not found", c.Address))
		}

		// TODO handle moving with PreviousAddress
		node.Changes = c.Change
		g.Nodes[c.Address] = node
	}

	return errors.Join(errs...)
}

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

	graph.discoverPresentNodes(root, ModuleConfig{})
	graph.discoverPriorNodes(&plan)
	graph.discoverEdges(root, ModuleConfig{})
	graph.changesFrom(&plan)

	return &graph, nil
}
