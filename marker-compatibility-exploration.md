# Notes
elementRelation: associative (lists only), atomic (lists/maps), separable (maps only)

from no listType specification -- to atomic

# Plan
all combinations of updates to listType: atomic, set, map
all combinations of updates to mapType: atomic, granular

Dimensions:
- listType/mapType -- all combinations
- operation type -- apply vs patch

# Server side apply from the start

## x-kubernetes-map-type
1. `granular` refers to the map, not the fields. The fields are atomic in behaviour; they can only ever be managed by a single manager.

To show in practice:

With this configuration (crd.yml)
```
properties:
  colour:
    type: object
    additionalProperties:
    type: string
    x-kubernetes-map-type: granular
```

And this spec (object.yml)
```
spec:
  colour:
    name:   turquoise
    hue:    light
    saturation: strong
```

`k apply -f object-map-blues.yml --server-side=true --field-manager=first` makes `first` the manager of name, hue, saturation (but not the map)

with this similar spec
And this spec (object2.yml)
```
spec:
  colour:
    name:   turquoise
    hue:    light
    saturation: different
```
`k apply -f object-map-blues.yml --server-side=true --field-manager=second` will fail with conflict.

However if instead `first` and `second` apply the following specs, correspondingly:

```
spec:
  colour:
    name:   turquoise
    hue:    light
```

```
spec:
  colour:
    saturation: opaque
```

Then we end up with:

```
apiVersion: colours.example.com/v1
kind: ColourMap
metadata:
  creationTimestamp: "2020-01-09T13:00:59Z"
  generation: 2
  managedFields:
  - apiVersion: colours.example.com/v1
    fieldsType: FieldsV1
    fieldsV1:
      f:spec:
        f:colour:
          f:hue: {}
          f:name: {}
    manager: first
    operation: Apply
    time: "2020-01-09T13:00:59Z"
  - apiVersion: colours.example.com/v1
    fieldsType: FieldsV1
    fieldsV1:
      f:spec:
        f:colour:
          f:saturation: {}
    manager: second
    operation: Apply
    time: "2020-01-09T13:01:18Z"
  name: blue-map
  namespace: default
  resourceVersion: "4106"
  selfLink: /apis/colours.example.com/v1/namespaces/default/colourmaps/blue-map
  uid: b7daa416-c549-4a29-9ab2-58122654e1f8
spec:
  colour:
    hue: strong
    name: turquoise
    saturation: opaque
```

## x-kubernetes-list-type (and potentially x-list-map-keys)
### Updates between lists of scalars (set <-> atomic)

1. When moving: Atomic -> set => should never be a problem

1. Set -> atomic should be fine if single manager, should fail otherwise? => that was my assumption, it's a bit more intricate in practice.
Given a `x-kubernetes-list-type: set` with multiple managers on a single field:
- When updating `x-kubernetes-list-type: set-> atomic` in crd and issueing `k apply -f crd.yml` --> success and no changes in the objects.
- When applying the (pre-existing) custom objects:
`k apply <object with changes> --field-manager=<anybody>` --> succeeds (even when touching fields that other managers previously set/managed), and replaces the object with the values provided. I.e. "overrules" other managers.
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

### Lists of scalars <-> map (set <-> map and atomic <-> map)
1. Given atomic sets, when I update the CRD list field to `map`:
* Existing custom objects can no longer be updated, regardless of manager.
```
Error from server: failed to create typed live object: errors:
  .spec.colours: element 0: associative list with keys may not have non-map elements
  .spec.colours: element 1: associative list with keys may not have non-map elements
```
Maybe that's because: how could the server possibly know how to change the scalar into a map?
-[ ] Perhaps it can be possible with condition: there's only one map key, and all other fields in the map have defaults.

* New objects can be created fine.
  - The resulting list of nested objects is a set (uniqueness calculated using map keys).
  - The map entries of the list are themselves granular (I can update a field in the entry without having to update all other fields, or replace the entire map).
  - The top level list is also granular: multiple managers can update different entries independently.
  - Management of the fields is inherited when someone assumes management of the entry. In other words, whoever manages entry [3] also manages _all_ of its fields.

~However: say I put a default for one of the fields --> it works fine. Defaults the field in new objects as expected.
When I change the default, the new default is not respected. The old one continues to be used.~ => not true, I probably hadn't saved/applied the CRD as I thought.
e.g. with a CRD:

```
...
hue:
  type: string
  default: "medium"
```
when applying
```
...
- name: straw
```
I still get:
```
  ...
  - hue: light
    name: straw
```
(`light` was the previous default).
~-[ ] Either a bug or something to document, if it's intentional (`you can only set defaults once`).~

1. Given a map, when I try to update to set/atomic with scalar types, I'll get an error at object apply time:
```
Error from server: failed to create typed live object: errors:
  .spec.colours: element 0: associative list without keys has an element that's a map type
  .spec.colours: element 1: associative list without keys has an element that's a map type
  (etc. for all the fields)
```
Just applying the updated CRD (that sets `x-kubernetes-list-type: set` or `x-kubernetes-list-type: atomic` - tried both - where it used to be `x-kubernetes-list-type: map`) also empties the contents (but keeps the element type=map) of the object spec:
```
...
spec:
  colours:
  - {}
  - {}
  - {}
  - {}
  - {}
  - {}
  - {}
  - {}
  - {}
```

### Updating the map keys
* Failures if spec to be applied contains entries that duplicate or omit the new key-as expected.
* ...but also if the spec of the existing object contains duplicates of the old key, or entries that omit it. If it's in that state, the object can't be updated as is.
-[ ] So probably a good idea to default from the beginning (to avoid omitted fields).
-[ ] Alternative/Migration strategy: have an intermediate CRD where map keys contain the combination of all[1] keys, apply a more "explicit" spec (i.e. define exact values you want to see appear) and then update to the CRD with the keys and defaults you want to have going forward.
[1] Not necessarily all, a combination that would allow for all entries to be appear "unique", and contains the keys you eventually want to use is good enough.

## x-map-type
TODO

## x-struct-type
TODO

# Existing non-server-side applied objects, moving to ssa
## x-list-type
### From non-SSA to SSA
Without SSA, lists behave as atomic (`x-list-type` is not considered even if set, in case you were wondering).


Regardless of what `x-list-type` gets set to:
The whole list is managed by `before-first-apply` (through `Update`):
```
- apiVersion: colours.example.com/v1
    fieldsType: FieldsV1
    fieldsV1:
      f:metadata:
        f:annotations:
          f:kubectl.kubernetes.io/last-applied-configuration: {}
      f:spec:
        f:colours: {}
    manager: before-first-apply
    operation: Update
```

In many ways, what happens next is similar to updates that have an existing atomic list as their starting point.

Depending on what `x-list-type` gets configured to:
* If `x-list-type: atomic` => The _only_ SSA you can run is to claim co-ownership of the _whole_ list by applying the existing values exactly (no more).
Then you become a co-owner of the list, together with `before-first-apply`. You can't (ever, while the list is atomic) apply anything else except preexisting values-you'll get a conflict otherwise.
  * (also see below. `From SSA to non-SSA`) You can atomically replace the list, if you don't use SSA though (`k apply -f object.yml`).
  ```
  managedFields:
  - apiVersion: colours.example.com/v1
    fieldsType: FieldsV1
    fieldsV1:
      f:metadata:
        f:annotations:
          f:kubectl.kubernetes.io/last-applied-configuration: {}
      f:spec:
        f:colours: {}
    manager: kubectl
    operation: Update
    time: "2019-12-23T16:11:34Z"
  ```
* If `x-list-type: set` => you can directly apply new fields.
  * Preexisting will show up as managed by `before-first-apply`, new will be managed by the new managers.
  * You will co-manage preexisting ones that you apply, as expected.

### From SSA to non-SSA
The list will be replaced atomically when a non-SSA gets issued. Interestingly this operation _will_ update the object's managed fields, and make `kubectl` the only manager entry (even if you try to pass another through `--field-manager`). `kubectl` manages each of the fields, _not_ the list as a whole.
```
managedFields:
  - apiVersion: colours.example.com/v1
    fieldsType: FieldsV1
    fieldsV1:
      f:metadata:
        f:annotations:
          f:kubectl.kubernetes.io/last-applied-configuration: {}
      f:spec:
        f:colours:
          v:"emerald": {}
          v:"lime": {}
          v:"olive": {}
    manager: kubectl
    operation: Update
    time: "2019-12-23T16:25:49Z"
```

### x-list-type: map
Very similar to above - you start with a map, whose initial entries are managed by `before-first-apply` with `operation: Update`, once you switch to SSA.

Equally the SSA -> non-SSA path will leave you with `kubectl` via `Update` managing all the fields.

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
