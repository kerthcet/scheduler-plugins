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
	"encoding/json"
	"fmt"
)

// inputString looks like "[[-1,2,1,2], [2,-1,0,2], [1,0,-1,1],[2,2,1,-1]]"
func decodeFloatSlice(inputString string) ([][]int, error) {
	// Define a slice to hold the result
	var result [][]int

	// Use the encoding/json package to unmarshal the string into the slice
	err := json.Unmarshal([]byte(inputString), &result)
	if err != nil {
		fmt.Println("Error decoding string:", err)
		return nil, err
	}

	return result, nil

}
