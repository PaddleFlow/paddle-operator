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
	// 1. Get SampleSet Object
	var sampleSet v1alpha1.SampleSet
	if err := r.Get(ctx, req.NamespacedName, &sampleSet); err != nil {
		r.Log.V(1).Info("SampleSet ", req.NamespacedName, "not found")
		return utils.NoRequeueWithError(err)
	}

	// 2. Add Finalizer
	if r.finalizer(ctx, &sampleSet) {
		return utils.RequeueImmediately()
	}

	// 3. Get Driver and construct reconcile context
	cacheDriver, err := driver.GetDriver(sampleSet.Spec.CSI.Driver)
	if err != nil {
		return utils.NoRequeueWithError(nil)
	}
	RCtx := common.ReconcileContext{
		Ctx:      ctx,
		Client:   r.Client,
		Req:      &req,
		Log:      r.Log,
		Scheme:   r.Scheme,
		Recorder: r.Recorder,
		Driver:   cacheDriver,
	}

	//
	if utils.HasDeletionTimestamp(sampleSet.ObjectMeta) {
		return r.deleteSampleSet(&RCtx, &sampleSet)
	}

	ssrp := SampleSetReconcilePhase{&RCtx, &sampleSet}

	return ssrp.reconcilePhase()
}

func (r *SampleSetReconciler) finalizer(ctx context.Context, sampleSet *v1alpha1.SampleSet) bool {
	if !utils.HasNotFinalizer(sampleSet.ObjectMeta, FINALIZER) { return false }
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

	return utils.NoRequeue()
}


// SetupWithManager sets up the controller with the Manager.
func (r *SampleSetReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.SampleSet{}).
		Complete(r)
}

type SampleSetReconcilePhase struct {
	*common.ReconcileContext
	*v1alpha1.SampleSet
}

func (s *SampleSetReconcilePhase) reconcilePhase() (ctrl.Result, error) {
	switch s.SampleSet.Status.Phase {
	case common.SampleSetNone:
		return s.reconcilePhase()
	case common.SampleSetFailed:

	}
	return ctrl.Result{}, nil
}

// reconcileNone 1. create PV/Storage Class/PVC
func (s *SampleSetReconcilePhase) reconcileNone() (ctrl.Result, error) {
	s.Log.WithName("reconcileNone")

	// 1. check if persistent volume is already exists which have same name as sampleset
	pv := &v1.PersistentVolume{}
	err := s.Get(s.Ctx, s.Req.NamespacedName, pv)
	if client.IgnoreNotFound(err) != nil {
		s.Log.V(1).Error(err,"get pv error")
		return utils.NoRequeueWithError(err)
	}
	if err == nil {
		_, exist := pv.ObjectMeta.Labels[common.PaddleLabel]
		if !exist {
			exitsErr := fmt.Errorf("PV %s is already exists", s.Req.NamespacedName)
			s.Log.V(1).Error(exitsErr,"create pv error")
			return utils.NoRequeueWithError(exitsErr)
		}
		// if pv is already create by paddle-operator
		if pv.Status.Phase != v1.VolumeBound {
			return utils.RequeueAfter(1 * time.Second)
		}
		// if pv is bounded then update the sampleset's phase
		s.SampleSet.Status.Phase = common.SampleSetBound
		if upErr := s.Client.Update(s.Ctx, s.SampleSet); upErr != nil {
			s.Log.V(1).Error(upErr,"update SampleSet phase error")
			return utils.NoRequeueWithError(upErr)
		}
		return utils.RequeueImmediately()
	}

	// 2. check if secret name is none or secret not exists
	secret := &v1.Secret{}
	if secretName := s.SampleSet.Spec.SecretRef.Name; secretName != "" {
		key := client.ObjectKey{
			Name: secretName,
			Namespace: s.Req.Namespace,
		}
		if err := s.Get(s.Ctx, key, secret); err != nil {
			s.Log.V(1).Error(err, "get secret error")
			s.Recorder.Eventf(s.SampleSet, v1.EventTypeWarning,
				common.ErrorSecretNotExists, err.Error())
			return utils.NoRequeueWithError(err)
		}
	} else {
		err := fmt.Errorf("secretRef is not set")
		s.Log.V(1).Error(err,"get secret error")
		s.Recorder.Eventf(s.SampleSet, v1.EventTypeWarning,
			common.ErrorSecretNotExists, err.Error())
		return utils.NoRequeueWithError(err)
	}

	// construct request context
	rc := common.RequestContext{Log: s.Log, Req: s.Req,
		SampleSet: s.SampleSet, Secret: secret}

	// 3. create persistent volume and set its name as the SampleSet
	if err := s.CreatePV(pv, rc); err != nil {
		s.Log.V(1).Error(err,"error occur then create pv")
		return utils.NoRequeueWithError(err)
	}

	// 4. check if persistent volume claim is exists and have same name as sampleset
	pvc := &v1.PersistentVolumeClaim{}
	err = s.Get(s.Ctx, s.Req.NamespacedName, pvc)
	if client.IgnoreNotFound(err) != nil {
		s.Log.V(1).Error(err,"get pvc error")
		return utils.NoRequeueWithError(err)
	}
	if err == nil {
		_, exists := pvc.ObjectMeta.Labels[common.PaddleLabel]
		if !exists {
			exitsErr := fmt.Errorf("PVC %s is already exists", s.Req.NamespacedName)
			s.Log.V(1).Error(exitsErr,"create pvc error")
			return utils.NoRequeueWithError(exitsErr)
		}
		return utils.RequeueAfter(1 * time.Second)
	}

	// 5. create persistent volume claim and set its name as the SampleSet
	if err := s.CreatePVC(pvc, rc); err != nil {
		s.Log.V(1).Error(err,"error occur then create pvc")
		return utils.NoRequeueWithError(err)
	}

	return utils.RequeueAfter(1 * time.Second)
}

func (s *SampleSetReconcilePhase) reconcileBound() (ctrl.Result, error) {
	return ctrl.Result{}, nil
}
