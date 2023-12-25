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

import "errors"

// This is mostly inspired by https://github.com/NVIDIA/go-gpuallocator.
// P2PLinkType defines the link information between two devices.
type P2PLinkType uint

// The following constants define the nature of a link between two devices.
// These include peer-2-peer and NVLink information.
const (
	P2PLinkUnknown P2PLinkType = iota
	P2PLinkCrossCPU
	P2PLinkSameCPU
	P2PLinkHostBridge
	P2PLinkMultiSwitch
	P2PLinkSingleSwitch
	P2PLinkSameBoard
	SingleNVLINKLink
	TwoNVLINKLinks
	ThreeNVLINKLinks
	FourNVLINKLinks
	FiveNVLINKLinks
	SixNVLINKLinks
	SevenNVLINKLinks
	EightNVLINKLinks
	NineNVLINKLinks
	TenNVLINKLinks
	ElevenNVLINKLinks
	TwelveNVLINKLinks
	ThirteenNVLINKLinks
	FourteenNVLINKLinks
	FifteenNVLINKLinks
	SixteenNVLINKLinks
	SeventeenNVLINKLinks
	EighteenNVLINKLinks
)

type P2PLink struct {
	GPU  *Device
	Type P2PLinkType
}

type Device struct {
	Index int
	Links map[int][]P2PLink
}

// The device matrix looks like:
// +----------+----------+----------+----------+----------+
// |          |   GPU0   |  GPU1    |   GPU2   |   GPU3   |
// +----------+----------+----------+----------+----------+
// |   GPU0   |    -1    |     8    |     8    |     7    |
// +----------+----------+----------+----------+----------+
// |   GPU1   |     8    |    -1    |     7    |     7    |
// +----------+----------+----------+----------+----------+
// |   GPU2   |     8    |     7    |    -1    |     8    |
// +----------+----------+----------+----------+----------+
// |   GPU3   |     7    |     7    |     8    |    -1    |
// +----------+----------+----------+----------+----------+
func NewDeviceList(matrix [][]int) ([]*Device, error) {
	if len(matrix) == 0 {
		return nil, errors.New("no devices provided")
	}
	if !validateSlice[int](matrix, func(matrix1, matrix2 []int) bool {
		return len(matrix1) == len(matrix2)
	}) {
		return nil, errors.New("invalid device list")
	}

	deviceLen := len(matrix[0])

	devices := make([]*Device, 0, deviceLen)
	for i := 0; i < deviceLen; i++ {
		devices = append(devices, newDevice(i))
	}

	for i, links := range matrix {
		for j, linkType := range links {
			if i == j {
				continue
			}
			if devices[i].Links == nil {
				devices[i].Links = make(map[int][]P2PLink, deviceLen-1)
			}

			// TODO: Do we need a list of P2PLink?
			// See issue: https://github.com/NVIDIA/go-gpuallocator/issues/21.
			devices[i].Links[j] = []P2PLink{
				{
					GPU:  devices[j],
					Type: P2PLinkType(linkType),
				},
			}
		}
	}

	return devices, nil
}

func validateSlice[T any](slice [][]T, f func([]T, []T) bool) bool {
	if len(slice) == 0 || len(slice) == 1 {
		return true
	}

	for i := 0; i < len(slice)-1; i++ {
		for j := i + 1; j < len(slice); j++ {
			if !f(slice[i], slice[j]) {
				return false
			}
		}
	}

	return true
}

func newDevice(index int) *Device {
	return &Device{Index: index}
}
