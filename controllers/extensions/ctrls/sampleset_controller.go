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
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"
)

var (
	FINALIZER = "finalizer.sampleset.paddlepaddle.org"
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
	r.Log.V(1).Info("trigger by ","object" , req.NamespacedName.String())

	// 1. Get SampleSet Object
	var sampleSet v1alpha1.SampleSet
	if err := r.Get(ctx, req.NamespacedName, &sampleSet); err != nil {
		r.Log.V(1).Info("sampleset not find", "sampleset", req.NamespacedName.String())
		return utils.NoRequeueWithError(err)
	}

	// 2. Add Finalizer
	if r.finalizer(ctx, &sampleSet) {
		return utils.RequeueImmediately()
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
		return utils.NoRequeueWithError(nil)
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
	if utils.HasDeletionTimestamp(sampleSet.ObjectMeta) {
		return r.deleteSampleSet(&RCtx, &sampleSet)
	}

	ssrp := SampleSetReconcilePhase{&RCtx, &sampleSet, CSIDriver}

	return ssrp.reconcilePhase()
}

func (r *SampleSetReconciler) finalizer(ctx context.Context, sampleSet *v1alpha1.SampleSet) bool {
	if utils.HasFinalizer(sampleSet.ObjectMeta, FINALIZER) { return false }
	if utils.HasDeletionTimestamp(sampleSet.ObjectMeta) { return false }

	sampleSet.ObjectMeta.Finalizers = append(sampleSet.ObjectMeta.Finalizers, FINALIZER)
	if err := r.Update(ctx, sampleSet); err != nil {
		r.Log.Error(err, "update sampleset error when add finalizer")
		return false
	}
	return true
}

func (r *SampleSetReconciler) deleteSampleSet(
	ctx *common.ReconcileContext,
	sampleSet *v1alpha1.SampleSet) (ctrl.Result, error) {

	// 判断是否有PaddleJob运行

	// 1. 终止正在运行的SampleJob任务, 提交SampleJob deleteCache 任务, 并禁止提交其他的

	// 2. 等待deleteCache任务完成以后, 删除缓存节点上的Labels, 删除 Runtime DaemonSet

	// 3. 删除pvc / pv 等资源

	return utils.NoRequeue()
}


// SetupWithManager sets up the controller with the Manager.
func (r *SampleSetReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.SampleSet{}).
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
	switch s.SampleSet.Status.Phase {
	case common.SampleSetNone:
		return s.reconcileNone()
	case common.SampleSetBound:
		return s.reconcileBound()
	case common.SampleSetMount:
		return s.reconcileMount()
	case common.SampleSetReady:
		return s.reconcileReady()
	}
	return utils.NoRequeue()
}

// reconcileNone After user create SampleSet CR then create PV and PVC automatically
func (s *SampleSetReconcilePhase) reconcileNone() (ctrl.Result, error) {
	s.Log.WithName("reconcileNone")
	volumeKey := client.ObjectKey{Name: s.Req.Name}
	volumeName := volumeKey.String()

	// 1. check if persistent volume is already exists which have same name as SampleSet
	pv := &v1.PersistentVolume{}
	err := s.Get(s.Ctx, volumeKey, pv)
	if client.IgnoreNotFound(err) != nil {
		err := fmt.Errorf("get pv %s error: %s", volumeName, err.Error())
		return utils.NoRequeueWithError(err)
	}
	s.Log.V(1).Info("========error: ", "err", err)
	if err == nil {
		label := s.GetLabel(s.SampleSet.ObjectMeta.Name)
		if _, exist := pv.ObjectMeta.Labels[label]; !exist {
			exitsErr := fmt.Errorf("pv %s is already exists", volumeName)
			s.Recorder.Eventf(s.SampleSet, v1.EventTypeWarning, common.ErrorPVAlreadyExists, exitsErr.Error())
			return utils.NoRequeueWithError(exitsErr)
		}
		// if pv is already create by paddle-operator
		if pv.Status.Phase != v1.VolumeBound {
			s.Log.V(1).Info("requeue and wait pv Bound", "pv", volumeKey)
			return utils.RequeueAfter(1 * time.Second)
		}
		// if pv is bounded then update the sampleset's phase
		s.SampleSet.Status.Phase = common.SampleSetBound
		if upErr := s.Status().Update(s.Ctx, s.SampleSet); upErr != nil {
			return utils.NoRequeueWithError(upErr)
		}
		s.Log.V(1).Info("update SampleSet phase to Bound", "SampleSet", volumeKey)
		return utils.RequeueAfter(1 * time.Second)
	}
	s.Log.V(1).Info("pv is not exists", "pv", volumeKey)

	// 2. check if secret name is none or secret not exists
	if s.SampleSet.Spec.SecretRef == nil {
		err := errors.New("secretRef is empty")
		s.Recorder.Eventf(s.SampleSet, v1.EventTypeWarning, common.ErrorSecretNotExists, err.Error())
		return utils.NoRequeueWithError(err)
	}

	secret := &v1.Secret{}
	if secretName := s.SampleSet.Spec.SecretRef.Name; secretName != "" {
		key := client.ObjectKey{
			Name: secretName,
			Namespace: s.Req.Namespace,
		}
		if err := s.Get(s.Ctx, key, secret); err != nil {
			e := fmt.Errorf("get secret %s error: %s", secretName, err.Error())
			s.Recorder.Eventf(s.SampleSet, v1.EventTypeWarning, common.ErrorSecretNotExists, e.Error())
			return utils.NoRequeueWithError(e)
		}
	} else {
		err := errors.New("secretRef name is not set")
		s.Recorder.Eventf(s.SampleSet, v1.EventTypeWarning, common.ErrorSecretNotExists, err.Error())
		return utils.NoRequeueWithError(err)
	}
	s.Log.V(1).Info("secret is find", "secret", secret.ObjectMeta.Name)

	// construct request context
	rc := common.RequestContext{Log: s.Log, Req: s.Req,
		SampleSet: s.SampleSet, Secret: secret}

	// 3. create persistent volume and set its name as the SampleSet
	if err := s.CreatePV(pv, rc); err != nil {
		e := fmt.Errorf("create pv %s error: %s", volumeKey, err.Error())
		s.Recorder.Eventf(s.SampleSet, v1.EventTypeWarning, common.ErrorCreatePV, e.Error())
		return utils.NoRequeueWithError(e)
	}
	if err := s.Create(s.Ctx, pv); err != nil {
		e := fmt.Errorf("create pv %s error: %s", volumeKey, err.Error())
		s.Recorder.Eventf(s.SampleSet, v1.EventTypeWarning, common.ErrorCreatePV, e.Error())
		return utils.NoRequeueWithError(e)
	}
	s.Log.V(1).Info("create pv successfully", "pv", volumeKey)

	// 4. check if persistent volume claim is exists and have same name as SampleSet
	pvc := &v1.PersistentVolumeClaim{}
	err = s.Get(s.Ctx, volumeKey, pvc)
	if client.IgnoreNotFound(err) != nil {
		e := fmt.Errorf("get pvc error: %s", err.Error())
		return utils.NoRequeueWithError(e)
	}
	if err == nil {
		label := s.GetLabel(s.SampleSet.ObjectMeta.Name)
		if _, exists := pvc.ObjectMeta.Labels[label]; !exists {
			e := fmt.Errorf("pvc %s is already exists", volumeKey)
			s.Recorder.Eventf(s.SampleSet, v1.EventTypeWarning, common.ErrorPVCAlreadyExists, e.Error())
			return utils.NoRequeueWithError(e)
		}
		return utils.RequeueAfter(1 * time.Second)
	}
	s.Log.V(1).Info("pvc is not exists", "pvc", volumeKey)

	// 5. create persistent volume claim and set its name as the SampleSet
	if err := s.CreatePVC(pvc, rc); err != nil {
		e := fmt.Errorf("create pvc %s error: %s", volumeKey, err.Error())
		s.Recorder.Eventf(s.SampleSet, v1.EventTypeWarning, common.ErrorCreatePVC, e.Error())
		return utils.NoRequeueWithError(e)
	}
	if err := s.Create(s.Ctx, pvc); err != nil {
		e := fmt.Errorf("create pvc %s error: %s", volumeKey, err.Error())
		s.Recorder.Eventf(s.SampleSet, v1.EventTypeWarning, common.ErrorCreatePV, e.Error())
		return utils.NoRequeueWithError(e)
	}

	s.Log.V(1).Info("create pvc successfully and requeue", "pvc", volumeKey)
	s.Recorder.Eventf(s.SampleSet, v1.EventTypeNormal, "", "create pv and pvc successfully")
	return utils.RequeueAfter(1 * time.Second)
}

func (s *SampleSetReconcilePhase) reconcileBound() (ctrl.Result, error) {
	s.Log.Info("*****************")
	return utils.NoRequeue()
	//return utils.RequeueAfter(1 * time.Second)
}

func (s *SampleSetReconcilePhase) reconcileMount() (ctrl.Result, error) {
	return utils.RequeueAfter(20 * time.Second)
}

func (s *SampleSetReconcilePhase) reconcileReady() (ctrl.Result, error) {
	return utils.NoRequeue()
}
