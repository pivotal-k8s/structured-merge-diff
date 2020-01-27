/*
Copyright 2020 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package merge_test

import (
	"testing"

	"sigs.k8s.io/structured-merge-diff/v3/fieldpath"
	. "sigs.k8s.io/structured-merge-diff/v3/internal/fixture"
	"sigs.k8s.io/structured-merge-diff/v3/merge"
	"sigs.k8s.io/structured-merge-diff/v3/typed"

)

var atomicMapParser = func() Parser {
	parser, err := typed.NewParser(`types:
- name: type
  map:
    fields:
      - name: map
        type:
          namedType: atomicMap
- name: atomicMap
  map:
    elementType:
      scalar: string
    elementRelationship: atomic
`)
	if err != nil {
		panic(err)
	}
	return SameVersionParser{T: parser.Type("type")}
}()

var granularMapParser = func() Parser {
	parser, err := typed.NewParser(`types:
- name: type
  map:
    fields:
      - name: map
        type:
          namedType: granularMap
- name: granularMap
  map:
    elementType:
      scalar: string
    elementRelationship: separable
`)
	if err != nil {
		panic(err)
	}
	return SameVersionParser{T: parser.Type("type")}
}()

func TestAtomicMap(t *testing.T) {
	tests := map[string]TestCase{
		"atomic replace": {
			Ops: []Operation{
				Apply{
					Manager:    "first",
					APIVersion: "v1",
					Object: `
            map:
              name: a
          `,
				},
				Apply{
					Manager:    "first",
					APIVersion: "v1",
					Object: `
            map:
              id: two
          `,
				},
			},
			Object: `
        map:
          id: two
      `,
			APIVersion: "v1",
			Managed: fieldpath.ManagedFields{
				"first": fieldpath.NewVersionedSet(
					_NS(
						_P("map"),
					),
					"v1",
					true,
				),
			},
		},
		"conflict": {
			Ops: []Operation{
				Apply{
					Manager:    "first",
					APIVersion: "v1",
					Object: `
            map:
              name: a
              value: one
          `,
				},
				Apply{
					Manager:    "second",
					APIVersion: "v1",
					Object: `
            map:
              name: a
              value: two
          `,
						Conflicts: merge.Conflicts{
							merge.Conflict{Manager: "first", Path: _P("map")},
						},
				},
			},
			Object: `
        map:
          name: a
          value: one
      `,
			APIVersion: "v1",
			Managed: fieldpath.ManagedFields{
				"first": fieldpath.NewVersionedSet(
					_NS(
						_P("map"),
					),
					"v1",
					true,
				),
			},
		},
		"co-ownership": {
			Ops: []Operation{
				Apply{
					Manager:    "first",
					APIVersion: "v1",
					Object: `
            map:
              name: a
              value: one
          `,
				},
				Apply{
					Manager:    "second",
					APIVersion: "v1",
					Object: `
            map:
              name: a
              value: one
          `,
				},
				Apply{
					Manager:    "first",
					APIVersion: "v1",
					Object: `
            map:
              name: a
              value: two
          `,
						Conflicts: merge.Conflicts{
							merge.Conflict{Manager: "second", Path: _P("map")},
						},
				},
			},
			Object: `
        map:
          name: a
          value: one
      `,
			APIVersion: "v1",
			Managed: fieldpath.ManagedFields{
				"first": fieldpath.NewVersionedSet(
					_NS(
						_P("map"),
					),
					"v1",
					true,
				),
				"second": fieldpath.NewVersionedSet(
					_NS(
						_P("map"),
					),
					"v1",
					true,
				),
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			if err := test.Test(atomicMapParser); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestGranularMap(t *testing.T) {
	tests := map[string]TestCase{
		"atomic replace": {
			Ops: []Operation{
				Apply{
					Manager:    "first",
					APIVersion: "v1",
					Object: `
            map:
              name: a
          `,
				},
				Apply{
					Manager:    "first",
					APIVersion: "v1",
					Object: `
            map:
              id: two
          `,
				},
			},
			Object: `
        map:
          id: two
      `,
			APIVersion: "v1",
			Managed: fieldpath.ManagedFields{
				"first": fieldpath.NewVersionedSet(
					_NS(
						_P("map", _P("name")),
					),
					"v1",
					true,
				),
				"second": fieldpath.NewVersionedSet(
					_NS(
						_P("map", _KBF("name"), "name"),
					),
					"v1",
					true,
				),
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			if err := test.Test(granularMapParser); err != nil {
				t.Fatal(err)
			}
		})
	}
}
