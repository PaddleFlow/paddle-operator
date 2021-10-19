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
	pdv1 "github.com/paddleflow/paddle-operator/api/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"strconv"
	"strings"
)

const (
	PodNameKey  = "paddle-pod-key"
	TaskNameKey = "paddle-task-name"
	TaskTypeKey = "paddle-task-type"
	TaskRankKey = "paddle-task-rank"
	PodRankKey  = "paddle-pod-rank"
)

func getTrainingRole(taskName string) string {
	if strings.HasPrefix(taskName, "ps") {
		return "PSERVER"
	} else if strings.HasPrefix(taskName, "worker") {
		return "TRAINER"
	}
	seq := strings.Split(taskName, "-")
	return strings.ToUpper(seq[0])
}

func genTaskType(taskName string) string {
	seq := strings.Split(taskName, "-")
	return strings.ToUpper(seq[0])
}

func genTaskChildName(name string, taskName string, idx int) string {
	return fmt.Sprintf("%s-%s-%d", name, taskName, idx)
}

// build element

func constructConfigMap(pdj *pdv1.PaddleJob, childPods corev1.PodList) (cm *corev1.ConfigMap) {
	cm = &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Labels:      map[string]string{},
			Annotations: map[string]string{},
			Name:        pdj.Name,
			Namespace:   pdj.Namespace,
		},
	}

	var paddle_port string
	if pdj.Spec.Intranet == pdv1.HostNetwork {
		paddle_port = pdj.ObjectMeta.Annotations[hostPort]
	} else {
		paddle_port = fmt.Sprintf("%d", PADDLE_PORT)
	}
	cm.Data = map[string]string{
		"TRAINER_PORTS_NUM": fmt.Sprintf("%d", HOST_PORT_NUM),
		"PADDLE_PORT":       paddle_port,
	}

	genEndpoint := func(pod *corev1.Pod) string {
		if pdj.Spec.Intranet == pdv1.Service {
			return fmt.Sprintf("%s", pod.Name)
		} else {
			return fmt.Sprintf("%s", pod.Status.PodIP)
		}
	}

	if pdj.Spec.Mode == pdv1.PaddleJobModeCollective {
		ends := make([]string, len(childPods.Items))
		types := make([]string, len(childPods.Items))
		for _, pod := range childPods.Items {
			i, _ := strconv.Atoi(pod.ObjectMeta.Annotations[TaskRankKey])
			j, _ := strconv.Atoi(pod.ObjectMeta.Annotations[PodRankKey])
			idx := (i+1)*(j+1) - 1
			ends[idx] = genEndpoint(&pod)
			types[idx] = strings.ToUpper(pod.ObjectMeta.Annotations[TaskTypeKey])
		}
		cm.Data["PADDLE_TRAINERS"] = strings.Join(ends, ",")
		cm.Data["PADDLE_TRAINERS_TYPES"] = strings.Join(types, ",")
		cm.Data["PADDLE_TRAINERS_NUM"] = fmt.Sprintf("%d", len(childPods.Items))

	} else if pdj.Spec.Mode == pdv1.PaddleJobModePS {
		/*
			        for _, pod := range childPods.Items {
			            idx, _ := strconv.Atoi(pod.ObjectMeta.Annotations[TaskRankKey])
			            ends[idx] = genEndpoint(&pod)
			            types[idx] = strings.ToUpper(pod.ObjectMeta.Annotations[TaskTypeKey])
			        }
					cm.Data["PADDLE_PSERVERS_IP_PORT_LIST"] = strings.Join(resources[pdv1.ResourcePS], ",")
					cm.Data["PADDLE_TRAINER_ENDPOINTS"] = strings.Join(resources[pdv1.ResourceWorker], ",")
					cm.Data["PADDLE_TRAINERS_NUM"] = fmt.Sprintf("%d", pdj.Spec.Worker.Replicas)
					cm.Data["PADDLE_HETER_ENDPOINTS"] = strings.Join(resources[pdv1.ResourceHeter], ",")

				if pdj.Spec.WithGloo != nil && *pdj.Spec.WithGloo > 0 && len(resources[pdv1.ResourcePS]) > 0 {
					cm.Data["PADDLE_WITH_GLOO"] = fmt.Sprintf("%d", *pdj.Spec.WithGloo)
					cm.Data["PADDLE_GLOO_RENDEZVOUS"] = "3"
					cm.Data["PADDLE_GLOO_HTTP_ENDPOINT"] = strings.Replace(resources[pdv1.ResourcePS][0],
						fmt.Sprintf(":%d", PADDLE_PORT),
						fmt.Sprintf(":%d", PADDLE_PORT+HOST_PORT_NUM-2),
						1)
				}
		*/
	}
	return cm
}

func constructPod(pdj *pdv1.PaddleJob, taskN int, idx int) (pod *corev1.Pod) {
	task := pdj.Spec.Tasks[taskN]
	pod = &corev1.Pod{}

	pod.ObjectMeta = *task.Template.ObjectMeta.DeepCopy()
	pod.Spec = *task.Template.Spec.DeepCopy()

	name := genTaskChildName(pdj.Name, task.Name, idx)
	pod.ObjectMeta.Name = name
	pod.ObjectMeta.Namespace = pdj.Namespace

	if pod.ObjectMeta.Labels == nil {
		pod.ObjectMeta.Labels = map[string]string{}
	}
	pod.ObjectMeta.Labels[PodNameKey] = name
	pod.ObjectMeta.Labels[TaskNameKey] = task.Name

	if pod.ObjectMeta.Annotations == nil {
		pod.ObjectMeta.Annotations = map[string]string{}
	}
	taskType := genTaskType(task.Name)
	pod.ObjectMeta.Annotations[PodRankKey] = fmt.Sprintf("%d", idx)
	pod.ObjectMeta.Annotations[TaskTypeKey] = taskType
	pod.ObjectMeta.Annotations[TaskRankKey] = fmt.Sprintf("%d", taskN)
	pod.ObjectMeta.Annotations[TaskNameKey] = task.Name

	pod.Spec.Hostname = name
	pod.Spec.Subdomain = name

	envIP := corev1.EnvVar{
		Name: "POD_IP",
	}
	if pdj.Spec.Intranet == pdv1.Service {
		envIP.Value = name
	} else {
		envIP.ValueFrom = &corev1.EnvVarSource{
			FieldRef: &corev1.ObjectFieldSelector{
				FieldPath: "status.podIP",
			},
		}
	}
	envRank := corev1.EnvVar{
		Name:  "PADDLE_TRAINER_ID",
		Value: fmt.Sprintf("%d", idx),
	}
	envRole := corev1.EnvVar{
		Name:  "TRAINING_ROLE",
		Value: getTrainingRole(task.Name),
	}
	envRole2 := corev1.EnvVar{
		Name:  "PADDLE_TRAINING_ROLE",
		Value: getTrainingRole(task.Name),
	}
	pod.Spec.Containers[0].Env = append(pod.Spec.Containers[0].Env, envIP, envRank, envRole, envRole2)

	if pdj.Spec.Elastic != nil {
		envJobID := corev1.EnvVar{
			Name:  "PADDLE_ELASTIC_JOB_ID",
			Value: fmt.Sprintf("%s-%s", pdj.Namespace, pdj.Name),
		}
		envNP := corev1.EnvVar{
			Name:  "PADDLE_ELASTIC_NP",
			Value: fmt.Sprintf("%d", pdj.Spec.Tasks[0].Replicas),
		}
		envTimeout := corev1.EnvVar{
			Name:  "PADDLE_ELASTIC_TIMEOUT",
			Value: "60",
		}

		pod.Spec.Containers[0].Env = append(pod.Spec.Containers[0].Env, envJobID, envNP, envTimeout)
	} else {
		envF := corev1.EnvFromSource{
			ConfigMapRef: &corev1.ConfigMapEnvSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: pdj.Name,
				},
			},
		}

		pod.Spec.Containers[0].EnvFrom = append(pod.Spec.Containers[0].EnvFrom, envF)
	}

	if pdj.Spec.Intranet == pdv1.Service {
		pod.Spec.Containers[0].Ports = append(pod.Spec.Containers[0].Ports, corev1.ContainerPort{ContainerPort: PADDLE_PORT})
	} else if pdj.Spec.Intranet == pdv1.HostNetwork {
		pod.Spec.HostNetwork = true
	}

	if pdj.Spec.Elastic != nil {
		pod.Spec.RestartPolicy = "OnFailure"
	}

	coInit := genCoordinateInitContainer()
	pod.Spec.InitContainers = append(pod.Spec.InitContainers, coInit)

	return pod
}

func genCoordinateInitContainer() corev1.Container {
	c := corev1.Container{
		Name:            coordContainerName,
		Image:           coordContainerImage,
		ImagePullPolicy: corev1.PullIfNotPresent,
		Command:         coordContainerCmd,
		Resources: corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse(coordContainerCpu),
				corev1.ResourceMemory: resource.MustParse(coordContainerMem),
				//corev1.ResourceEphemeralStorage: resource.MustParse(),
			},
		},
	}
	return c
}
