package graph

import (
	"encoding/json"
	"io"

	tfjson "github.com/hashicorp/terraform-json"
)

type Graph struct {
	Nodes map[string]Node `json:"nodes"`
	Edges []Edge          `json:"edges"`
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

func (g *Graph) NodesFrom(mc *tfjson.ModuleCall, cfg ModuleConfig) {
	// TODO variables, outputs as nodes, especially for child modules
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

		// implicit deps
		g.Edges = append(g.Edges, processExpressions(absoluteAddress, r.Expressions)...)

		// explicit deps
		for _, dep := range r.DependsOn {
			g.Edges = append(g.Edges, Edge{
				From: prefix + dep,
				To:   absoluteAddress,
			})
		}
	}

	// recurse into child modules
	for name, m := range mc.Module.ModuleCalls {
		moduleAbsoluteAddress := prefix + "module." + name

		// g.Edges = append(g.Edges, processExpressions(moduleAbsoluteAddress, m.Expressions)...)

		g.NodesFrom(m, ModuleConfig{
			Name:    name,
			Address: moduleAbsoluteAddress,
			Parent:  cfg.Address,
		})
	}
}

func (g *Graph) ChangesFrom(p *tfjson.Plan) {
	for _, c := range p.ResourceChanges {
		// TODO handle moving with PreviousAddress
		if entry, ok := g.Nodes[c.Address]; ok {
			entry.Changes = c.Change

			g.Nodes[c.Address] = entry
		}
	}
}

// Process module(edges):
// process resources(process expressions)
// process module calls(process expressions)

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

func processExpressions(resource string, expressions map[string]*tfjson.Expression) []Edge {
	edges := make([]Edge, 0, 10)

	for _, e := range expressions {
		if e.ConstantValue != tfjson.UnknownConstantValue {
			// this is just static value
			continue
		}

		for _, dep := range e.References {
			edges = append(edges, Edge{
				From: dep,
				To:   resource,
			})
		}

		for _, nested := range e.NestedBlocks {
			edges = append(edges, processExpressions(resource, nested)...)
		}
	}

	return edges
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

	var g = Graph{
		Nodes: make(map[string]Node, 100),
		Edges: make([]Edge, 100),
	}

	g.NodesFrom(root, ModuleConfig{})
	g.ChangesFrom(&plan)
	return &g, nil
}
