# Notes
elementRelation: associative (lists only), atomic (lists/maps), separable (maps only)

from no listType specification -- to atomic

# Plan
all combinations of updates to listType: atomic, set, map
all combinations of updates to mapType: atomic, granular

Dimensions:
- listType/mapType -- all combinations
- operation type -- apply vs patch

# Server side apply all the way
## With: list of scalars

1. When moving: Atomic -> set => should never be a problem

1. Set -> atomic should be fine if single manager, should fail otherwise?
Set with multiple managers on a single field:
- update crd to x-list-type: set --> fine
- k apply <object with no changes> --field-manager=<last manager who changed the object> --> succeeds, even without passing --force-conflicts, and replaces the object with the values provided. I.e. drops the other manager.
- k apply <object with changes> --field-manager=<anybody> --> succeeds (even when touching fields that other managers previously set/managed)
It leaves previous entries in `managedFields` intact though. You end up with something like:
```
managedFields:
  - apiVersion: colours.example.com/v1
    fieldsType: FieldsV1
    fieldsV1:
      f:spec:
        f:colours:
          v:"black": {}
    manager: third
    operation: Apply
    time: "2019-12-23T09:52:05Z"
  - apiVersion: colours.example.com/v1
    fieldsType: FieldsV1
    fieldsV1:
      f:spec:
        f:colours:
          v:"red": {}
    manager: fourth
    operation: Apply
    time: "2019-12-23T09:53:12Z"
  - apiVersion: colours.example.com/v1
    fieldsType: FieldsV1
    fieldsV1:
      f:spec:
        f:colours: {}
    manager: fifth
    operation: Apply
    time: "2019-12-23T10:02:50Z"
  name: simple-colour-set
  namespace: default
  resourceVersion: "2631"
  selfLink: /apis/colours.example.com/v1/namespaces/default/associativesets/simple-colour-set
  uid: 27cb3520-763c-43e1-98c6-d5e48b818d40
```
The update of atomic-> set also does that, but has the manager of the entire spec.colours first (the update set-> atomic has the top-level manager last). It seems like there's a dependency on the order of `managedFields` to figure out who manages what.

This extends to even top-level managers; they don't get evicted when a new field manager starts managing the field. You end up with the following in an object:
```
managedFields:
  - apiVersion: colours.example.com/v1
    fieldsType: FieldsV1
    fieldsV1:
      f:spec:
        f:colours:
          v:"black": {}
    manager: third
    operation: Apply
    time: "2019-12-23T09:52:05Z"
  - apiVersion: colours.example.com/v1
    fieldsType: FieldsV1
    fieldsV1:
      f:spec:
        f:colours: {}
    manager: fifth
    operation: Apply
    time: "2019-12-23T10:02:50Z"
  - apiVersion: colours.example.com/v1
    fieldsType: FieldsV1
    fieldsV1:
      f:spec:
        f:colours: {}
    manager: fourth
    operation: Apply
    time: "2019-12-23T10:09:40Z"
```
From that point on, any operations from ~managers other than the last top-level one fail~ any manager, including the most recent top-level one (`fourth` in this case) will fail. Unless of course they apply the existing state (thus adding themselves as co-managers) or they `--force-conflicts`.
See also `More Observations`.

## List, map
1. Given atomic sets, when I update the CRD list field to `map`:
2.1: Repeat 1.1 and 1.2
2.2: When keys change -> ???

If I go from not using server side apply, to using it?


# Existing non-server-side applied objects, moving to ssa


# More observations:
1. The managers stick around in managedFields, even when they're no longer around. I.e. when the fields that they manage are "gone", or when the list is atomic and a different manager has claimed ownership. Is that intentional? What's the use case?
e.g. even after `first` was not managing anything, because `red` was completely gone:
```
managedFields:
  - apiVersion: colours.example.com/v1
    fieldsType: FieldsV1
    fieldsV1:
      f:spec:
        f:colours:
          v:"red": {}
    manager: first
    operation: Apply
    time: "2019-12-20T15:26:29Z"
```

1. Sending empty contents from list removes the manager and does NOT empty (or in other ways modify) the list.
This can mean that there might be fields with no manager. E.g. if `first` applied `blue`, and then `first` applies `[]` --> blue stays around but no longer shows up under `managedFields`.
