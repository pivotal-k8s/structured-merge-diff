elementRelation: associative (lists only), atomic (lists/maps), separable (maps only)

from no listType specification -- to atomic


all combinations of updates to listType: atomic, set, map
all combinations of updates to mapType: atomic, granular

Dimensions:
- listType/mapType -- all combinations
- operation type -- apply vs patch




1. List of scalars
1.1 Atomic -> associative should never be a problem
1.2 Associative -> atomic should be fine if single manager, should fail otherwise
Associative with multiple managers on a single field:
- update crd to x-list-type: associative --> fine
- k apply <object with no changes> --field-manager=<last manager who changed the object> --> succeeds, even without passing --force-conflicts, and replaces the object with the values provided. I.e. drops the other manager.
- k apply <object with changes> --field-manager=<anybody> --> succeeds (even when touching fields that other managers previously set/managed)


If a previous manager tries to apply something they (according to `managedFields`) already manage:
± mn |add-kubebuilder {1} U:2 ?:1 ✗| → k get at/simple-colour-set -o yaml
W1220 15:34:14.956293   58289 loader.go:223] Config not found: /Users/pivotal/.kube/kind-config-1.16.3
apiVersion: colours.example.com/v1
kind: AssociativeSet
metadata:
  creationTimestamp: "2019-12-20T15:17:50Z"
  generation: 6
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
  - apiVersion: colours.example.com/v1
    fieldsType: FieldsV1
    fieldsV1:
      f:spec:
        f:colours: {}
    manager: second
    operation: Apply
    time: "2019-12-20T15:27:47Z"
  - apiVersion: colours.example.com/v1
    fieldsType: FieldsV1
    fieldsV1:
      f:spec:
        f:colours:
          v:"pink": {}
    manager: third
    operation: Apply
    time: "2019-12-20T15:30:01Z"
  name: simple-colour-set
  namespace: default
  resourceVersion: "273302"
  selfLink: /apis/colours.example.com/v1/namespaces/default/associativesets/simple-colour-set
  uid: d58f1075-bb3c-4ad4-95f4-397e45bcd97c
spec:
  colours:
  - orange
  - pink

 2019-12-20 15:34:15 ⌚ ruby 2.4.1p111 piv-ws-vauxhall in ~/workspace/crd-playground/list-map-type
± mn |add-kubebuilder {1} U:2 ?:1 ✗| → k apply -f manifests/object-colours.yml --server-side=true --field-manager=second
W1220 15:36:02.687997   59398 loader.go:223] Config not found: /Users/pivotal/.kube/kind-config-1.16.3
associativeset.colours.example.com/simple-colour-set serverside-applied

If a previous manager tries to force-apply

2. List, map
2.1: Repeat 1.1 and 1.2
2.2: When keys change -> ???u




More observations:
* The managers stick around in managedFields, even when they're no longer around. I.e. when the fields that they manage are "gone". Is that intentional? What's the use case?
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
