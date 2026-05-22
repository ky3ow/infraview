# How to run

```bash
go mod tidy
go run ./cmd/infraview-server # starts web server at localhost:8080
```

# Main functionality

- Http server for interactive graph +
- See graph of your resources and variables, with connections between and dependencies +
- Highlight resource operations(create,update,delete) +
- Filtering +
- Grouping +
- Resource details, what triggered the change, old/new values +-
- Plan history, separated by "namespaces"(e.g. you create namespace per git branch) (coming)
- CI/CD integration via REST api calls(possibly cli tool as project grows) (coming)

## Nice to have

- User rules for custom grouping
- Addons(e.g. infracost annotations, probably just client side visual things from data supplied with plan?)

## Non-goals

Any sort of actions in Core, like applies or approvals, can be partially solved with addons. 

Example: show the pending approval in UI, and on click redirect user to actual approval url(e.g. github actions)

# Test coverage

go test ./test/graph/ -cover -v -coverpkg=./pkg/graph

# Checklist

- [x] Treat special keywords while generating graph
  - [ ] local. ? < not planned
  - [x] var. (edge)
  - [x] module. (edge + Parent thingy)
  - [x] each. (edge + parent? individual nodes are only from Planned values, references and parents should be in configuration) 
  - [x] count. (edge + parent? individual nodes are only from Planned values, references and parents should be in configuration) 
- [x] module outputs
- [x] deletion - get node from `prior_state`

# Insiprations

- [rover](https://github.com/im2nguyen/rover)
- [terraform-visual](https://github.com/hieven/terraform-visual)
- [pluralith](https://github.com/Pluralith/pluralith-cli)

# Links

- [terraform json output](https://developer.hashicorp.com/terraform/internals/json-format)
- [go helper library](https://github.com/hashicorp/terraform-json)
- [some article](https://getcoal.medium.com/golang-handling-terraform-files-a37371c621e3)
- [mock terraform provider used for initial testing](https://github.com/hashicorp/terraform-provider-tfcoremock/tree/main)
- [layout](https://github.com/golang-standards/project-layout?tab=readme-ov-file#go-directories)
- [Cytoscape.js, library for graph visualization](https://js.cytoscape.org)
- [cytoscape-klay, another algorithm for cytoscape]()
- [another graph lib](https://www.sigmajs.org)

# Terraform json representation

```json
{
  "format_version": "1.0",

  // "prior_state" is a representation of the state that the configuration is
  // being applied to, using the state representation described above.
  "prior_state":  <state-representation>,

  // "applyable" indicates that it would make sense for a wrapping automation
  // to try to apply this plan, possibly after asking a human operator for
  // approval.
  "applyable": true,

  // "complete" indicates that Terraform expects that after applying this
  // plan the actual state will match the desired state.
  "complete": true,

  // "errored" indicates whether planning failed. An errored plan cannot be applied,
  // but the actions planned before failure may help to understand the error.
  "errored": false,

  // "configuration" is a representation of the configuration being applied to the
  // prior state, using the configuration representation described above.
  "configuration": <configuration-representation>,

  // "planned_values" is a description of what is known so far of the outcome in
  // the standard value representation, with any as-yet-unknown values omitted.
  "planned_values": <values-representation>,

  // "proposed_unknown" is a representation of the attributes, including any
  // potentially-unknown attributes. Each value is replaced with "true" or
  // "false" depending on whether it is known in the proposed plan.
  "proposed_unknown": <values-representation>,

  // "variables" is a representation of all the variables provided for the given
  // plan. This is structured as a map similar to the output map so we can add
  // additional fields in later.
  "variables": {
    "varname": {
      "value": "varvalue"
    },
  },

  // "resource_changes" is a description of the individual change actions that
  // Terraform plans to use to move from the prior state to a new state
  // matching the configuration.
  "resource_changes": [
    // Each element of this array describes the action to take
    // for one instance object. All resources in the
    // configuration are included in this list.
    {
      // "address" is the full absolute address of the resource instance this
      // change applies to, in the same format as addresses in a value
      // representation.
      "address": "module.child.aws_instance.foo[0]",

      // "previous_address" is the full absolute address of this resource
      // instance as it was known after the previous Terraform run.
      // Included only if the address has changed, e.g. by handling
      // a "moved" block in the configuration.
      "previous_address": "module.instances.aws_instance.foo[0]",

      // "module_address", if set, is the module portion of the above address.
      // Omitted if the instance is in the root module.
      "module_address": "module.child",

      // "mode", "type", "name", and "index" have the same meaning as in a
      // value representation.
      "mode": "managed",
      "type": "aws_instance",
      "name": "foo",
      "index": 0,

      // "deposed", if set, indicates that this action applies to a "deposed"
      // object of the given instance rather than to its "current" object.
      // Omitted for changes to the current object. "address" and "deposed"
      // together form a unique key across all change objects in a particular
      // plan. The value is an opaque key representing the specific deposed
      // object.
      "deposed": "deadbeef",

      // "change" describes the change that will be made to the indicated
      // object. The <change-representation> is detailed in a section below.
      "change": <change-representation>,

      // "action_reason" is some optional extra context about why the
      // actions given inside "change" were selected. This is the JSON
      // equivalent of annotations shown in the normal plan output like
      // "is tainted, so must be replaced" as opposed to just "must be
      // replaced".
      //
      // These reason codes are display hints only and the set of possible
      // hints may change over time. Users of this must be prepared to
      // encounter unrecognized reasons and treat them as unspecified reasons.
      //
      // The current set of possible values is:
      // - "replace_because_tainted": the object in question is marked as
      //   "tainted" in the prior state, so Terraform planned to replace it.
      // - "replace_because_cannot_update": the provider indicated that one
      //   of the requested changes isn't possible without replacing the
      //   existing object with a new object.
      // - "replace_by_request": the user explicitly called for this object
      //   to be replaced as an option when creating the plan, which therefore
      //   overrode what would have been a "no-op" or "update" action otherwise.
      // - "delete_because_no_resource_config": Terraform found no resource
      //   configuration corresponding to this instance.
      // - "delete_because_no_module": The resource instance belongs to a
      //   module instance that's no longer declared, perhaps due to changing
      //   the "count" or "for_each" argument on one of the containing modules.
      // - "delete_because_wrong_repetition": The instance key portion of the
      //   resource address isn't of a suitable type for the corresponding
      //   resource's configured repetition mode (count, for_each, or neither).
      // - "delete_because_count_index": The corresponding resource uses count,
      //   but the instance key is out of range for the currently-configured
      //   count value.
      // - "delete_because_each_key": The corresponding resource uses for_each,
      //   but the instance key doesn't match any of the keys in the
      //   currently-configured for_each value.
      // - "read_because_config_unknown": For a data resource, Terraform cannot
      //   read the data during the plan phase because of values in the
      //   configuration that won't be known until the apply phase.
      // - "read_because_dependency_pending": For a data resource, Terraform
      //   cannot read the data during the plan phase because the data
      //   resource depends on at least one managed resource that also has
      //   a pending change in the same plan.
      //
      // If there is no special reason to note, Terraform will omit this
      // property altogether.
      "action_reason": "replace_because_tainted"
    }
  ],

  // "resource_drift" is a description of the changes Terraform detected
  // when it compared the most recent state to the prior saved state.
  "resource_drift": [
    {
        // "resource_drift" uses the same object structure as
        // "resource_changes".
    }
  ],

  // "relevant_attributes" lists the sources of all values contributing to
  // changes in the plan. You can use "relevant_attributes" to filter
  // "resource_drift" and determine which external changes may have affected the
  // plan result.
  "relevant_attributes": [
    {
      "resource": "aws_instance.foo",
      "attribute": "attr",
    }
  ]

  // "output_changes" describes the planned changes to the output values of the
  // root module.
  "output_changes": {
    // Keys are the defined output value names.
    "foo": {

      // "change" describes the change that will be made to the indicated output
      // value, using the same representation as for resource changes except
      // that the only valid actions values are:
      //   ["create"]
      //   ["update"]
      //   ["delete"]
      // In the Terraform CLI 0.12.0 release, Terraform is not yet fully able to
      // track changes to output values, so the actions indicated may not be
      // fully accurate, but the "after" value will always be correct.
      "change": <change-representation>,
    }
  },

  // "checks" describes the partial results for any checkable objects, such as
  // resources with postconditions, with as much information as Terraform can
  // recognize at plan time. Some objects will have status "unknown" to
  // indicate that their status will only be determined after applying the plan.
  "checks" <checks-representation>
}
```


## Food for thought

### Need IR?

Pre-process config, changes and prior_state in something simpler

### Configuration

Has `root_module` that contains array of resources
Has `module_calls`, a map of "name -> metadata", contains source, expressions(call parameters), module field(map of outputs, array of resources)
Has `resource_drift` which has resource changes that also occured because of DRIFTS

#### Resource

- address
- mode(managed, or data)
- type(resource kind, e.g. s3 bucket)
- name
- provider config key
- expressions array
- depends_on array

#### Expressions

Map of Name -> value

value can be:

- object with key `constant_value`, which has a constant value
- object with key `references`, which is array, defines dependencies
- array of Expressions(map Name -> value)

If a value, or block has a reference on other dependency - WHOLE value or block is replaced with `references` in the configuration section
