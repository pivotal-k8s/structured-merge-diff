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
	"testing"

	"sigs.k8s.io/structured-merge-diff/v2/fieldpath"
)

func TestAtomicList(t *testing.T) {
	testcase := TestCase{
		Ops: []Operation{
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
				Manager:    "manager-two",
				APIVersion: "v1",
				Object: `
          list:
          - c
          - d
        `,
			},
		},
	}

	expectedObject := `
    list:
    - c
    - d
  `

	managedFields := fieldpath.ManagedFields{
		"manager-two": fieldpath.NewVersionedSet(
			_NS(
				_P("list", _KBF("name", "c")),
				_P("list", _KBF("name", "d"), "name"),
			),
			"v3",
			false,
		),
		"apply-two": fieldpath.NewVersionedSet(
			_NS(
				_P("list", _KBF("name", "c")),
				_P("list", _KBF("name", "c"), "name"),
			),
			"v2",
			false,
		),
	}

	//run tests
}
