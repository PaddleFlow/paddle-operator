package utils

import (
	"encoding/base64"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"reflect"
	ctrl "sigs.k8s.io/controller-runtime"
	"strings"
	"time"
)

// HasDeletionTimestamp method to check if an object need to delete.
func HasDeletionTimestamp(obj metav1.ObjectMeta) bool {
	return !obj.GetDeletionTimestamp().IsZero()
}

// HasFinalizer check
func HasFinalizer(obj metav1.ObjectMeta, finalizer string) bool {
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

func NoZeroOptionToMap(optionMap map[string]reflect.Value, i interface{}) {
	elem := reflect.ValueOf(i).Elem()
	for i := 0; i < elem.NumField(); i++ {
		value := elem.Field(i)
		if value.IsZero() {
			continue
		}
		field := elem.Type().Field(i)
		tag := field.Tag.Get("json")
		option := strings.Split(tag, ",")[0]
		optionMap[option] = value
	}
}

func Base64Decode(data []byte) (string, error) {
	s, err := base64.StdEncoding.DecodeString(string(data))
	return string(s), err
}

