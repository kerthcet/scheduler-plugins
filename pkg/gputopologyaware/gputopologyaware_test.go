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
	"testing"

	"github.com/google/go-cmp/cmp"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubernetes/pkg/scheduler/framework"
	fakeframework "k8s.io/kubernetes/pkg/scheduler/framework/fake"
	"k8s.io/kubernetes/pkg/scheduler/framework/plugins/defaultbinder"
	"k8s.io/kubernetes/pkg/scheduler/framework/plugins/queuesort"
	frameworkruntime "k8s.io/kubernetes/pkg/scheduler/framework/runtime"
	st "k8s.io/kubernetes/pkg/scheduler/testing"
)

func TestGPUTopologyAware(t *testing.T) {
	testCases := []struct {
		name                   string
		pod                    *v1.Pod
		nodes                  []*v1.Node
		wantPreScoreStatusCode framework.Code
		wantScoreStatusCode    framework.Code
		wantScores             []int64
	}{
		{
			name:                   "pod with no GPU requested",
			pod:                    st.MakePod().Lim(map[v1.ResourceName]string{"cpu": "2"}).Obj(),
			wantPreScoreStatusCode: framework.Skip,
		},
		{
			name: "no node with GPU topology matrix provided",
			pod:  st.MakePod().Lim(map[v1.ResourceName]string{nvidiaGPU: "2"}).Obj(),
			nodes: []*v1.Node{
				makeNodeWithTopology("node-1", ""),
			},
			wantPreScoreStatusCode: framework.Success,
			wantScores:             []int64{0},
		},
		{
			name: "pod with 1 GPU requested",
			pod:  st.MakePod().Lim(map[v1.ResourceName]string{nvidiaGPU: "1"}).Obj(),
			nodes: []*v1.Node{
				makeNodeWithTopology("node-0", "[[-1,8,8,7],[8,-1,7,7],[8,7,-1,10],[7,7,10,-1]]"),
				makeNodeWithTopology("node-1", "[[-1,8,16,4],[8,-1,14,10],[16,14,-1,10],[4,10,10,-1]]"),
				makeNodeWithTopology("node-2", "[[-1,8,8,8],[8,-1,8,8],[8,8,-1,8],[8,8,8,-1]]"),
			},
			wantScores: []int64{0, 0, 0},
		},
		{
			name: "pod with 2 GPU requested",
			pod:  st.MakePod().Lim(map[v1.ResourceName]string{nvidiaGPU: "2"}).Obj(),
			nodes: []*v1.Node{
				makeNodeWithTopology("node-0", "[[-1,8,8,7],[8,-1,7,7],[8,7,-1,10],[7,7,10,-1]]"),
				makeNodeWithTopology("node-1", "[[-1,8,24,4],[8,-1,14,10],[24,14,-1,10],[4,10,10,-1]]"),
				makeNodeWithTopology("node-2", "[[-1,8,8,8],[8,-1,8,8],[8,8,-1,8],[8,8,8,-1]]"),
			},
			wantScores: []int64{22, 100, 11},
		},
		{
			name: "pod with 3 GPU requested",
			pod:  st.MakePod().Lim(map[v1.ResourceName]string{nvidiaGPU: "3"}).Obj(),
			nodes: []*v1.Node{
				makeNodeWithTopology("node-0", "[[-1,8,8,7],[8,-1,7,7],[8,7,-1,10],[7,7,10,-1]]"),
				makeNodeWithTopology("node-1", "[[-1,8,24,4],[8,-1,14,10],[24,14,-1,10],[4,10,10,-1]]"),
				makeNodeWithTopology("node-2", "[[-1,8,8,8],[8,-1,8,8],[8,8,-1,8],[8,8,8,-1]]"),
			},
			wantScores: []int64{12, 51, 11},
		},
		{
			name: "pod with 4 GPU requested",
			pod:  st.MakePod().Lim(map[v1.ResourceName]string{nvidiaGPU: "4"}).Obj(),
			nodes: []*v1.Node{
				makeNodeWithTopology("node-0", "[[-1,8,8,7],[8,-1,7,7],[8,7,-1,10],[7,7,10,-1]]"),
				makeNodeWithTopology("node-1", "[[-1,8,24,4],[8,-1,14,10],[24,14,-1,10],[4,10,10,-1]]"),
				makeNodeWithTopology("node-2", "[[-1,8,8,8],[8,-1,8,8],[8,8,-1,8],[8,8,8,-1]]"),
			},
			wantScores: []int64{10, 33, 11},
		},
	}
	// expectedList: []framework.NodeScore{{Name: "node1", Score: 18}, {Name: "node5", Score: framework.MaxNodeScore}, {Name: "node2", Score: 36}},
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			// cs := clientsetfake.NewSimpleClientset()
			// informerFactory := informers.NewSharedInformerFactory(cs, 0)
			registeredPlugins := []st.RegisterPluginFunc{
				st.RegisterBindPlugin(defaultbinder.Name, defaultbinder.New),
				st.RegisterQueueSortPlugin(queuesort.Name, queuesort.New),
				st.RegisterPluginAsExtensions(Name, New, "PreScore", "Score"),
			}
			nodeInfos := buildNodeInfos(tc.nodes)
			fakeSharedLister := &fakeSharedLister{nodes: nodeInfos}

			fh, err := st.NewFramework(
				registeredPlugins,
				"default-scheduler",
				ctx.Done(),
				// frameworkruntime.WithClientSet(cs),
				// frameworkruntime.WithInformerFactory(informerFactory),
				frameworkruntime.WithSnapshotSharedLister(fakeSharedLister),
			)
			if err != nil {
				t.Fatalf("fail to create framework: %v", err)
			}

			pl, err := New(nil, fh)
			if err != nil {
				t.Fatalf("failed to new plugin: %v", err)
			}

			plugin := pl.(*GPUTopologyAware)
			status := plugin.PreScore(ctx, nil, tc.pod, tc.nodes)
			if status.Code() != tc.wantPreScoreStatusCode {
				t.Fatalf("unexpected preScore status code, want: %v, got: %v", tc.wantPreScoreStatusCode, status.Code())
			}

			if status.IsSkip() {
				return
			}

			var gotScores []int64
			for _, node := range tc.nodes {
				score, status := plugin.Score(ctx, nil, tc.pod, node.Name)
				if status.Code() != tc.wantScoreStatusCode {
					t.Fatalf("unexpected preScore status code, want: %v, got: %v", tc.wantScoreStatusCode, status.Code())
				}
				gotScores = append(gotScores, score)
			}

			if diff := cmp.Diff(tc.wantScores, gotScores); diff != "" {
				t.Fatalf("unexpected result, (-want,+got): \n%s", diff)
			}
		})
	}
}

var _ framework.SharedLister = &fakeSharedLister{}

type fakeSharedLister struct {
	nodes []*framework.NodeInfo
}

func (f *fakeSharedLister) StorageInfos() framework.StorageInfoLister {
	return nil
}

func (f *fakeSharedLister) NodeInfos() framework.NodeInfoLister {
	return fakeframework.NodeInfoLister(f.nodes)
}

// buildNodeInfos build NodeInfo slice from a v1.Node slice
func buildNodeInfos(nodes []*v1.Node) []*framework.NodeInfo {
	res := make([]*framework.NodeInfo, len(nodes))
	for i := 0; i < len(nodes); i++ {
		res[i] = framework.NewNodeInfo()
		res[i].SetNode(nodes[i])
	}
	return res
}

// TODO: Migrate to scheduler testing library once we bump to v1.30.
func makeNodeWithTopology(name string, topologyMatrix string) *v1.Node {
	return &v1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Annotations: map[string]string{
				TOPOLOGY_INFO_KEY: topologyMatrix,
			},
		},
	}
}
