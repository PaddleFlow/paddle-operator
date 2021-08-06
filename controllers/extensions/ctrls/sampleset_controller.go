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
	"reflect"
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
//+kubebuilder:rbac:groups=batch.paddlepaddle.org,resources=samplejobs,verbs=get;list;watch
//+kubebuilder:rbac:groups=batch.paddlepaddle.org,resources=paddlejobs,verbs=get;list;watch
//+kubebuilder:rbac:groups="",resources=secrets,verbs=get
//+kubebuilder:rbac:groups="",resources=persistentvolumes,verbs=get;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=persistentvolumeclaims,verbs=get;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=services,verbs=get;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=statefulsets,verbs=get;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=nodes,verbs=get;list;update;patch
//+kubebuilder:rbac:groups="",resources=pods,verbs=get;list
//+kubebuilder:rbac:groups="",resources=events,verbs=get;list

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
	driver.Driver
	*common.ReconcileContext
	SampleSet *v1alpha1.SampleSet
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
	case common.SampleSetSyncFailed:
		return s.reconcileSyncFailed()
	case common.SampleSetPartialReady:
		return s.reconcilePartialReady()
	case common.SampleSetReady:
		return s.reconcileReady()
	}
	s.Log.Error(fmt.Errorf("phase %s not support", s.SampleSet.Status.Phase), "")
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

	// 4. check if persistent volume claim is exist and named as the SampleSet
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

		// wait util at least one replica ready and update the phase of SampleSet to Mount
		if statefulSet.Status.ReadyReplicas > 0 {
			s.SampleSet.Status.Phase = common.SampleSetMount
			runtimeStatus := &v1alpha1.RuntimeStatus{
				SpecReplicas: s.SampleSet.Spec.Partitions,
				ReadyReplicas: statefulSet.Status.ReadyReplicas,
				RuntimeReady: fmt.Sprintf("%d/%d",
					statefulSet.Status.ReadyReplicas,
					s.SampleSet.Spec.Partitions,
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
		if err := utils.PostJobOption(options, syncJobName, baseUri, common.PathSyncOptions); err != nil {
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

	// 2. wait util the first runtime server produce cache info file
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

	// 3. update SampleSet phase to partial ready
	s.SampleSet.Status.Phase = common.SampleSetPartialReady
	if err := s.Status().Update(s.Ctx, s.SampleSet); err != nil {
		e := fmt.Errorf("update sampleset %s error: %s", s.Req.Name, err.Error())
		return utils.RequeueWithError(e)
	}
	s.Log.Info("update sampleset phase to partial ready")

	return utils.NoRequeue()
}

// reconcileSyncing wait the sync data job done and return to mount phase
func (s *SampleSetReconcilePhase) reconcileSyncing() (ctrl.Result, error) {
	s.Log.WithValues("phase", "reconcileSyncing")
	runtimeName := s.GetRuntimeName(s.Req.Name)
	serviceName := s.GetServiceName(s.Req.Name)
	baseUri := utils.GetBaseUri(runtimeName, serviceName, 0)
	filename := s.SampleSet.Status.JobsName.SyncJobName

	// 1. get the status of cache status from runtime server 0
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

	// 2. get the result of sync data job
	result, err := utils.GetJobResult(filename, baseUri, common.PathSyncResult)
	if err != nil {
		e := fmt.Errorf("get sync job result error: %s", err.Error())
		return utils.RequeueWithError(e)
	}
	if result.Status == common.JobStatusRunning {
		s.Log.Info("wait util sync job done")
		return utils.RequeueAfter(30 * time.Second)
	}

	// 3. if sync job status is failed, then update phase to SyncFailed
	if result.Status == common.JobStatusFail {
		e := errors.New(result.Message)
		s.SampleSet.Status.Phase = common.SampleSetSyncFailed
		s.Log.Error(e, "sync job error", "jobName", filename)
		s.Recorder.Event(s.SampleSet, v1.EventTypeWarning, common.ErrorDoSyncJob, e.Error())
	}
	// 3. if sync job status is success, then return phase to mount
	if result.Status == common.JobStatusSuccess {
		s.Log.Info("sync job success")
		s.SampleSet.Status.Phase = common.SampleSetMount
		s.Recorder.Event(s.SampleSet, v1.EventTypeNormal, common.EventCreate, "data is synchronized")
	}
	// 4. update SampleSet phase
	if err := s.Status().Update(s.Ctx, s.SampleSet); err != nil {
		e := fmt.Errorf("update sampleset %s error: %s", s.Req.Name, err.Error())
		return utils.RequeueWithError(e)
	}
	s.Log.Info("return SampleSet phase to mount")
	return utils.NoRequeue()
}

// reconcileReady reconcile
func (s *SampleSetReconcilePhase) reconcileReady() (ctrl.Result, error) {
	s.Log.WithValues("phase", "reconcilePartialReady")
	runtimeName := s.GetRuntimeName(s.Req.Name)
	serviceName := s.GetServiceName(s.Req.Name)

	// 1. check whether spec.partitions is changed, if partitions is changed,
	// update the replicas of StatefulSet and update SampleSet phase to partial ready.
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
	specReplicas := *statefulSet.Spec.Replicas
	readyReplicas := statefulSet.Status.ReadyReplicas
	partitions := s.SampleSet.Spec.Partitions
	// if runtime server ready replicas not equal SampleSet spec partitions, then update phase to partial ready
	if specReplicas != partitions || readyReplicas != partitions {
		s.SampleSet.Status.Phase = common.SampleSetPartialReady
		if err := s.Status().Update(s.Ctx, s.SampleSet); err != nil {
			e := fmt.Errorf("update sampleset %s error: %s", s.Req.Name, err.Error())
			return utils.RequeueWithError(e)
		}
		s.Log.Info("update sampleset phase to partial ready")
	}

	// 2. get the cache status from runtime servers
	newStatus, err := utils.CollectAllCacheStatus(runtimeName, serviceName, int(partitions))
	if err != nil {
		s.Log.Error(err, "get cache status error")
		return utils.RequeueAfter(10 * time.Second)
	}
	if newStatus.ErrorMassage != "" {
		s.Log.Error(errors.New(newStatus.ErrorMassage), "error massage from cache status")
	}
	s.Log.Info("collect all cache status successful")
	if !reflect.DeepEqual(newStatus, s.SampleSet.Status.CacheStatus) {
		s.SampleSet.Status.CacheStatus = newStatus
		if err := s.Status().Update(s.Ctx, s.SampleSet); err != nil {
			e := fmt.Errorf("update sampleset %s error: %s", s.Req.Name, err.Error())
			return utils.RequeueWithError(e)
		}
		s.Log.Info("updated sampleset cache status")
	}

	return utils.NoRequeue()
}
