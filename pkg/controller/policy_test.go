package controller

import (
	"fmt"
	"testing"

	"github.com/bottlerocket-os/bottlerocket-update-operator/pkg/intent"
	"github.com/bottlerocket-os/bottlerocket-update-operator/pkg/internal/intents"
	"github.com/bottlerocket-os/bottlerocket-update-operator/pkg/internal/testoutput"
	"github.com/bottlerocket-os/bottlerocket-update-operator/pkg/logging"
	"github.com/bottlerocket-os/bottlerocket-update-operator/pkg/marker"

	"gotest.tools/assert"
)

func TestPolicyCheck(t *testing.T) {
	cases := []struct {
		Name         string
		PolicyCheck  *PolicyCheck
		ShouldPermit bool
		ShouldError  bool
	}{
		// should not update when threshold would be exceeded
		{
			Name:         "update-available-maxactive",
			ShouldPermit: false,
			PolicyCheck: &PolicyCheck{
				Intent:        intents.Stabilized(intents.WithUpdateAvailable(marker.NodeUpdateUnavailable)),
				ClusterActive: maxClusterActive,
				ClusterCount:  maxClusterActive + 1,
			},
		},
		// stabilize should always be permitted
		{
			Name:         "stabilize-new",
			ShouldPermit: true,
			PolicyCheck: &PolicyCheck{
				Intent:        intents.PendingStabilizing(),
				ClusterActive: maxClusterActive,
				ClusterCount:  maxClusterActive + 1,
			},
		},
		{
			Name:         "stabilize-new",
			ShouldPermit: true,
			PolicyCheck: &PolicyCheck{
				Intent:        intents.PendingStabilizing(),
				ClusterActive: 0,
				ClusterCount:  maxClusterActive + 1,
			},
		},
		{
			Name:         "stabilize-new",
			ShouldPermit: true,
			PolicyCheck: &PolicyCheck{
				Intent:        intents.PendingStabilizing(),
				ClusterActive: maxClusterActive,
				ClusterCount:  maxClusterActive + 1,
			},
		},
		{
			Name:         "perform-max-active",
			ShouldPermit: false,
			PolicyCheck: &PolicyCheck{
				Intent:        intents.PendingPrepareUpdate(),
				ClusterActive: maxClusterActive,
				ClusterCount:  maxClusterActive + 1,
			},
		},
		{
			Name:         "perform-over-threshold",
			ShouldPermit: false,
			PolicyCheck: &PolicyCheck{
				Intent:        intents.PendingPrepareUpdate(),
				ClusterActive: maxClusterActive + 1,
				ClusterCount:  maxClusterActive + 1,
			},
		},
		{
			// After an update, we'll need to handle the Node's intent to
			// uncordon it.
			Name:         "updated",
			ShouldPermit: true,
			PolicyCheck: &PolicyCheck{
				Intent:        intents.UpdateSuccess(),
				ClusterActive: maxClusterActive,
				ClusterCount:  maxClusterActive + 1,
			},
		},
	}

	for _, tc := range cases {

		check := tc.PolicyCheck
		t.Run(fmt.Sprintf("%s(%s) %d/%d", tc.Name, check.Intent.DisplayString(), check.ClusterActive, check.ClusterCount),
			func(t *testing.T) {
				policy := defaultPolicy{
					log: testoutput.Logger(t, logging.New("policy-check")),
				}

				permit, err := policy.Check(check)
				assert.Equal(t, tc.ShouldPermit, permit)
				if tc.ShouldError {
					assert.Error(t, err, "")
				} else {
					assert.NilError(t, err)
				}
			})
	}
}

func TestIsClusterActiveIntents(t *testing.T) {
	cases := []struct {
		Intent   *intent.Intent
		Expected bool
	}{
		// Nodes beginning updates are actively working towards a goal, they're
		// active and should be counted.
		{Intent: intents.PendingPrepareUpdate(), Expected: true},
		// Nodes waiting to start an update are actively working.
		{Intent: intents.PendingUpdate(), Expected: true},
		// Updates success is yet to be handled, so should "occupy" a slot in
		// the active count.
		{Intent: intents.UpdateSuccess(), Expected: true},
		// Errors should prevent others from making progress (eg: error prevents
		// updates in cluster) and "occupy" a slot in the active count.
		{Intent: intents.UpdateError(), Expected: true},
		// Resets and Stabilization are normative, non-intrusive operations and
		// shouldn't add to active count.
		{Intent: intents.PendingStabilizing(), Expected: false},
		{Intent: intents.Reset(), Expected: false},
	}

	for _, tc := range cases {
		t.Run(tc.Intent.DisplayString(), func(t *testing.T) {
			actual := isClusterActive(tc.Intent)
			assert.Equal(t, tc.Expected, actual)
		})
	}
}
