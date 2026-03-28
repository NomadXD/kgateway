//go:build e2e

package faultinjection

import (
	"path/filepath"

	"github.com/kgateway-dev/kgateway/v2/pkg/utils/fsutils"
	"github.com/kgateway-dev/kgateway/v2/test/e2e/tests/base"
)

var (
	// manifests
	serviceManifest    = filepath.Join(fsutils.MustGetThisDir(), "testdata", "service.yaml")
	httpRoutesManifest = filepath.Join(fsutils.MustGetThisDir(), "testdata", "httproutes.yaml")
	faultAbortManifest = filepath.Join(fsutils.MustGetThisDir(), "testdata", "tp-fault-abort.yaml")
	faultDelayManifest = filepath.Join(fsutils.MustGetThisDir(), "testdata", "tp-fault-delay.yaml")

	setup = base.TestCase{
		Manifests: []string{serviceManifest},
	}

	testCases = map[string]*base.TestCase{
		"TestFaultInjectionAbortOnRoute": {
			Manifests: []string{httpRoutesManifest, faultAbortManifest},
		},
		"TestFaultInjectionAbortDoesNotAffectOtherRoutes": {
			Manifests: []string{httpRoutesManifest, faultAbortManifest},
		},
		"TestFaultInjectionDelayOnRoute": {
			Manifests: []string{httpRoutesManifest, faultDelayManifest},
		},
	}
)
