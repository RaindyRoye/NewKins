package util

import (
	"fmt"
	"runtime/debug"

	"github.com/sirupsen/logrus"
)

// RecoverLog recovers from a panic and logs it with a stack trace at Warn level.
// Intended for use in deferred blocks:
//
//	defer util.RecoverLog("functionName")
//
// It replaces the common pattern:
//
//	defer func() {
//	    if err := recover(); err != nil {
//	        logrus.Warnf("label recover:%v", err)
//	        logrus.Warnf("stack:%s", string(debug.Stack()))
//	    }
//	}()
func RecoverLog(label string) {
	if r := recover(); r != nil {
		logrus.Warnf("%s recover: %v", label, r)
		logrus.Warnf("%s stack:\n%s", label, string(debug.Stack()))
	}
}

// RecoverLogf is like RecoverLog but logs at Error level.
// Use for critical goroutine panics that should not silently fail.
func RecoverLogf(label string) {
	if r := recover(); r != nil {
		logrus.Errorf("%s panic: %v", label, r)
		logrus.Errorf("%s stack:\n%s", label, string(debug.Stack()))
	}
}

// RecoverError recovers from a panic and assigns the result to *errp
// as a formatted error. Use with named return values:
//
//	func Do() (rterr error) {
//	    defer util.RecoverError(&rterr, "Do")
//	    ...
//	}
func RecoverError(errp *error, label string) {
	if r := recover(); r != nil {
		logrus.Warnf("%s panic: %v", label, r)
		logrus.Warnf("%s stack:\n%s", label, string(debug.Stack()))
		if e, ok := r.(error); ok {
			*errp = fmt.Errorf("%s: %w", label, e)
		} else {
			*errp = fmt.Errorf("%s: %v", label, r)
		}
	}
}
