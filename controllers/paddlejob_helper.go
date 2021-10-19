// Copyright 2021 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package controllers

import (
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	pdv1 "github.com/paddleflow/paddle-operator/api/v1"
	volcano "volcano.sh/apis/pkg/apis/scheduling/v1beta1"
)

const (
	schedulerNameVolcano         = "volcano"
	schedulingPodGroupAnnotation = "scheduling.k8s.io/group-name"

	coordContainerName  = "coord-paddle"
	coordContainerImage = "busybox:1"
	coordContainerCpu   = "10m"
	coordContainerMem   = "10m"
)

var (
	coordContainerCmd = []string{"sh", "-c", "while true; do if [ -f goon ]; then exit 0; else sleep 0.1; fi; done"}
)

func getRes(pdj *pdv1.PaddleJob, taskName string) (res *pdv1.TaskSpec) {
	for i, r := range pdj.Spec.Tasks {
		if r.Name == taskName {
			return pdj.Spec.Tasks[i]
		}
	}
	return nil
}

// status related

func isAllPodsReady(pdj *pdv1.PaddleJob, childPods corev1.PodList) bool {
	if !isAllPodsCreated(pdj) {
		return false
	}
	for _, pod := range childPods.Items {
		if pod.Status.PodIP == "" {
			return false
		}
	}
	return true
}

func isAllPodsCreated(pdj *pdv1.PaddleJob) bool {
	for _, t := range pdj.Spec.Tasks {
		if !isPodCreated(t, pdj.Status.Tasks[t.Name]) {
			return false
		}
	}
	return true
}

func isPodCreated(spec *pdv1.TaskSpec, status *pdv1.ResourceStatus) bool {
	if spec == nil {
		return true
	}
	if status != nil && len(status.Refs) == spec.Replicas {
		return true
	}
	return false
}

func isFailed(status *pdv1.ResourceStatus) bool {
	return status != nil && status.Failed > 0
}
func isPending(status *pdv1.ResourceStatus) bool {
	return status != nil && status.Pending > 0
}
func isStarting(status *pdv1.ResourceStatus) bool {
	return status != nil && status.Starting > 0
}
func isRunning(spec *pdv1.TaskSpec, status *pdv1.ResourceStatus) bool {
	return spec == nil || (status != nil && spec.Replicas == status.Running)
}
func isCompleted(spec *pdv1.TaskSpec, status *pdv1.ResourceStatus) bool {
	return spec == nil || (status != nil && spec.Replicas == status.Succeeded)
}

func getPaddleJobPhase(pdj *pdv1.PaddleJob) pdv1.PaddleJobPhase {

	// final phase won't change any more
	if pdj.Status.Phase == pdv1.Completed {
		return pdv1.Completed
	} else if pdj.Status.Phase == pdv1.Failed {
		return pdv1.Failed
	}

	for _, status := range pdj.Status.Tasks {
		if isFailed(status) {
			return pdv1.Failed
		} else if isStarting(status) {
			return pdv1.Starting
		} else if isPending(status) {
			return pdv1.Pending
		}
	}
	checkAll := func(check func(spec *pdv1.TaskSpec, status *pdv1.ResourceStatus) bool) bool {
		for _, t := range pdj.Spec.Tasks {
			if !check(t, pdj.Status.Tasks[t.Name]) {
				return false
			}
		}
		return true
	}
	if checkAll(isRunning) {
		return pdv1.Running
	}
	if checkAll(isCompleted) {
		return pdv1.Completed
	}

	if pdj.Status.Phase == "" {
		return pdv1.Pending
	}

	return pdj.Status.Phase
}

func isPodRealRuning(pod *corev1.Pod) bool {
	if pod.Status.Phase != corev1.PodRunning {
		return false
	}
	for i := range pod.Status.InitContainerStatuses {
		container := pod.Status.InitContainerStatuses[i]
		if !container.Ready {
			return false
		}
	}
	for i := range pod.Status.ContainerStatuses {
		container := pod.Status.ContainerStatuses[i]
		if !container.Ready || container.State.Running == nil {
			return false
		}
	}
	return true
}

func isAllCoordContainerRunning(childPods corev1.PodList) bool {
	for i, _ := range childPods.Items {
		if !isCoordContainerRunning(&childPods.Items[i]) {
			return false
		}
	}
	return true
}

func isCoordContainerRunning(pod *corev1.Pod) bool {
	if pod.Status.Phase != corev1.PodPending {
		return false
	}
	for i := range pod.Status.InitContainerStatuses {
		container := pod.Status.InitContainerStatuses[i]
		if container.Name == coordContainerName && container.State.Running != nil {
			return true
		}
	}
	return false
}

func getPaddleJobStartTime(pdj *pdv1.PaddleJob) *metav1.Time {
	if pdj.Status.StartTime.IsZero() && pdj.Status.Phase == pdv1.Running {
		tmp := metav1.Now()
		return &tmp
	}
	return pdj.Status.StartTime
}

func getPaddleJobCompleteTime(pdj *pdv1.PaddleJob) *metav1.Time {
	if pdj.Status.CompletionTime.IsZero() && (pdj.Status.Phase == pdv1.Completed || pdj.Status.Phase == pdv1.Failed) {
		tmp := metav1.Now()
		return &tmp
	}
	return pdj.Status.CompletionTime
}

func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}
func removeString(slice []string, s string) (result []string) {
	for _, item := range slice {
		if item == s {
			continue
		}
		result = append(result, item)
	}
	return
}

func constructService4Pod(pod corev1.Pod) *corev1.Service {
	var ports = []corev1.ServicePort{}
	for i := 0; i < HOST_PORT_NUM; i++ {
		ports = append(ports, corev1.ServicePort{
			Name: fmt.Sprintf("p-%d", i),
			Port: int32(PADDLE_PORT + i),
		})
	}
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pod.Name,
			Namespace: pod.Namespace,
			Labels:    map[string]string{},
		},
		Spec: corev1.ServiceSpec{
			Ports: ports,
			Selector: map[string]string{
				PodNameKey: pod.Name,
			},
			ClusterIP: "None",
		},
	}
	return svc
}

// for volcano

func withoutVolcano(pdj *pdv1.PaddleJob) bool {
	check := func(rs *pdv1.TaskSpec) bool {
		if rs != nil &&
			rs.Template.Spec.SchedulerName != "" &&
			rs.Template.Spec.SchedulerName != schedulerNameVolcano {
			return true
		} else {
			return false
		}
	}
	for _, spec := range pdj.Spec.Tasks {
		if check(spec) {
			return true
		}
	}
	return false
}

func constructPodGroup(pdj *pdv1.PaddleJob) *volcano.PodGroup {
	pg := &volcano.PodGroup{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: pdj.Namespace,
			Name:      pdj.Name,
		},
	}

	pg.Spec.MinMember = getTotalReplicas(pdj)
	pg.Spec.MinResources = getPGMinResource(pdj)

	if pdj.Spec.SchedulingPolicy != nil {
		// minAvailable specified by user which not equals to total replicas
		// DO NOT make sense in current paddle scenario
		if pdj.Spec.SchedulingPolicy.MinAvailable != nil {
			pg.Spec.MinMember = *pdj.Spec.SchedulingPolicy.MinAvailable
		}
		if pdj.Spec.SchedulingPolicy.Queue != "" {
			pg.Spec.Queue = pdj.Spec.SchedulingPolicy.Queue
		}
		if pdj.Spec.SchedulingPolicy.PriorityClass != "" {
			pg.Spec.PriorityClassName = pdj.Spec.SchedulingPolicy.PriorityClass
		}
		if pdj.Spec.SchedulingPolicy.MinResources != nil {
			pg.Spec.MinResources = &pdj.Spec.SchedulingPolicy.MinResources
		}
	}

	return pg
}

func getTotalReplicas(pdj *pdv1.PaddleJob) int32 {
	total := 0
	for _, spec := range pdj.Spec.Tasks {
		if spec != nil {
			total += spec.Replicas
		}
	}
	return int32(total)
}

func getPGMinResource(pdj *pdv1.PaddleJob) *corev1.ResourceList {
	addRes := func(crr, res corev1.ResourceList) {
		for name, quantity := range res {
			if value, ok := crr[name]; !ok {
				crr[name] = quantity.DeepCopy()
			} else {
				value.Add(quantity)
				crr[name] = value
			}
		}
	}
	// consider only the case minMember == minAvailable
	totalRes := corev1.ResourceList{}
	for _, spec := range pdj.Spec.Tasks {
		if spec == nil {
			continue
		}
		for i := 0; i < spec.Replicas; i++ {
			for _, c := range spec.Template.Spec.Containers {
				if c.Resources.Requests != nil {
					addRes(totalRes, c.Resources.Requests)
				} else {
					addRes(totalRes, c.Resources.Limits)
				}
			}
		}
	}
	return &totalRes
}
