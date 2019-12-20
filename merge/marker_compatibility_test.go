/*
Copyright 2018 The Kubernetes Authors.

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
	"fmt"
	"testing"

	"sigs.k8s.io/structured-merge-diff/v2/fieldpath"
	. "sigs.k8s.io/structured-merge-diff/v2/internal/fixture"
	"sigs.k8s.io/structured-merge-diff/v2/typed"
)

var atomicListParser = func() typed.ParseableType {
	parser, err := typed.NewParser(`types:
- name: sets
  map:
    fields:
    - name: list
      type:
        list:
          elementType:
            scalar: string
          elementRelationship: atomic`)
	if err != nil {
		panic(err)
	}
	return parser.Type("sets")
}()

var setListParser = func() typed.ParseableType {
	parser, err := typed.NewParser(`types:
- name: sets
  map:
    fields:
    - name: list
      type:
        list:
          elementType:
            scalar: string
          elementRelationship: associative`)
	if err != nil {
		panic(err)
	}
	return parser.Type("sets")
}()

func TestAtomicList(t *testing.T) {
	operationsSequence := []Operation{
		//apply the object once
		Apply{
			Manager:    "manager-one",
			APIVersion: "v1",
			Object: `
				list:
				- a
				- b
			`,
		},
		//apply the object
		Apply{
			Manager:    "manager-one",
			APIVersion: "v2",
			Object: `
				list:
				- c
				- d
			`,
		},
	}

	managedFields := fieldpath.ManagedFields{
		"manager-one": fieldpath.NewVersionedSet(
			_NS(
				_P("list"),
			),
			"v2",
			false,
		),
	}

	testcase := TestCase{
		Ops: operationsSequence,
		Object: `
	    list:
	    - c
	    - d
	  `,
		Managed: managedFields,
	}

	//run tests

	t.Run("atomic list test", func(t *testing.T) {
		if err := testcase.Test(atomicListParser); err != nil {
			fmt.Printf("%#v\n", err)
			t.Fatal(err)
		}
	})
}

func TestAssociativeList(t *testing.T) {
	operationsSequence := []Operation{
		//apply the object once
		Apply{
			Manager:    "manager-one",
			APIVersion: "v1",
			Object: `
				list:
				- a
				- b
			`,
		},
		//reapply the object
		Apply{
			Manager:    "manager-two",
			APIVersion: "v2",
			Object: `
				list:
				- c
				- d
			`,
		},
	}

	managedFields := fieldpath.ManagedFields{
		"manager-one": fieldpath.NewVersionedSet(
			_NS(
				_P("list", _V("a")),
				_P("list", _V("b")),
			),
			"v1",
			false,
		),
		"manager-two": fieldpath.NewVersionedSet(
			_NS(
				_P("list", _V("c")),
				_P("list", _V("d")),
			),
			"v2",
			false,
		),
	}

	testcase := TestCase{
		Ops: operationsSequence,
		Object: `list:
- a
- b
- c
- d
`,
		Managed: managedFields,
	}

	//run tests

	// for name, test := range tests {
	t.Run("separable test", func(t *testing.T) {
		if err := testcase.Test(setListParser); err != nil {
			fmt.Printf("%#v\n", err)
			t.Fatal(err)
		}
	})
	// }
}

func TestAssociativeToAtomicMultipleAppliersShouldFail(t *testing.T) {
	operationsSequence := []Operation{
		//apply the object once
		Apply{
			Manager:    "manager-one",
			APIVersion: "v1",
			Object: `
				list:
				- a
				- b
			`,
		},
		//reapply the object
		Apply{
			Manager:    "manager-two",
			APIVersion: "v2",
			Object: `
				list:
				- c
				- d
			`,
		},
	}

	expectedManagedFields := fieldpath.ManagedFields{
		"manager-one": fieldpath.NewVersionedSet(
			_NS(
				_P("list", _V("a")),
				_P("list", _V("b")),
			),
			"v1",
			false,
		),
	}

	testcase := TestCase{
		Ops: operationsSequence,
		Object: `list:
- a
- b
`,
		Managed: expectedManagedFields,
	}

	t.Run("associative to atomic list test", func(t *testing.T) {
		if err := testcase.TestParserChange(atomicListParser, setListParser); err != nil {
			fmt.Printf("%#v\n", err)
			t.Fatal(err)
		}
	})
}
