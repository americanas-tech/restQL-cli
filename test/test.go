package test

import (
	"reflect"
	"testing"
)

func Equal(t *testing.T, got, expected interface{}) {
	if !reflect.DeepEqual(got, expected) {
		t.Fatalf("got = %+#v, want = %+#v", got, expected)
	}
}
