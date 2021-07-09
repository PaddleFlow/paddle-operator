package utils

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// HasDeletionTimestamp method to check if an object need to delete.
func HasDeletionTimestamp(obj metav1.ObjectMeta) bool {
	return !obj.GetDeletionTimestamp().IsZero()
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