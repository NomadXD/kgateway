//go:build e2e

package faultinjection

import (
	"path/filepath"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kgateway-dev/kgateway/v2/pkg/utils/fsutils"
	"github.com/kgateway-dev/kgateway/v2/test/e2e/tests/base"
)

var (
	gatewayObjectMeta = metav1.ObjectMeta{
		Name:      "gateway",
		Namespace: "kgateway-base",
	}

	// manifests
	serviceManifest           = filepath.Join(fsutils.MustGetThisDir(), "testdata", "service.yaml")
	httpRoutesManifest        = filepath.Join(fsutils.MustGetThisDir(), "testdata", "httproutes.yaml")
	faultAbortManifest        = filepath.Join(fsutils.MustGetThisDir(), "testdata", "tp-fault-abort.yaml")
	faultDelayManifest        = filepath.Join(fsutils.MustGetThisDir(), "testdata", "tp-fault-delay.yaml")
	faultAbortGatewayManifest = filepath.Join(fsutils.MustGetThisDir(), "testdata", "tp-fault-abort-gateway.yaml")
	faultDisableRouteManifest = filepath.Join(fsutils.MustGetThisDir(), "testdata", "tp-fault-disable-route.yaml")

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
		"TestFaultInjectionAbortOnGateway": {
			Manifests: []string{httpRoutesManifest, faultAbortGatewayManifest},
		},
		"TestFaultInjectionAbortOnGatewayAffectsAllRoutes": {
			Manifests: []string{httpRoutesManifest, faultAbortGatewayManifest},
		},
		"TestFaultInjectionDisableOverridesGatewayPolicy": {
			Manifests: []string{httpRoutesManifest, faultAbortGatewayManifest, faultDisableRouteManifest},
		},
		"TestFaultInjectionDisableDoesNotAffectOtherRoutes": {
			Manifests: []string{httpRoutesManifest, faultAbortGatewayManifest, faultDisableRouteManifest},
		},
	}
)
