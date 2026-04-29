//go:build e2e

package faultinjection

import (
	"context"
	"net/http"
	"time"

	"github.com/onsi/gomega"
	"github.com/stretchr/testify/suite"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/kgateway-dev/kgateway/v2/pkg/utils/requestutils/curl"
	"github.com/kgateway-dev/kgateway/v2/test/e2e"
	"github.com/kgateway-dev/kgateway/v2/test/e2e/common"
	"github.com/kgateway-dev/kgateway/v2/test/e2e/tests/base"
	"github.com/kgateway-dev/kgateway/v2/test/envoyutils/admincli"
	testmatchers "github.com/kgateway-dev/kgateway/v2/test/gomega/matchers"
)

var _ e2e.NewSuiteFunc = NewTestingSuite

// testingSuite is a suite of tests for testing fault injection policies
type testingSuite struct {
	*base.BaseTestingSuite
}

func NewTestingSuite(ctx context.Context, testInst *e2e.TestInstallation) suite.TestingSuite {
	return &testingSuite{
		BaseTestingSuite: base.NewBaseTestingSuite(ctx, testInst, setup, testCases),
	}
}

// BeforeTest overrides the base to wait for fault injection xDS config to propagate to
// Envoy after manifests are applied. Without this, the test races against xDS propagation
// and sees 200 instead of the injected fault response on slow clusters.
func (s *testingSuite) BeforeTest(suiteName, testName string) {
	s.BaseTestingSuite.BeforeTest(suiteName, testName)
	if _, ok := s.TestCases[testName]; ok {
		s.waitForFaultFilterInEnvoy()
	}
}

// waitForFaultFilterInEnvoy polls the Envoy admin config dump until the fault injection
// filter appears as an active typed_per_filter_config key (not just as a disabled listener
// filter). This proves the xDS route config update was received by Envoy.
func (s *testingSuite) waitForFaultFilterInEnvoy() {
	s.TestInstallation.AssertionsT(s.T()).AssertEnvoyAdminApi(
		s.Ctx,
		gatewayObjectMeta,
		func(ctx context.Context, adminClient *admincli.Client) {
			s.TestInstallation.AssertionsT(s.T()).Gomega.Eventually(func(g gomega.Gomega) {
				cfgDump, err := adminClient.GetConfigDump(ctx, nil)
				g.Expect(err).NotTo(gomega.HaveOccurred(), "failed to get Envoy config dump")

				cfgJSON, err := protojson.Marshal(cfgDump)
				g.Expect(err).NotTo(gomega.HaveOccurred(), "failed to marshal Envoy config dump")

				// In the JSON config dump, "envoy.filters.http.fault": appears as a map key
				// inside typedPerFilterConfig when the policy is active. This is distinct from
				// the listener declaration ("name": "envoy.filters.http.fault") which is always
				// present with disabled:true whenever any fault policy exists.
				g.Expect(string(cfgJSON)).To(gomega.ContainSubstring(`"envoy.filters.http.fault":`),
					"fault injection config should be active in Envoy route or vhost typed_per_filter_config")
			}).WithTimeout(30*time.Second).WithPolling(time.Second).
				Should(gomega.Succeed(), "fault injection xDS config should propagate to Envoy")
		},
	)
}

// TestFaultInjectionAbortOnRoute verifies that a TrafficPolicy with 100% abort at HTTP 503
// returns the configured status code on the targeted route.
func (s *testingSuite) TestFaultInjectionAbortOnRoute() {
	common.BaseGateway.Send(
		s.T(),
		&testmatchers.HttpResponse{
			StatusCode: http.StatusServiceUnavailable,
		},
		curl.WithPort(80),
		curl.WithPath("/fault/status/200"),
		curl.WithHostHeader("example.com"),
	)
}

// TestFaultInjectionAbortDoesNotAffectOtherRoutes verifies that a fault injection policy
// attached to one route does not affect other routes without the policy.
func (s *testingSuite) TestFaultInjectionAbortDoesNotAffectOtherRoutes() {
	common.BaseGateway.Send(
		s.T(),
		&testmatchers.HttpResponse{
			StatusCode: http.StatusOK,
		},
		curl.WithPort(80),
		curl.WithPath("/no-fault/status/200"),
		curl.WithHostHeader("example.com"),
	)
}

// TestFaultInjectionDelayOnRoute verifies that a TrafficPolicy with delay configured
// still returns a successful response (delay is applied but request is not aborted).
// It also verifies the response takes at least the configured delay duration.
func (s *testingSuite) TestFaultInjectionDelayOnRoute() {
	start := time.Now()

	common.BaseGateway.Send(
		s.T(),
		&testmatchers.HttpResponse{
			StatusCode: http.StatusOK,
		},
		curl.WithPort(80),
		curl.WithPath("/fault/status/200"),
		curl.WithHostHeader("example.com"),
	)

	elapsed := time.Since(start)
	s.GreaterOrEqual(elapsed, 100*time.Millisecond, "expected response to take at least 100ms due to fault delay injection")
}

// TestFaultInjectionAbortOnGateway verifies that a TrafficPolicy with abort attached
// to a Gateway returns the configured status code on a route through that gateway.
func (s *testingSuite) TestFaultInjectionAbortOnGateway() {
	common.BaseGateway.Send(
		s.T(),
		&testmatchers.HttpResponse{
			StatusCode: http.StatusServiceUnavailable,
		},
		curl.WithPort(80),
		curl.WithPath("/fault/status/200"),
		curl.WithHostHeader("example.com"),
	)
}

// TestFaultInjectionAbortOnGatewayAffectsAllRoutes verifies that a fault injection
// policy attached to a Gateway affects all routes on that gateway, not just one.
func (s *testingSuite) TestFaultInjectionAbortOnGatewayAffectsAllRoutes() {
	common.BaseGateway.Send(
		s.T(),
		&testmatchers.HttpResponse{
			StatusCode: http.StatusServiceUnavailable,
		},
		curl.WithPort(80),
		curl.WithPath("/no-fault/status/200"),
		curl.WithHostHeader("example.com"),
	)
}

// TestFaultInjectionDisableOverridesGatewayPolicy verifies that a route-level
// TrafficPolicy with faultInjection.disable overrides a gateway-level fault
// injection policy, allowing the route to respond normally.
func (s *testingSuite) TestFaultInjectionDisableOverridesGatewayPolicy() {
	common.BaseGateway.Send(
		s.T(),
		&testmatchers.HttpResponse{
			StatusCode: http.StatusOK,
		},
		curl.WithPort(80),
		curl.WithPath("/no-fault/status/200"),
		curl.WithHostHeader("example.com"),
	)
}

// TestFaultInjectionDisableDoesNotAffectOtherRoutes verifies that a route-level
// disable does not affect other routes that should still have the gateway-level
// fault injection applied.
func (s *testingSuite) TestFaultInjectionDisableDoesNotAffectOtherRoutes() {
	common.BaseGateway.Send(
		s.T(),
		&testmatchers.HttpResponse{
			StatusCode: http.StatusServiceUnavailable,
		},
		curl.WithPort(80),
		curl.WithPath("/fault/status/200"),
		curl.WithHostHeader("example.com"),
	)
}
