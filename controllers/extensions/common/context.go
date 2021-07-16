package common

import (
	"context"
	"github.com/go-logr/logr"
	"github.com/paddleflow/paddle-operator/api/v1alpha1"
	"github.com/paddleflow/paddle-operator/controllers/extensions/driver"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ReconcileContext struct {
	//
	client.Client

	//
	driver.Driver

	//
	Ctx context.Context

	//
	Req *ctrl.Request

	//
	Log logr.Logger

	//
	Scheme *runtime.Scheme

	//
	Recorder record.EventRecorder
}

type RequestContext struct {
	//
	Log logr.Logger

	//
	Req *ctrl.Request

	//
	SampleSet *v1alpha1.SampleSet

	//
	Secret *v1.Secret
}