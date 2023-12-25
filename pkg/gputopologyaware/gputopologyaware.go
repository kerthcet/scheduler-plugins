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
	"context"
	"errors"
	"fmt"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/kubernetes/pkg/api/v1/resource"
	"k8s.io/kubernetes/pkg/scheduler/framework"
)

const (
	// Name of the plugin used in the plugin registry and configurations.
	Name = "gputopologyaware"

	nvidiaGPU = "nvidia.com/gpu"
	// FIXME: the key should based on how NFD labels node. Still TBD.
	// See issue: https://github.com/NVIDIA/k8s-device-plugin/issues/465.
	TOPOLOGY_INFO_KEY = "node.gpu.info/topology"

	maxScoreFactor = 1800
)

// GPUTopologyAware can be aware of the GPU topology.
type GPUTopologyAware struct {
	sharedLister framework.SharedLister
}

var _ framework.PreScorePlugin = &GPUTopologyAware{}
var _ framework.ScorePlugin = &GPUTopologyAware{}

// New creates a GPUTopologyAware.
func New(plArgs runtime.Object, h framework.Handle) (framework.Plugin, error) {
	return &GPUTopologyAware{
		sharedLister: h.SnapshotSharedLister(),
	}, nil
}

// Name returns the name of the plugin.
func (pl *GPUTopologyAware) Name() string {
	return Name
}

func (pl *GPUTopologyAware) PreScore(ctx context.Context, cycleState *framework.CycleState, pod *v1.Pod, nodes []*v1.Node) *framework.Status {
	var requiredGPU bool

	// InitContainer ignored here for it's rare to set GPU in initContainer but normal container.
	// TODO: we should consider sidecar container one day.
	for _, c := range pod.Spec.Containers {
		// plain GPU is required to configure the Limits, see https://kubernetes.io/docs/tasks/manage-gpus/scheduling-gpus/
		if _, ok := c.Resources.Limits[nvidiaGPU]; ok {
			requiredGPU = true
			break
		}
	}

	// If pod doesn't require nvidia.com/gpu, then we take it as not interest with gpu topology.
	// We don't support vGPU or MIG right now.
	if !requiredGPU {
		return framework.NewStatus(framework.Skip)
	}
	return nil
}

func (pl *GPUTopologyAware) Score(ctx context.Context, cycleState *framework.CycleState, pod *v1.Pod, nodeName string) (int64, *framework.Status) {
	nodeInfo, err := pl.sharedLister.NodeInfos().Get(nodeName)
	if err != nil {
		return 0, framework.AsStatus(fmt.Errorf("failed to get node %q from Snapshot: %w", nodeName, err))
	}

	topologies, err := getTopologies(nodeInfo)
	if err != nil {
		return 0, framework.AsStatus(errors.New("failed to get topology matrix"))
	}

	if len(topologies) == 0 {
		return 0, nil
	}

	devices, err := NewDeviceList(topologies)
	if err != nil {
		return 0, framework.AsStatus(err)
	}

	// We take the Pod as a whole for we think all the containers should work closely.
	// If different containers for different responsibilities, we should extend here.
	limits := resource.PodLimits(pod, resource.PodResourcesOptions{})
	quota := limits[nvidiaGPU]
	_, score := allocate(devices, nil, int(quota.Value()))

	// We assume we're using the NVLink 4-gen, which may have
	// maximum 18 NVlinks per GPU to the limit.
	// TODO: Make factor-1800 configurable by NVLink generation in plugin arguments,
	// or maxScore might be too large.
	// Also optimize for requiring 1 GPU, see https://github.com/NVIDIA/go-gpuallocator/pull/18.
	maxScore := quota.Value() * (quota.Value() - 1) * maxScoreFactor / 2
	if maxScore == 0 {
		return 0, nil
	}

	return int64(score) * 100 / maxScore, nil
}

func (pl *GPUTopologyAware) ScoreExtensions() framework.ScoreExtensions {
	return pl
}

func (pl *GPUTopologyAware) NormalizeScore(ctx context.Context, state *framework.CycleState, p *v1.Pod, scores framework.NodeScoreList) *framework.Status {
	for i := range scores {
		if scores[i].Score > framework.MaxNodeScore {
			scores[i].Score = framework.MaxNodeScore
		}
	}
	return nil
}

func getTopologies(nodeInfo *framework.NodeInfo) ([][]int, error) {
	labels := nodeInfo.Node().GetAnnotations()
	if labels != nil && len(labels[TOPOLOGY_INFO_KEY]) != 0 {
		return decodeFloatSlice(labels[TOPOLOGY_INFO_KEY])
	}

	return nil, nil
}
