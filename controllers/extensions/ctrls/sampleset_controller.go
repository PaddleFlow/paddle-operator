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
	"strings"
	"time"

	"github.com/go-logr/logr"
	appv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/paddleflow/paddle-operator/api/v1alpha1"
	"github.com/paddleflow/paddle-operator/controllers/extensions/common"
	"github.com/paddleflow/paddle-operator/controllers/extensions/driver"
	"github.com/paddleflow/paddle-operator/controllers/extensions/utils"
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

	// 1. Get SampleSet
	var sampleSet v1alpha1.SampleSet
	if err := r.Get(ctx, req.NamespacedName, &sampleSet); err != nil {
		r.Log.Error(err, "unable to fetch SampleSet")
		return utils.RequeueWithError(client.IgnoreNotFound(err))
	}

	// 2. If has not finalizer and has not deletion timestamp, then add finalizer and requeue
	if !utils.HasFinalizer(&sampleSet.ObjectMeta, GetSampleSetFinalizer(req.Name)) &&
		!utils.HasDeletionTimestamp(&sampleSet.ObjectMeta) {
		r.Log.V(1).Info("add finalizer successfully")
		return r.AddFinalizer(ctx, &sampleSet)
	}

	// 3. Get driver and construct reconcile context
	var driverName v1alpha1.DriverName
	if sampleSet.Spec.CSI == nil {
		driverName = driver.DefaultDriver
	} else {
		driverName = sampleSet.Spec.CSI.Driver
	}
	CSIDriver, err := driver.GetDriver(driverName)
	if err != nil {
		r.Log.Error(err, "get driver error")
		r.Recorder.Event(&sampleSet, v1.EventTypeWarning,
			common.ErrorDriverNotExist, err.Error())
		return utils.NoRequeue()
	}
	RCtx := common.ReconcileContext{
		Ctx:      ctx,
		Client:   r.Client,
		Req:      &req,
		Log:      r.Log,
		Scheme:   r.Scheme,
		Recorder: r.Recorder,
	}
	// 4. construct SampleSet Reconcile Phase
	srp := SampleSetReconcilePhase{
		Driver: CSIDriver,
		SampleSet: &sampleSet,
		ReconcileContext: &RCtx,
	}
	return srp.reconcilePhase()
}

// AddFinalizer add finalizer to SampleSet
func (r *SampleSetReconciler) AddFinalizer (ctx context.Context, sampleSet *v1alpha1.SampleSet) (ctrl.Result, error) {
	sampleSetFinalizer := GetSampleSetFinalizer(sampleSet.Name)
	sampleSet.Finalizers = append(sampleSet.Finalizers, sampleSetFinalizer)
	if err := r.Update(ctx, sampleSet); err != nil {
		return utils.RequeueWithError(err)
	}
	return utils.NoRequeue()
}

// SetupWithManager sets up the controller with the Manager.
func (r *SampleSetReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := mgr.GetFieldIndexer().IndexField(
		context.Background(),
		&v1.Event{},
		common.IndexerKeyEvent,
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

func (s *SampleSetReconcilePhase) reconcilePhase() (ctrl.Result, error) {
	// if SampleSet has deletion timestamp the delete all resource create by this controller
	if utils.HasDeletionTimestamp(&s.SampleSet.ObjectMeta) {
		return s.deleteResource()
	}

	// Reconcile the phase of SampleSet from None to Ready
	switch s.SampleSet.Status.Phase {
	case common.SampleSetNone:
		return s.reconcileNone()
	case common.SampleSetBound:
		return s.reconcileBound()
	case common.SampleSetMount:
		return s.reconcileMount()
	case common.SampleSetSyncing:
		return s.reconcileSyncing()
	case common.SampleSetPartialReady:
		return s.reconcilePartialReady()
	case common.SampleSetReady:
		return s.reconcileReady()
	}
	s.Log.Error(fmt.Errorf("phase %s not support", s.SampleSet.Status.Phase), "")
	return utils.NoRequeue()
}

func (s *SampleSetReconcilePhase) deleteResource() (ctrl.Result, error) {
	s.Log.WithValues("phase", "deleteResource")

	// 1. 等待正在运行的PaddleJob 和 SampleJob任务完成
	// 2. 提交SampleJob deleteCache 任务, 并禁止提交其他的SampleJob
	// 3. 等待deleteCache任务完成以后, 删除缓存节点上的Labels
	// 4. 删除 Runtime DaemonSet
	// 5. 删除pvc
	// 6. 删除pv资源

	sampleSetName := s.SampleSet.Name
	label := s.GetLabel(sampleSetName)

	// delete label of cache nodes

	// 4. delete runtime StatefulSet
	runtimeFinalizer := GetRuntimeFinalizer(sampleSetName)
	if utils.HasFinalizer(&s.SampleSet.ObjectMeta, runtimeFinalizer) {
		runtimeName := s.GetRuntimeName(sampleSetName)
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
	podList := &v1.PodList{}
	lOpt := client.MatchingLabels(
		map[string]string{
			label: "true",
		})
	if err := s.List(s.Ctx, podList, lOpt); err != nil {
		s.Log.Error(err, "fetch statefulset pod list error")
		return utils.RequeueWithError(err)
	}
	if len(podList.Items) > 0 {
		s.Log.V(1).Info("wait all statefulset replicas deleted")
		return utils.RequeueAfter(5 * time.Second)
	}

	// 5. delete runtime service
	serviceFinalizer := GetServiceFinalizer(sampleSetName)
	if utils.HasFinalizer(&s.SampleSet.ObjectMeta, serviceFinalizer) {
		serviceName := s.GetServiceName(sampleSetName)
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

// reconcileNone After user create SampleSet CR then create PV and PVC automatically
func (s *SampleSetReconcilePhase) reconcileNone() (ctrl.Result, error) {
	s.Log.WithValues("phase", "reconcileNone")

	sampleSetName := s.Req.Name
	label := s.GetLabel(sampleSetName)

	// 1. check if persistent volume is already exist which have same name as SampleSet
	pv := &v1.PersistentVolume{}
	volumeKey := client.ObjectKey{Name: sampleSetName}
	err := s.Get(s.Ctx, volumeKey, pv)
	if err == nil {
		// if the pv not created by paddle-operator then return error
		if _, exist := pv.Labels[label]; !exist {
			e := fmt.Errorf("pv %s is already exist", sampleSetName)
			s.Recorder.Event(s.SampleSet, v1.EventTypeWarning, common.ErrorPVAlreadyExist, e.Error())
			return utils.RequeueWithError(e)
		}

		// if pv is already create by paddle-operator, wait util pv phase is Bound
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
		//return utils.RequeueAfter(1 * time.Second)
		return utils.NoRequeue()
	}
	if client.IgnoreNotFound(err) != nil {
		e := fmt.Errorf("get pv %s error: %s", sampleSetName, err.Error())
		return utils.RequeueWithError(e)
	}
	s.Log.V(1).Info("pv is not exist")

	// 2. check if secret name is none or secret not exist
	if s.SampleSet.Spec.SecretRef == nil {
		e := errors.New("spec.secretRef should not be empty")
		s.Log.Error(e, "spec.secretRef is nil")
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
		err := errors.New("spec.secretRef.name is not set")
		s.Log.Error(err, "spec.secretRef.name is empty string")
		s.Recorder.Event(s.SampleSet, v1.EventTypeWarning, common.ErrorSecretNotExist, err.Error())
		return utils.NoRequeue()
	}
	s.Log.V(1).Info("secret is find", "secret", secret.Name)

	// construct request context
	rc := common.RequestContext{Req: s.Req, SampleSet: s.SampleSet, Secret: secret}

	// 3. create persistent volume and set its name as the SampleSet
	if err := s.CreatePV(pv, rc); err != nil {
		s.Log.Error(err, "create pv error", "pv", sampleSetName)
		s.Recorder.Event(s.SampleSet, v1.EventTypeWarning, common.ErrorCreatePV, err.Error())
		return utils.NoRequeue()
	}
	if err := s.Create(s.Ctx, pv); err != nil {
		e := fmt.Errorf("create pv %s error: %s", sampleSetName, err.Error())
		s.Recorder.Event(s.SampleSet, v1.EventTypeWarning, common.ErrorCreatePV, e.Error())
		return utils.RequeueWithError(e)
	}
	s.SampleSet.Finalizers = append(s.SampleSet.Finalizers, GetPVFinalizer(sampleSetName))
	s.Log.V(1).Info("create pv successful")

	// 4. check if persistent volume claim is exist and have same name as the SampleSet
	pvc := &v1.PersistentVolumeClaim{}
	namespacedName := s.Req.NamespacedName.String()
	err = s.Get(s.Ctx, s.Req.NamespacedName, pvc)
	if client.IgnoreNotFound(err) != nil {
		e := fmt.Errorf("get pvc error: %s", err.Error())
		return utils.RequeueWithError(e)
	}
	if err == nil {
		if _, exist := pvc.Labels[label]; !exist {
			e := fmt.Errorf("pvc %s is already exist", namespacedName)
			s.Recorder.Event(s.SampleSet, v1.EventTypeWarning, common.ErrorPVCAlreadyExist, e.Error())
			return utils.RequeueWithError(e)
		}
		return utils.RequeueAfter(1 * time.Second)
	}
	s.Log.V(1).Info("pvc is not exist")

	// 5. create persistent volume claim and set its name as the SampleSet
	if err := s.CreatePVC(pvc, rc); err != nil {
		s.Log.Error(err, "create pvc error", "pvc", namespacedName)
		s.Recorder.Event(s.SampleSet, v1.EventTypeWarning, common.ErrorCreatePVC, err.Error())
		return utils.NoRequeue()
	}
	if err := s.Create(s.Ctx, pvc); err != nil {
		e := fmt.Errorf("create pvc %s error: %s", namespacedName, err.Error())
		s.Recorder.Event(s.SampleSet, v1.EventTypeWarning, common.ErrorCreatePV, e.Error())
		return utils.RequeueWithError(e)
	}
	s.SampleSet.Finalizers = append(s.SampleSet.Finalizers, GetPVCFinalizer(sampleSetName))
	s.Log.V(1).Info("create pvc successful")

	// 6. update the finalizers of SampleSet
	if err := s.Update(s.Ctx, s.SampleSet); err != nil {
		e := fmt.Errorf("update sampleset %s error: %s", namespacedName, err.Error())
		return utils.RequeueWithError(e)
	}
	s.Log.V(1).Info("update sampleset pv/pvc finalizer successful")

	s.Recorder.Eventf(s.SampleSet, v1.EventTypeNormal, common.EventCreate,
		"create pv and pvc: %s successful", sampleSetName)

	return utils.NoRequeue()
}

// reconcileBound After create PV/PVC then create runtime StatefulSet
func (s *SampleSetReconcilePhase) reconcileBound() (ctrl.Result, error) {
	s.Log.WithValues("phase", "reconcileBound")

	runtimeName := s.GetRuntimeName(s.Req.Name)
	serviceName := s.GetServiceName(s.Req.Name)
	runtimeKey := client.ObjectKey{
		Name: runtimeName,
		Namespace: s.Req.Namespace,
	}
	serviceKey := client.ObjectKey{
		Name: serviceName,
		Namespace: s.Req.Namespace,
	}
	statefulSetName := runtimeKey.String()
	rc := common.RequestContext{
		Req: s.Req,
		SampleSet: s.SampleSet,
	}

	// 1. create service for runtime StatefulSet
	service := &v1.Service{}
	if err := s.Get(s.Ctx, serviceKey, service); err != nil {
		if client.IgnoreNotFound(err) != nil {
			e := fmt.Errorf("get service %s error: %s", serviceName, err.Error())
			return utils.RequeueWithError(e)
		}
		// if service is not exist then create service for runtime server
		if err := s.CreateService(service, rc); err != nil {
			s.Log.Error(err, "create service error", "service", serviceName)
			s.Recorder.Event(s.SampleSet, v1.EventTypeWarning, common.ErrorCreateService, err.Error())
			return utils.NoRequeue()
		}
		if err := s.Create(s.Ctx, service); err != nil {
			e := fmt.Errorf("create service %s error: %s", serviceName, err.Error())
			s.Recorder.Event(s.SampleSet, v1.EventTypeWarning, common.ErrorCreateService, e.Error())
			return utils.RequeueWithError(e)
		}
		s.Log.V(1).Info("create service successful")

		// update SampleSet finalizer
		s.SampleSet.Finalizers = append(s.SampleSet.Finalizers, GetServiceFinalizer(s.Req.Name))
		if err := s.Update(s.Ctx, s.SampleSet); err != nil {
			e := fmt.Errorf("update sampleset %s error: %s", statefulSetName, err.Error())
			return utils.RequeueWithError(e)
		}
		s.Log.V(1).Info("update sampleset service finalizer successful")
		s.Recorder.Eventf(s.SampleSet, v1.EventTypeNormal, common.EventCreate,
			"create service: %s successful", serviceName)
		return utils.NoRequeue()
	}
	rc.Service = service

	// 2. check if StatefulSet is already exist
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
			return utils.NoRequeue()
		}

		// wait util at least one replica ready and update the phase of SampleSet to Mount
		if statefulSet.Status.ReadyReplicas > 0 {
			s.SampleSet.Status.Phase = common.SampleSetMount
			runtimeStatus := &v1alpha1.RuntimeStatus{
				SpecReplicas: statefulSet.Spec.Replicas,
				ReadyReplicas: &statefulSet.Status.ReadyReplicas,
				RuntimeReady: fmt.Sprintf("%d/%d",
					statefulSet.Status.ReadyReplicas,
					*statefulSet.Spec.Replicas,
				),
			}
			s.SampleSet.Status.RuntimeStatus = runtimeStatus
			if err := s.Status().Update(s.Ctx, s.SampleSet); err != nil {
				return utils.RequeueWithError(err)
			}
			s.Log.V(1).Info("update sampleset phase to mount")
			return utils.NoRequeue()
		}

		// check if the pod created by StatefulSet is work properly
		var eventList v1.EventList
		values := []string{"Pod", s.Req.Namespace, runtimeName+"-0", "Warning"}
		fOpt := client.MatchingFields{
			common.IndexerKeyEvent: strings.Join(values, "-"),
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
		// if the pod created by StatefulSet is not work properly then recorder event and requeue
		s.Recorder.Event(s.SampleSet, v1.EventTypeWarning, event.Reason, event.Message)
		return utils.RequeueWithError(errors.New(event.Message))
	}
	if client.IgnoreNotFound(err) != nil {
		e := fmt.Errorf("get statefulset %s error: %s", statefulSetName, err.Error())
		return utils.RequeueWithError(e)
	}

	// 3. get persistent volume and set to reconcile context
	pv := &v1.PersistentVolume{}
	volumeName := s.Req.Name
	volumeKey := client.ObjectKey{Name: volumeName}
	if err := s.Get(s.Ctx, volumeKey, pv); err != nil {
		e := fmt.Errorf("get pv %s error: %s", volumeName, err.Error())
		return utils.RequeueWithError(e)
	}
	rc.PV = pv

	// 4. create runtime StatefulSet
	if err := s.CreateRuntime(statefulSet, rc); err != nil {
		s.Log.Error(err, "create runtime error", "runtime", statefulSetName)
		s.Recorder.Event(s.SampleSet, v1.EventTypeWarning, common.ErrorCreateRuntime, err.Error())
		return utils.NoRequeue()
	}
	if err := s.Create(s.Ctx, statefulSet); err != nil {
		e := fmt.Errorf("create runtime %s error: %s", statefulSetName, err.Error())
		s.Recorder.Event(s.SampleSet, v1.EventTypeWarning, common.ErrorCreateRuntime, e.Error())
		return utils.RequeueWithError(e)
	}
	s.Log.V(1).Info("create runtime successful")

	// 5. add runtime finalizer to SampleSet and update SampleSet
	s.SampleSet.Finalizers = append(s.SampleSet.Finalizers, GetRuntimeFinalizer(volumeName))
	if err := s.Update(s.Ctx, s.SampleSet); err != nil {
		e := fmt.Errorf("update sampleset %s error: %s", statefulSetName, err.Error())
		return utils.RequeueWithError(e)
	}
	s.Log.V(1).Info("update sampleset runtime finalizer successful")

	s.Recorder.Eventf(s.SampleSet, v1.EventTypeNormal, common.EventCreate,
		"create statefulset: %s successful", statefulSetName)
	return utils.NoRequeue()
}

// reconcileMount After create runtime daemon set and mounted, before sync data job done
func (s *SampleSetReconcilePhase) reconcileMount() (ctrl.Result, error) {
	s.Log.WithValues("phase", "reconcileMount")

	runtimeName := s.GetRuntimeName(s.Req.Name)
	serviceName := s.GetServiceName(s.Req.Name)
	var syncJobName types.UID
	if s.SampleSet.Status.JobsName != nil {
		s.Log.Info("jobName is not nil....")
		syncJobName = s.SampleSet.Status.JobsName.SyncJobName
	}

	// 1. upload syncJobOptions to runtime server and trigger it to do sync data job
	if !s.SampleSet.Spec.NoSync && syncJobName == "" {
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

		options := &v1alpha1.SyncJobOptions{}
		if err := s.CreateSyncJobOptions(options, rc); err != nil {
			s.Log.Error(err, "create sync job options error")
			s.Recorder.Event(s.SampleSet, v1.EventTypeWarning, common.ErrorCreateSyncJobOption, err.Error())
			return utils.NoRequeue()
		}
		s.Log.Info("create sync job options successful")

		syncJobName = uuid.NewUUID()
		baseUri := utils.GetBaseUri(runtimeName, serviceName, 0)
		if err := utils.PostJobOptions(options, syncJobName, baseUri, common.PathSyncOptions); err != nil {
			e := fmt.Errorf("post sync job options error: %s", err.Error())
			s.Recorder.Event(s.SampleSet, v1.EventTypeWarning, common.ErrorUploadSyncJobOption, e.Error())
			return utils.RequeueWithError(err)
		}
		s.Log.Info("post sync job successfully")

		// Add jobsName to SampleSet
		if s.SampleSet.Status.JobsName == nil {
			s.SampleSet.Status.JobsName = &v1alpha1.JobsName{}
		}
		s.SampleSet.Status.JobsName.SyncJobName = syncJobName
		s.SampleSet.Status.Phase = common.SampleSetSyncing

		if err := s.Status().Update(s.Ctx, s.SampleSet); err != nil {
			e := fmt.Errorf("update sampleset %s error: %s", s.Req.Name, err.Error())
			return utils.RequeueWithError(e)
		}
		s.Log.Info("update sampleset phase to syncing")
		return utils.NoRequeue()
	}

	// wait util the first runtime server produce cache info file
	status, err := utils.CollectAllCacheStatus(runtimeName, serviceName, 1)
	if err != nil {
		s.Log.Error(err, "get cache status error")
		return utils.RequeueAfter(10 * time.Second)
	}
	if status.ErrorMassage != "" {
		s.Log.Error(errors.New(status.ErrorMassage), "error massage from cache status")
	}
	s.SampleSet.Status.CacheStatus = status
	s.Log.Info("collect all cache status successful")

	// get runtime StatefulSet
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
	// if StatefulSet ready replicas equal to SampleSet partitions, update phase to Ready otherwise PartialReady
	if statefulSet.Status.ReadyReplicas == s.SampleSet.Spec.Partitions {
		s.SampleSet.Status.Phase = common.SampleSetReady
	} else {
		s.SampleSet.Status.Phase = common.SampleSetPartialReady
	}

	if err := s.Update(s.Ctx, s.SampleSet); err != nil {
		e := fmt.Errorf("update sampleset %s error: %s", s.Req.Name, err.Error())
		return utils.RequeueWithError(e)
	}
	return utils.NoRequeue()
}

// reconcileSyncing wait the sync data job done and return to mount phase
func (s *SampleSetReconcilePhase) reconcileSyncing() (ctrl.Result, error) {
	s.Log.WithValues("phase", "reconcileSyncing")

	// 1. get the result of sync data job
	runtimeName := s.GetRuntimeName(s.Req.Name)
	serviceName := s.GetServiceName(s.Req.Name)
	baseUri := utils.GetBaseUri(runtimeName, serviceName, 0)
	filename := s.SampleSet.Status.JobsName.SyncJobName

	result := &common.JobResult{}
	err := utils.GetJobResult(result, filename, baseUri, common.PathSyncResult)
	if err != nil {
		e := fmt.Errorf("get sync job result error: %s", err.Error())
		return utils.RequeueWithError(e)
	}
	if result.Status == common.JobStatusRunning {
		s.Log.Info("wait util sync job done")
		return utils.RequeueAfter(30 * time.Second)
	}
	// 2. if sync job status is failed, then delete syncJobName
	if result.Status == common.JobStatusFail {
		s.SampleSet.Status.JobsName.SyncJobName = ""
		e := errors.New(result.Message)
		s.Log.Error(e, "sync job error", "jobName", filename)
		s.Recorder.Event(s.SampleSet, v1.EventTypeWarning, common.ErrorDoSyncJob, e.Error())
	}
	// 3. recorder event to SampleSet if success
	if result.Status == common.JobStatusSuccess {
		s.Log.Info("sync job success")
		s.Recorder.Event(s.SampleSet, v1.EventTypeNormal, common.EventCreate, "data is synchronized")
	}

	// 3. after sync job finish return SampleSet phase to Mount
	s.SampleSet.Status.Phase = common.SampleSetMount
	if err := s.Status().Update(s.Ctx, s.SampleSet); err != nil {
		e := fmt.Errorf("update sampleset %s error: %s", s.Req.Name, err.Error())
		return utils.RequeueWithError(e)
	}
	s.Log.Info("return SampleSet phase to mount")
	return utils.NoRequeue()
}

func (s *SampleSetReconcilePhase) reconcilePartialReady() (ctrl.Result, error) {
	s.Log.WithValues("phase", "reconcilePartialReady")
	return utils.NoRequeue()
}

// reconcileReady
func (s *SampleSetReconcilePhase) reconcileReady() (ctrl.Result, error) {
	s.Log.WithValues("phase", "reconcilePartialReady")

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
