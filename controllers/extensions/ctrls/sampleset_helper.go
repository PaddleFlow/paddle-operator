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

package ctrls

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	appv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/uuid"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/paddleflow/paddle-operator/api/v1alpha1"
	"github.com/paddleflow/paddle-operator/controllers/extensions/common"
	"github.com/paddleflow/paddle-operator/controllers/extensions/utils"
)

func (s *SampleSetReconcilePhase) deleteResource() (ctrl.Result, error) {
	s.Log.WithValues("phase", "deleteResource")
	sampleSetName := s.SampleSet.Name
	label := s.GetLabel(sampleSetName)
	serviceName := s.GetServiceName(sampleSetName)
	runtimeName := s.GetRuntimeName(sampleSetName)

	// 1. wait util all PaddleJob is finish


	// 2. request runtime server to delete cache data
	podList := &v1.PodList{}
	nOpt := client.InNamespace(s.Req.Namespace)
	lOpt := client.MatchingLabels(map[string]string{
		label: "true", "name": runtimeName,
	})
	if err := s.List(s.Ctx, podList, nOpt, lOpt); err != nil {
		s.Log.Error(err, "fetch statefulset pod list error")
		return utils.RequeueWithError(err)
	}

	// 3. delete label of cache nodes
	nodeList := &v1.NodeList{}
	if err := s.List(s.Ctx, nodeList,  client.HasLabels{label}); err != nil {
		e := fmt.Errorf("list nodes error: %s", err.Error())
		return utils.RequeueWithError(e)
	}
	for _, node := range nodeList.Items {
		delete(node.Labels, label)
		if err := s.Update(s.Ctx, &node); err != nil {
			e := fmt.Errorf("remove nodes %s label error: %s", node.Name, err.Error())
			return utils.RequeueWithError(e)
		}
		s.Log.Info("remove label from node success", "node", node.Name)
	}

	// 4. delete runtime StatefulSet
	runtimeFinalizer := GetRuntimeFinalizer(sampleSetName)
	if utils.HasFinalizer(&s.SampleSet.ObjectMeta, runtimeFinalizer) {
		runtimeKey := client.ObjectKey{
			Name: runtimeName,
			Namespace: s.Req.Namespace,
		}
		statefulSet := &appv1.StatefulSet{}
		err := s.Get(s.Ctx, runtimeKey, statefulSet)
		if client.IgnoreNotFound(err) != nil {
			return utils.RequeueWithError(err)
		}
		if err == nil {
			if e := s.Delete(s.Ctx, statefulSet); e != nil {
				s.Recorder.Event(s.SampleSet, v1.EventTypeWarning,
					common.ErrorDeleteRuntime, e.Error())
				return utils.RequeueWithError(err)
			}
			s.Log.V(1).Info("delete statefulset")
		}
		utils.RemoveFinalizer(&s.SampleSet.ObjectMeta, runtimeFinalizer)
		// update SampleSet finalizer
		if err := s.Update(s.Ctx, s.SampleSet); err != nil {
			return utils.RequeueWithError(err)
		}
		s.Log.V(1).Info("remove statefulset finalizer")
		return utils.NoRequeue()
	}

	// 5. wait all StatefulSet replicas deleted
	if len(podList.Items) > 0 {
		s.Log.V(1).Info("wait all statefulset replicas deleted")
		return utils.RequeueAfter(5 * time.Second)
	}

	// 5. delete runtime service
	serviceFinalizer := GetServiceFinalizer(sampleSetName)
	if utils.HasFinalizer(&s.SampleSet.ObjectMeta, serviceFinalizer) {
		serviceKey := client.ObjectKey{
			Name: serviceName,
			Namespace: s.Req.Namespace,
		}

		service := &v1.Service{}
		err := s.Get(s.Ctx, serviceKey, service)
		if client.IgnoreNotFound(err) != nil {
			return utils.RequeueWithError(err)
		}
		if err == nil {
			if e := s.Delete(s.Ctx, service); e != nil {
				s.Recorder.Event(s.SampleSet, v1.EventTypeWarning,
					common.ErrorDeleteService, e.Error())
				return utils.RequeueWithError(err)
			}
			s.Log.V(1).Info("delete service")
		}
		utils.RemoveFinalizer(&s.SampleSet.ObjectMeta, serviceFinalizer)
		// update SampleSet finalizer
		if err := s.Update(s.Ctx, s.SampleSet); err != nil {
			return utils.RequeueWithError(err)
		}
		s.Log.V(1).Info("remove service finalizer")
		return utils.NoRequeue()
	}

	// 6. delete pvc
	pvcFinalizer := GetPVCFinalizer(sampleSetName)
	if utils.HasFinalizer(&s.SampleSet.ObjectMeta, pvcFinalizer) {
		pvc := &v1.PersistentVolumeClaim{}
		err := s.Get(s.Ctx, s.Req.NamespacedName, pvc)
		if client.IgnoreNotFound(err) != nil {
			return utils.RequeueWithError(err)
		}
		if err == nil {
			if e := s.Delete(s.Ctx, pvc); e != nil {
				s.Recorder.Event(s.SampleSet, v1.EventTypeWarning,
					common.ErrorDeletePVC, e.Error())
				s.Log.Error(e, "delete pvc error")
				return utils.RequeueWithError(e)
			}
			s.Log.V(1).Info("delete pvc")
		}
		utils.RemoveFinalizer(&s.SampleSet.ObjectMeta, pvcFinalizer)
		s.Log.V(1).Info("remove pvc finalizer")
	}

	// 7. delete pv
	pvFinalizer := GetPVFinalizer(sampleSetName)
	if utils.HasFinalizer(&s.SampleSet.ObjectMeta, pvFinalizer) {
		pv := &v1.PersistentVolume{}
		volumeKey := client.ObjectKey{Name: sampleSetName}
		err := s.Get(s.Ctx, volumeKey, pv)
		if client.IgnoreNotFound(err) != nil {
			return utils.RequeueWithError(err)
		}
		if err == nil {
			if e := s.Delete(s.Ctx, pv); e != nil {
				s.Recorder.Event(s.SampleSet, v1.EventTypeWarning,
					common.ErrorDeletePV, e.Error())
				return utils.RequeueWithError(e)
			}
			s.Log.V(1).Info("deleted pv")
		}
		utils.RemoveFinalizer(&s.SampleSet.ObjectMeta, pvFinalizer)
		s.Log.V(1).Info("remove pv finalizer")
	}

	// 8. remove SampleSet finalizer and update SampleSet
	sampleSetFinalizer := GetSampleSetFinalizer(sampleSetName)
	if utils.HasFinalizer(&s.SampleSet.ObjectMeta, sampleSetFinalizer) {
		utils.RemoveFinalizer(&s.SampleSet.ObjectMeta, sampleSetFinalizer)
	}
	if err := s.Update(s.Ctx, s.SampleSet); err != nil {
		return utils.RequeueWithError(err)
	}

	s.Log.V(1).Info("==== deleted all resource ====")
	return utils.NoRequeue()
}

func (s *SampleSetReconcilePhase) reconcileSyncFailed() (ctrl.Result, error) {
	s.Log.WithValues("phase", "reconcileSyncFailed")

	// 1. if spec.noSync update to true, then return phase to mount
	if s.SampleSet.Spec.NoSync {
		s.SampleSet.Status.Phase = common.SampleSetMount
		if err := s.Status().Update(s.Ctx, s.SampleSet); err != nil {
			e := fmt.Errorf("update sampleset %s error: %s", s.Req.Name, err.Error())
			return utils.RequeueWithError(e)
		}
		s.Log.Info("noSync is true, return SampleSet phase to mount")
		return utils.NoRequeue()
	}
	runtimeName := s.GetRuntimeName(s.Req.Name)
	serviceName := s.GetServiceName(s.Req.Name)
	filename := s.SampleSet.Status.JobsName.SyncJobName
	baseUri := utils.GetBaseUri(runtimeName, serviceName, 0)

	// 2. get syncJobOptions from runtime server
	oldOptions := &v1alpha1.SyncJobOptions{}
	err := utils.GetJobOption(oldOptions, filename, baseUri, common.PathSyncOptions)
	if err != nil {
		e := fmt.Errorf("get sync job option error: %s", err.Error())
		return utils.RequeueWithError(e)
	}

	// 3. create syncJobOptions for check if SampleSet is update by
	secretName := s.SampleSet.Spec.SecretRef.Name
	secretKey := client.ObjectKey{
		Name: secretName,
		Namespace: s.Req.Namespace,
	}
	secret := &v1.Secret{}
	if err := s.Get(s.Ctx, secretKey, secret); err != nil {
		e := fmt.Errorf("get secret %s error: %s", secretName, err.Error())
		s.Recorder.Event(s.SampleSet, v1.EventTypeWarning, common.ErrorSecretNotExist, e.Error())
		return utils.RequeueWithError(e)
	}
	s.Log.Info("get secret successful")
	rc := common.RequestContext{Secret: secret, SampleSet: s.SampleSet}

	newOptions := &v1alpha1.SyncJobOptions{}
	if err := s.CreateSyncJobOptions(newOptions, rc); err != nil {
		s.Log.Error(err, "create sync job options error")
		s.Recorder.Event(s.SampleSet, v1.EventTypeWarning, common.ErrorCreateSyncJobOption, err.Error())
		return utils.NoRequeue()
	}

	// 4. if syncJobOptions have not updated, then no requeue
	if reflect.DeepEqual(oldOptions, newOptions) {
		s.Log.Info("syncJobOptions has not update")
		return utils.NoRequeue()
	}

	// 5. if SampleSet updated by user and syncJobOptions has changed,
	// then delete old syncJobName and return SampleSet phase to mount.
	s.SampleSet.Status.JobsName.SyncJobName = ""
	s.SampleSet.Status.Phase = common.SampleSetMount
	if err := s.Status().Update(s.Ctx, s.SampleSet); err != nil {
		e := fmt.Errorf("update sampleset %s error: %s", s.Req.Name, err.Error())
		return utils.RequeueWithError(e)
	}
	s.Log.Info("syncJobOptions changed, return SampleSet phase to mount")
	return utils.NoRequeue()
}

func (s *SampleSetReconcilePhase) reconcilePartialReady() (ctrl.Result, error) {
	s.Log.WithValues("phase", "reconcilePartialReady")

	needUpdateStatefulSet := false
	label := s.GetLabel(s.Req.Name)
	runtimeName := s.GetRuntimeName(s.Req.Name)
	serviceName := s.GetServiceName(s.Req.Name)

	// 1. get runtime StatefulSet
	runtimeKey := client.ObjectKey{
		Name: runtimeName,
		Namespace: s.Req.Namespace,
	}
	statefulSetName := runtimeKey.String()
	statefulSet := &appv1.StatefulSet{}
	if err := s.Get(s.Ctx, runtimeKey, statefulSet); err != nil {
		e := fmt.Errorf("get statefulset %s error: %s", statefulSetName, err.Error())
		return utils.RequeueWithError(e)
	}

	// 2. list runtime server pods
	podList := &v1.PodList{}
	nOpt := client.InNamespace(s.Req.Namespace)
	lOpt := client.MatchingLabels(statefulSet.Spec.Selector.MatchLabels)
	if err := s.List(s.Ctx, podList, nOpt, lOpt); err != nil {
		e := fmt.Errorf("list pods error: %s", err.Error())
		return utils.RequeueWithError(e)
	}
	nodePodMap := make(map[string]string)
	for _, pod := range podList.Items {
		if pod.Status.Phase == v1.PodRunning {
			nodeName := pod.Spec.NodeName
			nodePodMap[nodeName] = pod.Name
		}
	}

	// 3. clean the cache data before terminate runtime pods
	specReplicas := *statefulSet.Spec.Replicas
	partitions := s.SampleSet.Spec.Partitions
	if partitions < specReplicas {
		if err := s.cleanCacheData(nodePodMap, partitions); err != nil {
			e := fmt.Errorf("clean cache data error: %s", err.Error())
			return utils.RequeueWithError(e)
		}
		s.Log.Info("clean data before terminate runtime pods")
	}

	// 4. update nodes label
	for nodeName, value := range nodePodMap {
		node := &v1.Node{}
		key := client.ObjectKey{Name: nodeName}
		if err := s.Get(s.Ctx, key, node); err != nil {
			e := fmt.Errorf("get node %s error: %s", nodeName, err.Error())
			return utils.RequeueWithError(e)
		}
		// if label is exist and the value is equal to the name of pod, continue
		if v, exist := node.Labels[label]; exist && v == value { continue }

		// update the label of node
		node.Labels[label] = value
		if err := s.Update(s.Ctx, node); err != nil {
			e := fmt.Errorf("get node %s error: %s", nodeName, err.Error())
			return utils.RequeueWithError(e)
		}
		s.Log.Info("label node successful", "name", nodeName, "label", value)
	}

	// 5. list nodes with label
	nodeList := &v1.NodeList{}
	opt := client.HasLabels{label}
	if err := s.List(s.Ctx, nodeList, opt); err != nil {
		e := fmt.Errorf("list nodes error: %s", err.Error())
		return utils.RequeueWithError(e)
	}

	// 6. remove the label of nodes with terminate runtime pod
	for _, node := range nodeList.Items {
		labelValue := node.Labels[label]
		podName, exist := nodePodMap[node.Name]
		if exist && labelValue == podName {
			continue
		}
		delete(node.Labels, label)
		err := s.Update(s.Ctx, &node)
		if err != nil {
			e := fmt.Errorf("remove nodes %s label error: %s", node.Name, err.Error())
			return utils.RequeueWithError(e)
		}
		s.Log.Info("remove label from node success", "node", node.Name)
	}

	// 7. collect all cache status from running runtime server
	status, err := utils.CollectAllCacheStatus(runtimeName, serviceName, len(nodePodMap))
	if err != nil {
		s.Log.Error(err, "get cache status error")
		return utils.RequeueAfter(10 * time.Second)
	}
	if status.ErrorMassage != "" {
		s.Log.Error(errors.New(status.ErrorMassage), "error massage from cache status")
	}
	if !reflect.DeepEqual(status, s.SampleSet.Status.CacheStatus) {
		needUpdateStatefulSet = true
		s.SampleSet.Status.CacheStatus = status
		s.Log.Info("cache status has changed")
	}
	s.Log.Info("collect all cache status successful")

	// 8. update StatefulSet if spec replicas is not equal to the partitions of SampleSet
	if specReplicas != partitions {
		statefulSet.Spec.Replicas = &partitions
		if err := s.Update(s.Ctx, statefulSet); err != nil {
			e := fmt.Errorf("update statefulset %s error: %s", statefulSetName, err.Error())
			return utils.RequeueWithError(e)
		}
		s.Log.Info("update statefulset spec.replicas successful")
	}

	// 9. update SampleSet RuntimeStatus and phase
	newReadyReplicas := statefulSet.Status.ReadyReplicas
	oldSpecReplicas := s.SampleSet.Status.RuntimeStatus.SpecReplicas
	oldReadyReplicas := s.SampleSet.Status.RuntimeStatus.ReadyReplicas
	if oldSpecReplicas != partitions || oldReadyReplicas != newReadyReplicas {
		needUpdateStatefulSet = true
		s.SampleSet.Status.RuntimeStatus.SpecReplicas = partitions
		s.SampleSet.Status.RuntimeStatus.ReadyReplicas = newReadyReplicas
		s.SampleSet.Status.RuntimeStatus.RuntimeReady = fmt.Sprintf(
			"%d/%d", newReadyReplicas, partitions)
	}

	allReady := partitions == specReplicas
	allReady = allReady && specReplicas == newReadyReplicas
	allReady = allReady && len(nodePodMap) == int(newReadyReplicas)
	if allReady {
		s.SampleSet.Status.Phase = common.SampleSetReady
	}

	if needUpdateStatefulSet || allReady {
		if err := s.Status().Update(s.Ctx, s.SampleSet); err != nil {
			e := fmt.Errorf("update sampleset %s error: %s", s.Req.Name, err.Error())
			return utils.RequeueWithError(e)
		}
		s.Log.Info("update sampleset phase to ready")
		return utils.NoRequeue()
	}
	return utils.RequeueAfter(60 * time.Second)
}

func (s *SampleSetReconcilePhase) cleanCacheData(nodePodMap map[string]string, partitions int32) error {
	// get the name of running pod with need to clean cache data
	var indexList []int
	for _, podName := range nodePodMap {
		nameList := strings.Split(podName, "-")
		if len(nameList) == 0 { continue }
		indexStr := nameList[len(nameList)-1]
		index, err := strconv.Atoi(indexStr)
		if err != nil { continue }
		if index+1 <= int(partitions) { continue }
		indexList = append(indexList, index)
	}
	opt := &v1alpha1.ClearJobOptions{}
	err := s.CreateClearJobOptions(opt, common.RequestContext{})
	if err != nil {
		return err
	}

	runtimeName := s.GetRuntimeName(s.Req.Name)
	serviceName := s.GetServiceName(s.Req.Name)
	for _, index := range indexList {
		baseUri := utils.GetBaseUri(runtimeName, serviceName, index)
		err := utils.PostJobOption(opt, uuid.NewUUID(), baseUri, common.PathClearOptions)
		if err != nil { return err }
	}
	return nil
}

func GetSampleSetFinalizer(name string) string {
	return common.PaddleLabel + "/" + "sampleset-" + name
}

func GetPVFinalizer(name string) string {
	return common.PaddleLabel + "/" + "pv-" + name
}

func GetPVCFinalizer(name string) string {
	return common.PaddleLabel + "/" + "pvc-" + name
}

func GetServiceFinalizer(name string) string {
	return common.PaddleLabel + "/" + "service" + name
}

func GetRuntimeFinalizer(name string) string {
	return common.PaddleLabel + "/" + "runtime-" + name
}

// EventIndexerFunc index
func EventIndexerFunc(obj client.Object) []string {
	event := obj.(*v1.Event)
	keys := []string{
		event.InvolvedObject.Kind,
		event.InvolvedObject.Namespace,
		event.InvolvedObject.Name,
		event.Type,
	}
	keyStr := strings.Join(keys, "-")
	return []string{keyStr}
}
