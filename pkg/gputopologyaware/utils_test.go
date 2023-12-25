/*
Copyright 2023 The Kubernetes Authors.

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

package gputopologyaware

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func Test_decodeFloatSlice(t *testing.T) {
	testCases := []struct {
		name       string
		inputs     string
		wantOutput [][]int
		wantError  bool
	}{
		{
			name:   "correct inputs",
			inputs: "[[-1,2,1,2], [2,-1,0,2], [1,0,-1,1],[2,2,1,-1]]",
			wantOutput: [][]int{
				[]int{-1, 2, 1, 2},
				[]int{2, -1, 0, 2},
				[]int{1, 0, -1, 1},
				[]int{2, 2, 1, -1},
			},
		},
		{
			name:      "invalid inputs with float type",
			inputs:    "[[-1,2.1,1,2], [2,-1,0,2], [1,0,-1,1],[2,2,1,-1]]",
			wantError: true,
		},
		{
			name:      "invalid inputs with string type",
			inputs:    "[[-1,2.1,1,2], [2,foo,0,2], [1,0,-1,1],[2,2,1,-1]]",
			wantError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := decodeFloatSlice(tc.inputs)
			if err != nil && !tc.wantError {
				t.Fatalf("unexpected err: %v", err)
			}

			if diff := cmp.Diff(tc.wantOutput, got); diff != "" {
				t.Fatalf("unexpected result (-want +got}: \n%v", diff)
			}
		})
	}
}
