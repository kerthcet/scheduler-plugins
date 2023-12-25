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
	"github.com/google/go-cmp/cmp/cmpopts"
)

func Test_NewDeviceList(t *testing.T) {
	testCases := []struct {
		name        string
		matrix      [][]int
		wantDevices []*Device
		wantError   bool
	}{
		{
			name:      "empty matrix",
			matrix:    nil,
			wantError: true,
		},
		{
			name:      "invalid matrix",
			matrix:    [][]int{[]int{1, 2, 3}, []int{3, 2, 1}, []int{3, 0}},
			wantError: true,
		},
		{
			name: "correct  matrix",
			matrix: [][]int{
				[]int{-1, 8, 8, 7},
				[]int{8, -1, 7, 7},
				[]int{8, 7, -1, 8},
				[]int{7, 7, 8, -1},
			},
			wantDevices: []*Device{
				&Device{
					Index: 0,
					Links: map[int][]P2PLink{
						1: []P2PLink{
							{
								GPU:  &Device{Index: 1},
								Type: P2PLinkType(8),
							},
						},
						2: []P2PLink{
							{
								GPU:  &Device{Index: 2},
								Type: P2PLinkType(8),
							},
						},
						3: []P2PLink{
							{
								GPU:  &Device{Index: 3},
								Type: P2PLinkType(7),
							},
						},
					},
				},
				&Device{
					Index: 1,
					Links: map[int][]P2PLink{
						0: []P2PLink{
							{
								GPU:  &Device{Index: 0},
								Type: P2PLinkType(8),
							},
						},
						2: []P2PLink{
							{
								GPU:  &Device{Index: 2},
								Type: P2PLinkType(7),
							},
						},
						3: []P2PLink{
							{
								GPU:  &Device{Index: 3},
								Type: P2PLinkType(7),
							},
						},
					},
				},
				&Device{
					Index: 2,
					Links: map[int][]P2PLink{
						0: []P2PLink{
							{
								GPU:  &Device{Index: 0},
								Type: P2PLinkType(8),
							},
						},
						1: []P2PLink{
							{
								GPU:  &Device{Index: 1},
								Type: P2PLinkType(7),
							},
						},
						3: []P2PLink{
							{
								GPU:  &Device{Index: 3},
								Type: P2PLinkType(8),
							},
						},
					},
				},
				&Device{
					Index: 3,
					Links: map[int][]P2PLink{
						0: []P2PLink{
							{
								GPU:  &Device{Index: 0},
								Type: P2PLinkType(7),
							},
						},
						1: []P2PLink{
							{
								GPU:  &Device{Index: 1},
								Type: P2PLinkType(7),
							},
						},
						2: []P2PLink{
							{
								GPU:  &Device{Index: 2},
								Type: P2PLinkType(8),
							},
						},
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			devices, err := NewDeviceList(tc.matrix)
			if err != nil && !tc.wantError {
				t.Fatalf("error: %v", err)
			}

			// We ignore the Links here or will lead to cycle reference.
			if diff := cmp.Diff(tc.wantDevices, devices, cmpopts.IgnoreFields(Device{}, "Links")); diff != "" {
				t.Fatalf("unexpected result(-want +got): \n%s", diff)
			}
		})
	}
}
