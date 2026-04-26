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
}

type NodeRegistry map[string]Node

func (nr NodeRegistry) PopulateNodes(mc *tfjson.ModuleCall, cfg ModuleConfig) {
	var prefix string

	if cfg.Address != "" {
		// case for child modules
		prefix = cfg.Address + "."
		nr[cfg.Address] = Node{
			Name:       cfg.Name,
			Type:       "module",
			Mode:       tfjson.ManagedResourceMode,
			Address:    cfg.Address,
			Parameters: processInputs(mc),
		}
	} else {
		// this is a root module
		prefix = ""
	}

	// process resources
	for _, r := range mc.Module.Resources {
		absoluteAddress := prefix + r.Address

		nr[absoluteAddress] = Node{
			Name:       r.Name,
			Type:       r.Type,
			Mode:       r.Mode,
			Address:    absoluteAddress,
			Parameters: processParams(r),
		}
	}

	// recurse into child modules
	for name, m := range mc.Module.ModuleCalls {
		nr.PopulateNodes(m, ModuleConfig{
			Name:    name,
			Address: prefix + "module." + name,
		})
	}
	
}

// Process module(nodes):
// process variables
// process resources
// process outputs
// recurse(process module calls)

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

func generateNodes(p *tfjson.Plan) map[string]Node {
	root := &tfjson.ModuleCall{
		Module: p.Config.RootModule,
	}
	registry := make(NodeRegistry, 100)
	registry.PopulateNodes(root, ModuleConfig{})

	for _, c := range p.ResourceChanges {
		// TODO handle moving with PreviousAddress
		if entry, ok := registry[c.Address]; ok {
			entry.Changes = c.Change

			registry[c.Address] = entry
		}
	}

	return registry
}

func generateEdges(p *tfjson.Plan) []Edge {
	root := p.Config.RootModule

	edges := make([]Edge, 0, 40)

	for _, r := range root.Resources {
		edges = append(edges, processExpressions(r.Address, r.Expressions)...)

		for _, dep := range r.DependsOn {
			edges = append(edges, Edge{
				From: dep,
				To:   r.Address,
			})
		}
	}

	// TODO handle nested modules
	for name, m := range root.ModuleCalls {
		moduleAddress := "module." + name

		edges = append(edges, processExpressions(moduleAddress, m.Expressions)...)

		for _, r := range m.Module.Resources {
			resourceAddress := moduleAddress + "." + r.Address
			edges = append(edges, processExpressions(resourceAddress, r.Expressions)...)

			edges = append(edges, Edge{
				From: resourceAddress,
				To:   moduleAddress,
			})
		}
	}

	return edges
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

	nodes := generateNodes(&plan)

	edges := generateEdges(&plan)

	var graph = Graph{
		Nodes: nodes,
		Edges: edges,
	}

	return &graph, nil
}
