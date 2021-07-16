package utils

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"time"
)

// HasDeletionTimestamp method to check if an object need to delete.
func HasDeletionTimestamp(obj metav1.ObjectMeta) bool {
	return !obj.GetDeletionTimestamp().IsZero()
}

// HasNotFinalizer check
func HasNotFinalizer(obj metav1.ObjectMeta, finalizer string) bool {
	return ContainsString(obj.GetFinalizers(), finalizer)
}

func NoRequeue() (ctrl.Result, error) {
	return ctrl.Result{}, nil
}

func NoRequeueWithError(err error) (ctrl.Result, error) {
	return ctrl.Result{}, err
}

func RequeueImmediately() (ctrl.Result, error) {
	return ctrl.Result{Requeue: true}, nil
}

func RequeueAfter(requeueAfter time.Duration) (ctrl.Result, error) {
	return ctrl.Result{RequeueAfter: requeueAfter}, nil

}

// ContainsString Determine whether the string array contains a specific string, return true if contains the string and return false if not.
func ContainsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

func RemoveString(slice []string, s string) (result []string) {
	for _, item := range slice {
		if item == s {
			continue
		}
		result = append(result, item)
	}
	return
}

func IsEmpty() bool {
	return false
}
