package manager

import (
	"fmt"
	"k8s.io/apimachinery/pkg/util/uuid"
	"testing"
)

func TestUUID (t *testing.T) {

	uid := uuid.NewUUID()

	t.Log(fmt.Sprintf("uuid %s", uid))
}