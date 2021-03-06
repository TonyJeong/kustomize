package nameref

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/kustomize/api/k8sdeps/kunstruct"
	"sigs.k8s.io/kustomize/api/resid"
	"sigs.k8s.io/kustomize/api/resmap"
	"sigs.k8s.io/kustomize/api/resource"
	filtertest_test "sigs.k8s.io/kustomize/api/testutils/filtertest"
	"sigs.k8s.io/kustomize/api/types"
)

func TestNamerefFilter(t *testing.T) {
	testCases := map[string]struct {
		input         string
		candidates    string
		expected      string
		filter        Filter
		originalNames []string
	}{
		"simple scalar": {
			input: `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: dep
ref:
  name: oldName
`,
			candidates: `
apiVersion: apps/v1
kind: Secret
metadata:
  name: newName
---
apiVersion: apps/v1
kind: NotSecret
metadata:
  name: newName2
`,
			originalNames: []string{"oldName", ""},
			expected: `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: dep
ref:
  name: newName
`,
			filter: Filter{
				FieldSpec: types.FieldSpec{Path: "ref/name"},
				Target: resid.Gvk{
					Group:   "apps",
					Version: "v1",
					Kind:    "Secret",
				},
			},
		},
		"sequence": {
			input: `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: dep
seq:
- oldName1
- oldName2
`,
			candidates: `
apiVersion: apps/v1
kind: Secret
metadata:
  name: newName
---
apiVersion: apps/v1
kind: NotSecret
metadata:
  name: newName2
`,
			originalNames: []string{"oldName1", ""},
			expected: `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: dep
seq:
- newName
- oldName2
`,
			filter: Filter{
				FieldSpec: types.FieldSpec{Path: "seq"},
				Target: resid.Gvk{
					Group:   "apps",
					Version: "v1",
					Kind:    "Secret",
				},
			},
		},
		"mapping": {
			input: `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: dep
map:
  name: oldName
`,
			candidates: `
apiVersion: apps/v1
kind: Secret
metadata:
  name: newName
---
apiVersion: apps/v1
kind: NotSecret
metadata:
  name: newName2
`,
			originalNames: []string{"oldName", ""},
			expected: `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: dep
map:
  name: newName
`,
			filter: Filter{
				FieldSpec: types.FieldSpec{Path: "map"},
				Target: resid.Gvk{
					Group:   "apps",
					Version: "v1",
					Kind:    "Secret",
				},
			},
		},
		"mapping with namespace": {
			input: `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: dep
map:
  name: oldName
  namespace: oldNs
`,
			candidates: `
apiVersion: apps/v1
kind: Secret
metadata:
  name: newName
  namespace: oldNs
---
apiVersion: apps/v1
kind: NotSecret
metadata:
  name: newName2
`,
			originalNames: []string{"oldName", ""},
			expected: `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: dep
map:
  name: newName
  namespace: oldNs
`,
			filter: Filter{
				FieldSpec: types.FieldSpec{Path: "map"},
				Target: resid.Gvk{
					Group:   "apps",
					Version: "v1",
					Kind:    "Secret",
				},
			},
		},
	}

	for tn, tc := range testCases {
		t.Run(tn, func(t *testing.T) {
			factory := resource.NewFactory(kunstruct.NewKunstructuredFactoryImpl())
			referrer, err := factory.FromBytes([]byte(tc.input))
			if err != nil {
				t.Fatal(err)
			}
			tc.filter.Referrer = referrer

			resMapFactory := resmap.NewFactory(factory, nil)
			candidatesRes, err := factory.SliceFromBytesWithNames(
				tc.originalNames, []byte(tc.candidates))
			if err != nil {
				t.Fatal(err)
			}

			candidates := resMapFactory.FromResourceSlice(candidatesRes)
			tc.filter.ReferralCandidates = candidates

			if !assert.Equal(t,
				strings.TrimSpace(tc.expected),
				strings.TrimSpace(
					filtertest_test.RunFilter(t, tc.input, tc.filter))) {
				t.FailNow()
			}
		})
	}
}

func TestNamerefFilterUnhappy(t *testing.T) {
	testCases := map[string]struct {
		input         string
		candidates    string
		expected      string
		filter        Filter
		originalNames []string
	}{
		"invalid node type": {
			input: `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: dep
ref:
  name: null
`,
			candidates:    "",
			originalNames: []string{},
			expected:      "obj '' at path 'ref/name': node is expected to be either a string or a slice of string or a map of string",
			filter: Filter{
				FieldSpec: types.FieldSpec{Path: "ref/name"},
				Target: resid.Gvk{
					Group:   "apps",
					Version: "v1",
					Kind:    "Secret",
				},
			},
		},
		"multiple match": {
			input: `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: dep
ref:
  name: oldName
`,
			candidates: `
apiVersion: apps/v1
kind: Secret
metadata:
  name: newName
---
apiVersion: apps/v1
kind: Secret
metadata:
  name: newName2
`,
			originalNames: []string{"oldName", "oldName"},
			expected:      "",
			filter: Filter{
				FieldSpec: types.FieldSpec{Path: "ref/name"},
				Target: resid.Gvk{
					Group:   "apps",
					Version: "v1",
					Kind:    "Secret",
				},
			},
		},
		"no name": {
			input: `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: dep
ref:
  notName: oldName
`,
			candidates: `
apiVersion: apps/v1
kind: Secret
metadata:
  name: newName
---
apiVersion: apps/v1
kind: Secret
metadata:
  name: newName2
`,
			originalNames: []string{"oldName", "oldName"},
			expected:      "",
			filter: Filter{
				FieldSpec: types.FieldSpec{Path: "ref"},
				Target: resid.Gvk{
					Group:   "apps",
					Version: "v1",
					Kind:    "Secret",
				},
			},
		},
	}

	for tn, tc := range testCases {
		t.Run(tn, func(t *testing.T) {
			factory := resource.NewFactory(kunstruct.NewKunstructuredFactoryImpl())
			referrer, err := factory.FromBytes([]byte(tc.input))
			if err != nil {
				t.Fatal(err)
			}
			tc.filter.Referrer = referrer

			resMapFactory := resmap.NewFactory(factory, nil)
			candidatesRes, err := factory.SliceFromBytesWithNames(
				tc.originalNames, []byte(tc.candidates))
			if err != nil {
				t.Fatal(err)
			}

			candidates := resMapFactory.FromResourceSlice(candidatesRes)
			tc.filter.ReferralCandidates = candidates

			_, err = filtertest_test.RunFilterE(t, tc.input, tc.filter)
			if err == nil {
				t.Fatalf("expect an error")
			}
			if tc.expected != "" && !assert.EqualError(t, err, tc.expected) {
				t.FailNow()
			}
		})
	}
}
