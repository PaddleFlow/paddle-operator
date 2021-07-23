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
	"context"
	"errors"
	"fmt"
	"github.com/go-logr/logr"
	"github.com/paddleflow/paddle-operator/api/v1alpha1"
	"github.com/paddleflow/paddle-operator/controllers/extensions/common"
	"github.com/paddleflow/paddle-operator/controllers/extensions/driver"
	"github.com/paddleflow/paddle-operator/controllers/extensions/utils"
	appv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
	"time"
)

// SampleSetReconciler reconciles a SampleSet object
type SampleSetReconciler struct {
	client.Client
	Log      logr.Logger
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

//+kubebuilder:rbac:groups=batch.paddlepaddle.org,resources=samplesets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=batch.paddlepaddle.org,resources=samplesets/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=batch.paddlepaddle.org,resources=samplesets/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// the SampleSet object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.8.3/pkg/reconcile
func (r *SampleSetReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	r.Log.WithValues("SampleSet", req.NamespacedName)
	r.Log.V(1).Info("===============  Reconcile  ==============")

	// 1. Get sampleset object
	var sampleSet v1alpha1.SampleSet
	if err := r.Get(ctx, req.NamespacedName, &sampleSet); err != nil {
		e := fmt.Errorf("get sampleset %s error: %s", req.NamespacedName, err.Error())
		return utils.RequeueWithError(e)
	}

	// 2. If have not finalizer, add finalizer and requeue
	if !utils.HasFinalizer(&sampleSet.ObjectMeta, GetSampleSetFinalizer(req.Name)) {
		r.Log.V(1).Info("add finalizer successfully")
		return r.AddFinalizer(ctx, &sampleSet)
	}

	// 3. Get Driver and construct reconcile context
	var driverName v1alpha1.DriverName
	if sampleSet.Spec.CSI == nil {
		driverName = driver.DefaultDriver
	} else {
		driverName = sampleSet.Spec.CSI.Driver
	}
	CSIDriver, err := driver.GetDriver(driverName)
	if err != nil {
		r.Recorder.Event(&sampleSet, v1.EventTypeWarning,
			common.ErrorDriverNotExist, err.Error())
		return utils.NoRequeue()
	}

	// 4.
	RCtx := common.ReconcileContext{
		Ctx:      ctx,
		Client:   r.Client,
		Req:      &req,
		Log:      r.Log,
		Scheme:   r.Scheme,
		Recorder: r.Recorder,
	}

	//
	if utils.HasDeletionTimestamp(&sampleSet.ObjectMeta) {
		return r.deleteResource(&RCtx, &sampleSet)
	}

	ssrp := SampleSetReconcilePhase{&RCtx, &sampleSet, CSIDriver}

	return ssrp.reconcilePhase()
}

func (r *SampleSetReconciler) AddFinalizer (ctx context.Context, sampleSet *v1alpha1.SampleSet) (ctrl.Result, error) {
	if utils.HasDeletionTimestamp(&sampleSet.ObjectMeta) {
		return utils.NoRequeue()
	}
	sampleSetFinalizer := GetSampleSetFinalizer(sampleSet.Name)
	sampleSet.Finalizers = append(sampleSet.Finalizers, sampleSetFinalizer)
	if err := r.Update(ctx, sampleSet); err != nil {
		return utils.RequeueWithError(err)
	}
	return utils.RequeueImmediately()
}

func (r *SampleSetReconciler) deleteResource(
	ctx *common.ReconcileContext,
	sampleSet *v1alpha1.SampleSet) (ctrl.Result, error) {

	//sampleSetName := sampleSet.ObjectMeta.Name
	//pvcFinalizer := GetPVCFinalizer(sampleSetName)
	//if utils.HasFinalizer(sampleSet.ObjectMeta, pvcFinalizer) {
	//	r.deletePVC()
	//}


	// 判断是否有PaddleJob运行

	// 1. 等待正在运行的PaddleJob 和 SampleJob任务完成
	// 2. 提交SampleJob deleteCache 任务, 并禁止提交其他的SampleJob
	// 3. 等待deleteCache任务完成以后, 删除缓存节点上的Labels
	// 4. 删除 Runtime DaemonSet
	// 5. 删除pvc
	// 6. 删除pv资源

	return utils.NoRequeue()
}

func (r *SampleSetReconciler) deletePVC() {

}


// SetupWithManager sets up the controller with the Manager.
func (r *SampleSetReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := mgr.GetFieldIndexer().IndexField(
		context.Background(),
		&v1.Event{},
		common.EventIndexerKey,
		EventIndexerFunc); err != nil {
		return err
	}
	return ctrl.NewControllerManagedBy(mgr).
			For(&v1alpha1.SampleSet{}).
			Owns(&v1.Event{}).
			Complete(r)
}

type SampleSetReconcilePhase struct {
	//
	*common.ReconcileContext
	//
	SampleSet *v1alpha1.SampleSet
	//
	driver.Driver
}

//func NewSampleSetReconcilePhase() {
//	return
//}

func (s *SampleSetReconcilePhase) reconcilePhase() (ctrl.Result, error) {
	switch s.SampleSet.Status.Phase {
	case common.SampleSetNone:
		return s.reconcileNone()
	case common.SampleSetBound:
		return s.reconcileBound()
	case common.SampleSetMount:
		return s.reconcileMount()
	case common.SampleSetReady:
		return s.reconcileReady()
	//default:
	//	return utils.NoRequeue()
	}
	return utils.NoRequeue()
}

// reconcileNone After user create SampleSet CR then create PV and PVC automatically
func (s *SampleSetReconcilePhase) reconcileNone() (ctrl.Result, error) {
	s.Log.WithName("reconcileNone")
	volumeName := s.Req.Name
	label := s.GetLabel(volumeName)
	volumeKey := client.ObjectKey{Name: volumeName}

	// 1. check if persistent volume is already exist which have same name as SampleSet
	pv := &v1.PersistentVolume{}
	err := s.Get(s.Ctx, volumeKey, pv)
	if err == nil {
		// if the pv not created by paddle-operator then return error
		if _, exist := pv.Labels[label]; !exist {
			e := fmt.Errorf("pv %s is already exist", volumeName)
			s.Recorder.Event(s.SampleSet, v1.EventTypeWarning, common.ErrorPVAlreadyExist, e.Error())
			return utils.RequeueWithError(e)
		}

		// if pv is already create by paddle-operator
		if pv.Status.Phase != v1.VolumeBound {
			s.Log.V(1).Info("requeue and wait pv bound")
			return utils.RequeueAfter(1 * time.Second)
		}

		// if pv is bounded then update the phase of SampleSet to Bound
		s.SampleSet.Status.Phase = common.SampleSetBound
		if upErr := s.Status().Update(s.Ctx, s.SampleSet); upErr != nil {
			return utils.RequeueWithError(upErr)
		}
		s.Log.V(1).Info("update sampleset phase to bound")
		return utils.RequeueImmediately()
	}
	if client.IgnoreNotFound(err) != nil {
		e := fmt.Errorf("get pv %s error: %s", volumeName, err.Error())
		return utils.RequeueWithError(e)
	}
	s.Log.V(1).Info("pv is not exist")

	// 2. check if secret name is none or secret not exist
	if s.SampleSet.Spec.SecretRef == nil {
		e := errors.New("secretRef is empty")
		s.Recorder.Event(s.SampleSet, v1.EventTypeWarning, common.ErrorSecretNotExist, e.Error())
		return utils.NoRequeue()
	}
	secret := &v1.Secret{}
	if secretName := s.SampleSet.Spec.SecretRef.Name; secretName != "" {
		key := client.ObjectKey{
			Name: secretName,
			Namespace: s.Req.Namespace,
		}
		if err := s.Get(s.Ctx, key, secret); err != nil {
			e := fmt.Errorf("get secret %s error: %s", secretName, err.Error())
			s.Recorder.Event(s.SampleSet, v1.EventTypeWarning, common.ErrorSecretNotExist, e.Error())
			return utils.RequeueWithError(e)
		}
	} else {
		err := errors.New("secretRef name is not set")
		s.Recorder.Event(s.SampleSet, v1.EventTypeWarning, common.ErrorSecretNotExist, err.Error())
		return utils.NoRequeue()
	}
	s.Log.V(1).Info("secret is find", "secret", secret.Name)

	// construct request context
	rc := common.RequestContext{Req: s.Req, SampleSet: s.SampleSet, Secret: secret}

	// 3. create persistent volume and set its name as the SampleSet
	if err := s.CreatePV(pv, rc); err != nil {
		e := fmt.Errorf("create pv %s error: %s", volumeName, err.Error())
		s.Recorder.Event(s.SampleSet, v1.EventTypeWarning, common.ErrorCreatePV, e.Error())
		return utils.NoRequeue()
	}
	if err := s.Create(s.Ctx, pv); err != nil {
		e := fmt.Errorf("create pv %s error: %s", volumeName, err.Error())
		s.Recorder.Event(s.SampleSet, v1.EventTypeWarning, common.ErrorCreatePV, e.Error())
		return utils.RequeueWithError(e)
	}
	s.SampleSet.Finalizers = append(s.SampleSet.Finalizers, GetPVFinalizer(volumeName))
	s.Log.V(1).Info("create pv successfully")

	// 4. check if persistent volume claim is exist and have same name as the SampleSet
	pvc := &v1.PersistentVolumeClaim{}
	err = s.Get(s.Ctx, volumeKey, pvc)
	if client.IgnoreNotFound(err) != nil {
		e := fmt.Errorf("get pvc error: %s", err.Error())
		return utils.RequeueWithError(e)
	}
	if err == nil {
		if _, exist := pvc.Labels[label]; !exist {
			e := fmt.Errorf("pvc %s is already exist", volumeName)
			s.Recorder.Event(s.SampleSet, v1.EventTypeWarning, common.ErrorPVCAlreadyExist, e.Error())
			return utils.RequeueWithError(e)
		}
		return utils.RequeueAfter(1 * time.Second)
	}
	s.Log.V(1).Info("pvc is not exist")

	// 5. create persistent volume claim and set its name as the SampleSet
	if err := s.CreatePVC(pvc, rc); err != nil {
		e := fmt.Errorf("create pvc %s error: %s", volumeName, err.Error())
		s.Recorder.Event(s.SampleSet, v1.EventTypeWarning, common.ErrorCreatePVC, e.Error())
		return utils.NoRequeue()
	}
	if err := s.Create(s.Ctx, pvc); err != nil {
		e := fmt.Errorf("create pvc %s error: %s", volumeName, err.Error())
		s.Recorder.Event(s.SampleSet, v1.EventTypeWarning, common.ErrorCreatePV, e.Error())
		return utils.RequeueWithError(e)
	}
	s.SampleSet.Finalizers = append(s.SampleSet.Finalizers, GetPVCFinalizer(volumeName))
	s.Log.V(1).Info("create pvc successfully")

	// 6. update the finalizers of SampleSet
	namespacedName := s.Req.NamespacedName
	if err := s.Update(s.Ctx, s.SampleSet); err != nil {
		e := fmt.Errorf("update sampleset %s error: %s", namespacedName, err.Error())
		return utils.RequeueWithError(e)
	}
	s.Log.V(1).Info("update sampleset finalizer successfully")

	s.Recorder.Event(s.SampleSet, v1.EventTypeNormal, "", "create pv and pvc successfully")
	return utils.RequeueAfter(1 * time.Second)
}

// reconcileBound After create PV/PVC then create runtime StatefulSet
func (s *SampleSetReconcilePhase) reconcileBound() (ctrl.Result, error) {
	s.Log.WithName("reconcileBound")
	runtimeName := s.GetRuntimeName(s.Req.Name)
	runtimeKey := client.ObjectKey{
		Name: runtimeName,
		Namespace: s.Req.Namespace,
	}
	statefulSetName := runtimeKey.String()

	// 1. check if StatefulSet is already exist
	statefulSet := &appv1.StatefulSet{}
	err := s.Get(s.Ctx, runtimeKey, statefulSet)
	if err == nil {
		label := s.GetLabel(s.Req.Name)
		// if the StatefulSet not created by paddle-operator then return already exist error
		if _, exist := statefulSet.Labels[label]; !exist {
			e := fmt.Errorf("statefulset %s is already exist", statefulSetName)
			s.Recorder.Event(s.SampleSet, v1.EventTypeWarning, common.ErrorSSAlreadyExist, e.Error())
			return utils.RequeueWithError(e)
		}

		// if SampleSet partitions is changed then update StatefulSet replicas
		if *statefulSet.Spec.Replicas != s.SampleSet.Spec.Partitions {
			replicas := s.SampleSet.Spec.Partitions
			statefulSet.Spec.Replicas = &replicas
			if err := s.Update(s.Ctx, statefulSet); err != nil {
				e := fmt.Errorf("update statefulset %s error: %s", statefulSetName, err.Error())
				return utils.RequeueWithError(e)
			}
			s.Log.V(1).Info("update statefulset replicas")
			return utils.RequeueAfter(5 * time.Second)
		}

		// wait util at least one replica ready and update the phase of SampleSet to Mount
		if statefulSet.Status.ReadyReplicas > 0 {
			s.SampleSet.Status.Phase = common.SampleSetMount
			if err := s.Status().Update(s.Ctx, s.SampleSet); err != nil {
				return utils.RequeueWithError(err)
			}
			s.Log.V(1).Info("update sampleset phase to mount")
			return utils.RequeueImmediately()
		}

		// check if the pod created by StatefulSet is work properly
		var eventList v1.EventList
		values := []string{"Pod", s.Req.Namespace, runtimeName+"-0", "Warning"}
		fOpt:= client.MatchingFields{
			common.EventIndexerKey: strings.Join(values, "-"),
		}
		nOpt := client.InNamespace(s.Req.Namespace)
		lOpt := client.Limit(1)
		if err := s.List(s.Ctx, &eventList, fOpt, nOpt, lOpt); err != nil {
			e := fmt.Errorf("list event error: %s", err.Error())
			return utils.RequeueWithError(e)
		}
		eventLen := len(eventList.Items)
		if eventLen == 0 {
			s.Log.V(1).Info("wait at least one replica ready")
			return utils.RequeueAfter(5 * time.Second)
		}

		event := eventList.Items[0]
		// if the pod created by StatefulSet is not work properly then recorder event
		s.Recorder.Event(s.SampleSet, v1.EventTypeWarning, event.Reason, event.Message)
		return utils.RequeueWithError(errors.New(event.Message))
	}
	if client.IgnoreNotFound(err) != nil {
		e := fmt.Errorf("get statefulset %s error: %s", statefulSetName, err.Error())
		return utils.RequeueWithError(e)
	}

	// 2. get pv info and construct request context
	pv := &v1.PersistentVolume{}
	volumeName := s.Req.Name
	volumeKey := client.ObjectKey{Name: volumeName}
	if err := s.Get(s.Ctx, volumeKey, pv); err != nil {
		e := fmt.Errorf("get pv %s error: %s", volumeName, err.Error())
		return utils.RequeueWithError(e)
	}
	rc := common.RequestContext{Req: s.Req, SampleSet: s.SampleSet, PV: pv}

	// 3. create runtime StatefulSet
	if err := s.CreateRuntime(statefulSet, rc); err != nil {
		e := fmt.Errorf("create runtime %s error: %s", statefulSetName, err.Error())
		s.Recorder.Event(s.SampleSet, v1.EventTypeWarning, common.ErrorCreateRuntime, e.Error())
		return utils.NoRequeue()
	}
	if err := s.Create(s.Ctx, statefulSet); err != nil {
		e := fmt.Errorf("create runtime %s error: %s", statefulSetName, err.Error())
		s.Recorder.Event(s.SampleSet, v1.EventTypeWarning, common.ErrorCreateRuntime, e.Error())
		return utils.RequeueWithError(e)
	}
	s.Log.V(1).Info("create runtime successfully")

	// 4. add runtime finalizer to SampleSet and update SampleSet
	s.SampleSet.Finalizers = append(s.SampleSet.Finalizers, GetRuntimeFinalizer(volumeName))
	if err := s.Update(s.Ctx, s.SampleSet); err != nil {
		e := fmt.Errorf("update sampleset %s error: %s", statefulSetName, err.Error())
		return utils.RequeueWithError(e)
	}
	s.Log.V(1).Info("update sampleset finalizer successfully")

	s.Recorder.Event(s.SampleSet, v1.EventTypeNormal, "", "create runtime successfully")
	return utils.RequeueAfter(30 * time.Second)
}

// reconcileMount After create runtime daemon set, before data sync finish
func (s *SampleSetReconcilePhase) reconcileMount() (ctrl.Result, error) {
	s.Log.Info("************")
	return utils.RequeueAfter(20 * time.Second)
}

// reconcileReady
func (s *SampleSetReconcilePhase) reconcileReady() (ctrl.Result, error) {
	return utils.NoRequeue()
}

//func (s *SampleSetReconcilePhase) labelNodes(statefulSet *appv1.StatefulSet) error {
//	podList := &v1.PodList{}
//	label := s.GetLabel(s.Req.Name)
//	prefix := s.GetRuntimeName(s.Req.Name)
//	listOption :=  client.MatchingLabels{
//		label: "true",
//		"name": prefix,
//	}
//	if err := s.List(s.Ctx, podList, listOption); err != nil {
//		return fmt.Errorf("list pods by label %s error: %s", label, err.Error())
//	}
//	nodePodMap := make(map[string]v1.Pod)
//	for _, pod := range podList.Items {
//		if !strings.HasPrefix(pod.Name, prefix) {
//			continue
//		}
//		if pod.Spec.NodeName == "" {
//			continue
//		}
//		nodePodMap[pod.Spec.NodeName] = pod
//	}
//	if len(nodePodMap) != int(s.SampleSet.Spec.Partitions) {
//		return fmt.Errorf("the options of list statefulset pods is not valid")
//	}
//
//	// update the label of cache nodes and update pod nodeAffinity
//	for nodeName, podName := range nodePodMap {
//		node := &v1.Node{}
//		nodeKey := client.ObjectKey{Name: nodeName}
//		if err := s.Get(s.Ctx, nodeKey, node); err != nil {
//			return fmt.Errorf("get node %s error: %s", nodeName, err.Error())
//		}
//		index := strings.TrimPrefix(podName, prefix+"-")
//	}
//}

func GetSampleSetFinalizer(name string) string {
	return common.PaddleLabel + "/" + "sampleset-" + name
}

func GetPVFinalizer(name string) string {
	return common.PaddleLabel + "/" + "pv-" + name
}

func GetPVCFinalizer(name string) string {
	return common.PaddleLabel + "/" + "pvc-" + name
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