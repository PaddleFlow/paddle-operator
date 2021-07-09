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
	"github.com/go-logr/logr"
	"github.com/paddleflow/paddle-operator/controllers/extensions/utils"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"

	ssv1alpha1 "github.com/paddleflow/paddle-operator/api/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	finalizerName = "finalizers.sampleset.paddlepaddle.org"
)

type ReconcileRequestContext struct {
	context.Context
	types.NamespacedName
	Log logr.Logger
	Recorder record.EventRecorder
	client.Client
	RuntimeType   string
	FinalizerName string
	SampleSet ssv1alpha1.SampleSet
}

// SampleSetReconciler reconciles a SampleSet object
type SampleSetReconciler struct {
	client.Client
	Log         logr.Logger
	Scheme      *runtime.Scheme
	Recorder    record.EventRecorder
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
	var sampleSet ssv1alpha1.SampleSet
	if err := r.Get(ctx, req.NamespacedName, &sampleSet); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	rCtx := ReconcileRequestContext{
		Context: ctx,
		Log: r.Log.WithValues("SampleSet", req.NamespacedName),
		NamespacedName: req.NamespacedName,
		Recorder: r.Recorder,
		RuntimeType: "juicefs",
		Client: r.Client,
		FinalizerName: finalizerName,
		SampleSet: sampleSet,
	}

	// 1. Check if need to delete sampleset
	if utils.HasDeletionTimestamp(sampleSet.ObjectMeta) {
		return r.reconcileDeletion(rCtx)
	}

	// 2. Add finalizer and requeue
	if utils.ContainsString(sampleSet.ObjectMeta.GetFinalizers(), finalizerName) {
		return r.addFinalizerAndRequeue(rCtx)
	}

	switch sampleSet.Status.Phase {
	case ssv1alpha1.Unknown:
		return r.reconcileUnknown(rCtx)
	case ssv1alpha1.Bound:
		return r.reconcileBound(rCtx)
	default:
		rCtx.Log.Info("")
	}

	return ctrl.Result{}, nil
}

func (r *SampleSetReconciler) addFinalizerAndRequeue(ctx ReconcileRequestContext) (ctrl.Result, error) {
	return ctrl.Result{}, nil
}

func (r *SampleSetReconciler) reconcileDeletion(ctx ReconcileRequestContext) (ctrl.Result, error) {
	return ctrl.Result{}, nil
}

func (r *SampleSetReconciler) reconcileUnknown(ctx ReconcileRequestContext) (ctrl.Result, error) {
	return ctrl.Result{}, nil
}

func (r *SampleSetReconciler) reconcileBound(ctx ReconcileRequestContext) (ctrl.Result, error) {
	return ctrl.Result{}, nil
}


// SetupWithManager sets up the controller with the Manager.
func (r *SampleSetReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&ssv1alpha1.SampleSet{}).
		Complete(r)
}
