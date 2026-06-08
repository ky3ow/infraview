package graph

import (
	"errors"
	"fmt"

	tfjson "github.com/hashicorp/terraform-json"
	"github.com/ky3ow/infraview/pkg/log"
)

var logger = log.New("graph")

// Enhances *existing* resource nodes with their state
// and changes in form of json objects
func (g *Graph) discoverResourceChanges(p *tfjson.Plan) error {
	var errs []error

	for _, c := range p.ResourceChanges {
		node, ok := g.Nodes[c.Address]

		if !ok {
			logger.Warn("resource change has no matching node", "address", c.Address)
			errs = append(errs, fmt.Errorf("node %v not found", c.Address))
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
		changes, err := stripChange(c.Change)

		if err != nil {
			return err
		}

		node.Changes = changes

		g.Nodes[c.Address] = node
	}

	return errors.Join(errs...)
}

func (g *Graph) discoverParentState() {
	for addr, parent := range g.Nodes {
		if parent.Kind != ModuleNode && parent.Kind != LoopResourceNode {
			continue
		}

		var parentState ResourceState
		allSame := true
		for _, child := range g.Nodes {
			if child.Parent == parent.Address {
				// this needs knowledge that all children are resolved state, +1 to data reform TODO
				if child.Kind != ModuleNode && child.Kind != LoopResourceNode && child.Kind != ResourceNode {
					continue
				}

				if parentState == "" {
					parentState = child.State
				} else if parentState != child.State {
					allSame = false
				}
			}
		}

		if !allSame {
			parentState = StateModify
		}

		parent.State = parentState
		g.Nodes[addr] = parent
	}
}

// Returns stripped out of sensitive fields after/before changes.
// Returns error if before/after/-sensitive parts were not converted cleanly
// tfjson gives `any` which is not comfortable to work with.
// TODO unmarshall json with self written thing
func stripChange(c *tfjson.Change) (*NodeChange, error) {
	var stripped NodeChange

	after, err := stripSensitive(c.After, c.AfterSensitive)

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
