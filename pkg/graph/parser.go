package graph

import (
	"encoding/json"
	tfjson "github.com/hashicorp/terraform-json"
	"io"
)

type Graph struct {
	Nodes map[string]Node `json:"nodes"`
	Edges []Edge          `json:"edges"`
}

type Node struct {
	Name       string              `json:"name"`
	Type       string              `json:"type"`
	Mode       tfjson.ResourceMode `json:"mode"`
	Parameters []string            `json:"parameters"`
	Changes    *tfjson.Change      `json:"changes"`
}

type Edge struct {
	From string `json:"from"`
	To   string `json:"to"`
}

func generateNodes(p *tfjson.Plan) map[string]Node {
	root := p.Config.RootModule
	nodes := make(map[string]Node, 40)

	for _, r := range root.Resources {
		parameters := make([]string, 0, len(r.Expressions))
		for key, _ := range r.Expressions {
			parameters = append(parameters, key)
		}

		nodes[r.Address] = Node{
			Name:       r.Name,
			Type:       r.Type,
			Mode:       r.Mode,
			Parameters: parameters,
		}
	}

	for name, m := range root.ModuleCalls {
		parameters := make([]string, 0, len(m.Expressions))
		for key, _ := range m.Expressions {
			parameters = append(parameters, key)
		}

		moduleAddress := "module." + name

		// TODO handle nested modules
		nodes[moduleAddress] = Node{
			Name:       name,
			Type:       "module",
			Mode:       tfjson.ManagedResourceMode,
			Parameters: parameters,
		}

		for _, r := range m.Module.Resources {
			parameters := make([]string, 0, len(r.Expressions))
			for key, _ := range r.Expressions {
				parameters = append(parameters, key)
			}

			nodes[moduleAddress+"."+r.Address] = Node{
				Name:       r.Name,
				Type:       r.Type,
				Mode:       r.Mode,
				Parameters: parameters,
			}
		}
	}

	for _, c := range p.ResourceChanges {
		// TODO handle moving with PreviousAddress
		if entry, ok := nodes[c.Address]; ok {
			entry.Changes = c.Change

			nodes[c.Address] = entry
		}
	}

	return nodes
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
			edges = append(edges, processExpressions(moduleAddress+"."+r.Address, r.Expressions)...)
		}
	}

	return edges
}

func processExpressions(resource string, expressions map[string]*tfjson.Expression) []Edge {
	edges := make([]Edge, 0, 40)

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
