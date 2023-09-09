package test

import (
	"fmt"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"
)

func Equals(t *testing.T, exp, act interface{}) {
	if !reflect.DeepEqual(exp, act) {
		_, file, line, _ := runtime.Caller(1)
		fmt.Printf("\033[31m%s:%d:\n\n\tgot: %#v\n\n\texp: %#v\033[39m\n\n", filepath.Base(file), line, act, exp)
		t.FailNow()
	}
}
