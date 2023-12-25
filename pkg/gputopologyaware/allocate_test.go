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

func Test_allocate(t *testing.T) {
	device0 := newDevice(0)
	device1 := newDevice(1)
	device2 := newDevice(2)
	device3 := newDevice(3)
	invalidDevice2 := newDevice(2)

	// The device matrix looks like:
	// +----------+----------+----------+----------+----------+
	// |          |   GPU0   |  GPU1    |   GPU2   |   GPU3   |
	// +----------+----------+----------+----------+----------+
	// |   GPU0   |    -1    |     8    |     8    |     7    |
	// +----------+----------+----------+----------+----------+
	// |   GPU1   |     8    |    -1    |     7    |     7    |
	// +----------+----------+----------+----------+----------+
	// |   GPU2   |     8    |     7    |    -1    |     10    |
	// +----------+----------+----------+----------+----------+
	// |   GPU3   |     7    |     7    |     10    |    -1    |
	// +----------+----------+----------+----------+----------+
	device0.Links = map[int][]P2PLink{
		1: []P2PLink{
			{
				GPU:  device1,
				Type: P2PLinkType(8), // TwoNVLINKLinks 200
			},
		},
		2: []P2PLink{
			{
				GPU:  device2,
				Type: P2PLinkType(8),
			},
		},
		3: []P2PLink{
			{
				GPU:  device3,
				Type: P2PLinkType(7), // SingleNVLINK 100
			},
		},
	}

	device1.Links = map[int][]P2PLink{
		0: []P2PLink{
			{
				GPU:  device0,
				Type: P2PLinkType(8),
			},
		},
		2: []P2PLink{
			{
				GPU:  device2,
				Type: P2PLinkType(7),
			},
		},
		3: []P2PLink{
			{
				GPU:  device3,
				Type: P2PLinkType(7),
			},
		},
	}

	device2.Links = map[int][]P2PLink{
		0: []P2PLink{
			{
				GPU:  device0,
				Type: P2PLinkType(8),
			},
		},
		1: []P2PLink{
			{
				GPU:  device1,
				Type: P2PLinkType(7),
			},
		},
		3: []P2PLink{
			{
				GPU:  device3,
				Type: P2PLinkType(10), // ForNVLINKs 400
			},
		},
	}

	device3.Links = map[int][]P2PLink{
		0: []P2PLink{
			{
				GPU:  device0,
				Type: P2PLinkType(7),
			},
		},
		1: []P2PLink{
			{
				GPU:  device1,
				Type: P2PLinkType(7),
			},
		},
		2: []P2PLink{
			{
				GPU:  device2,
				Type: P2PLinkType(10),
			},
		},
	}

	invalidDevice2.Links = map[int][]P2PLink{
		0: []P2PLink{
			{
				GPU:  device0,
				Type: P2PLinkType(8),
			},
		},
		2: []P2PLink{
			{
				GPU:  device2,
				Type: P2PLinkType(7),
			},
		},
		3: []P2PLink{
			{
				GPU:  device3,
				Type: P2PLinkType(8),
			},
		},
	}

	testCases := []struct {
		name             string
		availableDevices []*Device
		size             int
		wantDevices      []*Device
		wantScore        int
	}{
		{
			// This requires optimization, however, we should keep in sync with k8s-device-plugin.
			name:             "request 1 GPU",
			availableDevices: []*Device{device0, device1, device2, device3},
			size:             1,
			wantDevices:      []*Device{device0},
			wantScore:        0,
		},
		{
			name:             "request full GPUs",
			availableDevices: []*Device{device0, device1, device2, device3},
			size:             4,
			wantDevices:      []*Device{device0, device1, device2, device3},
			wantScore:        1100,
		},
		{
			name:             "request 2 GPUs",
			availableDevices: []*Device{device0, device1, device2, device3},
			size:             2,
			wantDevices:      []*Device{device2, device3},
			wantScore:        400,
		},
		{
			name:             "request 3 GPUs",
			availableDevices: []*Device{device0, device1, device2, device3},
			size:             3,
			wantDevices:      []*Device{device0, device2, device3},
			wantScore:        700,
		},
		{
			name:             "invalid devices",
			availableDevices: []*Device{device0, device1, invalidDevice2, device3},
			size:             2,
			wantDevices:      nil,
			wantScore:        0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, score := allocate(tc.availableDevices, nil, tc.size)
			if diff := cmp.Diff(tc.wantDevices, got); diff != "" {
				t.Fatalf("unexpected result: (-want, +got): \n%v", diff)
			}

			if score != tc.wantScore {
				t.Fatalf("unexpected score, want: %v, got: %v", tc.wantScore, score)
			}
		})
	}

}
