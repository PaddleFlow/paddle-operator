package common

import (
	"context"
	"github.com/go-logr/logr"
	"github.com/paddleflow/paddle-operator/controllers/extensions/driver"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ReconcileContext struct {

	context.Context

	client.Client

	Log logr.Logger

	*runtime.Scheme

	Recorder record.EventRecorder

	driver.Driver
}
