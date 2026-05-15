package backendconfigpolicy

import (
	"testing"
	"time"

	envoyclusterv3 "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/wrapperspb"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/kgateway-dev/kgateway/v2/pkg/pluginsdk/ir"
	"github.com/kgateway-dev/kgateway/v2/pkg/pluginsdk/policy"
)

// TestMergeOlderWinsPerField verifies AugmentedShallowMerge semantics for BCP:
// where the older policy sets a field, it wins; where it doesn't, the newer
// policy fills it in.
func TestMergeOlderWinsPerField(t *testing.T) {
	older := &BackendConfigPolicyIR{
		ct:             time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
		connectTimeout: durationpb.New(5 * time.Second),
		circuitBreakers: &envoyclusterv3.CircuitBreakers{
			Thresholds: []*envoyclusterv3.CircuitBreakers_Thresholds{
				{MaxConnections: &wrapperspb.UInt32Value{Value: 100}},
			},
		},
	}
	newer := &BackendConfigPolicyIR{
		ct:             time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
		connectTimeout: durationpb.New(99 * time.Second),
		circuitBreakers: &envoyclusterv3.CircuitBreakers{
			Thresholds: []*envoyclusterv3.CircuitBreakers_Thresholds{
				{MaxConnections: &wrapperspb.UInt32Value{Value: 999}},
			},
		},
		perConnectionBufferLimitBytes: new(uint32(2048)),
	}

	gk := schema.GroupKind{Group: "gateway.kgateway.dev", Kind: "BackendConfigPolicy"}
	pols := []ir.PolicyAtt{
		{GroupKind: gk, PolicyRef: &ir.AttachedPolicyRef{Name: "older"}, PolicyIr: older},
		{GroupKind: gk, PolicyRef: &ir.AttachedPolicyRef{Name: "newer"}, PolicyIr: newer},
	}

	merged := policy.MergePolicies(pols, mergeBackendConfigPolicies, "")
	got, ok := merged.PolicyIr.(*BackendConfigPolicyIR)
	if !ok {
		t.Fatalf("merged PolicyIr is not *BackendConfigPolicyIR")
	}

	assert.Equal(t, 5*time.Second, got.connectTimeout.AsDuration(), "older connectTimeout should win")
	assert.Equal(t, uint32(100), got.circuitBreakers.GetThresholds()[0].GetMaxConnections().GetValue(), "older circuitBreakers should win")
	if assert.NotNil(t, got.perConnectionBufferLimitBytes, "newer perConnectionBufferLimitBytes should be filled in") {
		assert.Equal(t, uint32(2048), *got.perConnectionBufferLimitBytes)
	}
}

// TestMergeFillsUnsetFields verifies that fields unset in p1 get filled in by p2.
func TestMergeFillsUnsetFields(t *testing.T) {
	older := &BackendConfigPolicyIR{
		ct:             time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
		connectTimeout: durationpb.New(3 * time.Second),
		// circuitBreakers and upstreamProxyProtocol unset
	}
	newer := &BackendConfigPolicyIR{
		ct: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
		circuitBreakers: &envoyclusterv3.CircuitBreakers{
			Thresholds: []*envoyclusterv3.CircuitBreakers_Thresholds{
				{MaxConnections: &wrapperspb.UInt32Value{Value: 200}},
			},
		},
	}

	gk := schema.GroupKind{Group: "gateway.kgateway.dev", Kind: "BackendConfigPolicy"}
	merged := policy.MergePolicies([]ir.PolicyAtt{
		{GroupKind: gk, PolicyRef: &ir.AttachedPolicyRef{Name: "older"}, PolicyIr: older},
		{GroupKind: gk, PolicyRef: &ir.AttachedPolicyRef{Name: "newer"}, PolicyIr: newer},
	}, mergeBackendConfigPolicies, "")

	got := merged.PolicyIr.(*BackendConfigPolicyIR)
	assert.Equal(t, 3*time.Second, got.connectTimeout.AsDuration())
	assert.Equal(t, uint32(200), got.circuitBreakers.GetThresholds()[0].GetMaxConnections().GetValue())
}
