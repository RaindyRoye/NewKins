package engine

import (
	"testing"
)

func TestHbtpRunnerAuthFun_Exists(t *testing.T) {
	runner := &HbtpRunner{}
	authFun := runner.AuthFun()

	// Just verify AuthFun returns a non-nil function
	if authFun == nil {
		t.Error("AuthFun should return a non-nil function")
	}
}
