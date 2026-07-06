package util

import (
	"runtime/debug"

	"github.com/sirupsen/logrus"
)

// RecoverLogWarn is a deferred recovery handler that logs panics at Warn level.
// Use this for non-critical background goroutines where a panic should be logged
// but not propagated. Typical usage:
//
//	defer util.RecoverLogWarn("BuildTask.run")
func RecoverLogWarn(label string) {
	if r := recover(); r != nil {
		logrus.Warnf("%s panic: %v", label, r)
		logrus.Warnf("%s stack:\n%s", label, string(debug.Stack()))
	}
}

// RecoverLogError is a deferred recovery handler that logs panics at Error level.
// Use this for critical operations where a panic indicates a serious problem.
// Typical usage:
//
//	defer util.RecoverLogError("BuildEngine.goroutine")
func RecoverLogError(label string) {
	if r := recover(); r != nil {
		logrus.Errorf("%s panic: %v", label, r)
		logrus.Errorf("%s stack:\n%s", label, string(debug.Stack()))
	}
}
