package plugins

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"strconv"
	"strings"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/kubernetes/pkg/scheduler/framework"
)

type CustomSchedulerArgs struct {
	Mode string `json:"mode"`
}

type CustomScheduler struct {
	handle    framework.Handle
	scoreMode string
}

var _ framework.PreFilterPlugin = &CustomScheduler{}
var _ framework.ScorePlugin = &CustomScheduler{}

// Name is the name of the plugin used in Registry and configurations.
const (
	Name              string = "CustomScheduler"
	groupNameLabel    string = "podGroup"
	minAvailableLabel string = "minAvailable"
	leastMode         string = "Least"
	mostMode          string = "Most"
)

func (cs *CustomScheduler) Name() string {
	return Name
}

// New initializes and returns a new CustomScheduler plugin.
func New(obj runtime.Object, h framework.Handle) (framework.Plugin, error) {
	cs := CustomScheduler{}
	mode := leastMode
	if obj != nil {
		args := obj.(*runtime.Unknown)
		var csArgs CustomSchedulerArgs
		if err := json.Unmarshal(args.Raw, &csArgs); err != nil {
			fmt.Printf("Error unmarshal: %v\n", err)
		}
		mode = csArgs.Mode
		if mode != leastMode && mode != mostMode {
			return nil, fmt.Errorf("invalid mode, got %s", mode)
		}
	}
	cs.handle = h
	cs.scoreMode = mode
	log.Printf("Custom scheduler runs with the mode: %s.", mode)

	return &cs, nil
}

// // filter the pod if the pod in group is less than minAvailable
func (cs *CustomScheduler) PreFilter(ctx context.Context, state *framework.CycleState, pod *v1.Pod) (*framework.PreFilterResult, *framework.Status) {
	log.Printf("Pod %s is in Prefilter phase.", pod.Name)
	newStatus := framework.NewStatus(framework.Success, "")

	// TODO
	// 1. extract the label of the pod
	// 2. retrieve the pod with the same group label
	// // 3. justify if the pod can be scheduled
	groupName := pod.Labels[groupNameLabel]
	minAvailableStr := pod.Labels[minAvailableLabel]

	minAvailable, err := strconv.Atoi(minAvailableStr)
	if err != nil {
		return nil, framework.NewStatus(framework.Error, "Invalid minAvailable value")
	}

	podLister := cs.handle.SharedInformerFactory().Core().V1().Pods().Lister()
	pods, err := podLister.List(labels.Set{groupNameLabel: groupName}.AsSelector())
	if err != nil {
		return nil, framework.NewStatus(framework.Error, fmt.Sprintf("Error listing pods: %v", err))
	}

	// Check if the number of pods in the group meets the minAvailable requirement
	if len(pods) < minAvailable {
		return nil, framework.NewStatus(framework.Unschedulable, "Not enough pods in the group")
	}

	return nil, newStatus
}

// PreFilterExtensions returns a PreFilterExtensions interface if the plugin implements one.
func (cs *CustomScheduler) PreFilterExtensions() framework.PreFilterExtensions {
	return nil
}
func RemoveSubstring(s, sep string) string {
	if idx := strings.Index(s, sep); idx != -1 {
		return s[:idx]
	}
	return s
}

// Score invoked at the score extension point.
func (cs *CustomScheduler) Score(ctx context.Context, state *framework.CycleState, pod *v1.Pod, nodeName string) (int64, *framework.Status) {
	log.Printf("Pod %s is in Score phase. Calculate the score of Node %s.", pod.Name, nodeName)

	// TODO
	// 1. retrieve the node allocatable memory
	// 2. return the score based on the scheduler mode
	nodeInfo, err := cs.handle.SnapshotSharedLister().NodeInfos().Get(nodeName)
	if err != nil {
		return 0, framework.NewStatus(framework.Error, fmt.Sprintf("Failed to get node info: %v", err))
	}

	allocatableMemory := nodeInfo.Allocatable.Memory

	var score int64
	if cs.scoreMode == leastMode {
		score = -allocatableMemory // lower memory, higher score
	} else if cs.scoreMode == mostMode {
		score = allocatableMemory // higher memory, higher score
	}

	return score, framework.NewStatus(framework.Success, "")
}

// ensure the scores are within the valid range
func (cs *CustomScheduler) NormalizeScore(ctx context.Context, state *framework.CycleState, pod *v1.Pod, scores framework.NodeScoreList) *framework.Status {
	// TODO
	// find the range of the current score and map to the valid range
	var minScore, maxScore int64 = int64(math.MaxInt64), int64(math.MinInt64)

	for _, nodeScore := range scores {
		if nodeScore.Score < minScore {
			minScore = nodeScore.Score
		}
		if nodeScore.Score > maxScore {
			maxScore = nodeScore.Score
		}
	}

	for i, nodeScore := range scores {
		if maxScore != minScore {
			scores[i].Score = ((nodeScore.Score - minScore) * 100) / (maxScore - minScore)
		} else {
			scores[i].Score = 0
		}
	}
	return framework.NewStatus(framework.Success, "")
}

// ScoreExtensions of the Score plugin.
func (cs *CustomScheduler) ScoreExtensions() framework.ScoreExtensions {
	return cs
}
