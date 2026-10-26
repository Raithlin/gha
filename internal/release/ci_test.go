package release

import (
	"context"
	"testing"

	"github.com/raithlin/gha/pkg/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReleaseCIMixedPackageValidation(t *testing.T) {
	checkout, provider := readyFixtures()
	provider.checks = []*model.CheckRun{
		{Name: "native", Status: "completed", Conclusion: "skipped"},
		{Name: "Arch", Status: "completed", Conclusion: "success"},
		{Name: "Debian", Status: "completed", Conclusion: "success"},
		{Name: "Fedora", Status: "completed", Conclusion: "success"},
		{Name: "advisory", Status: "completed", Conclusion: "neutral"},
	}
	plan, err := NewService(checkout, provider).Prepare(context.Background(), checkout.state.Repository, "1.2.3", "", true)
	require.NoError(t, err)
	assert.True(t, plan.Ready, "%v", plan.Blockers)
}

func TestReleaseCIBlockingStatesNameEveryCheck(t *testing.T) {
	for _, tc := range []struct{ status, conclusion string }{
		{"completed", "failure"}, {"completed", "cancelled"}, {"completed", "timed_out"},
		{"completed", "action_required"}, {"completed", "stale"}, {"completed", ""},
		{"queued", ""}, {"in_progress", ""}, {"waiting", ""}, {"unknown", "success"},
	} {
		t.Run(tc.status+"/"+tc.conclusion, func(t *testing.T) {
			checkout, provider := readyFixtures()
			provider.checks = append(provider.checks, &model.CheckRun{Name: "package", Status: tc.status, Conclusion: tc.conclusion})
			plan, err := NewService(checkout, provider).Prepare(context.Background(), checkout.state.Repository, "1.2.3", "", true)
			require.NoError(t, err)
			assert.False(t, plan.Ready)
			require.NotEmpty(t, plan.CI.Blockers)
			assert.Equal(t, "package", plan.CI.Blockers[0].Name)
			assert.Equal(t, tc.status, plan.CI.Blockers[0].Status)
			assert.Equal(t, tc.conclusion, plan.CI.Blockers[0].Conclusion)
			assert.Contains(t, plan.Blockers, `CI check "package": status="`+tc.status+`" conclusion="`+tc.conclusion+`"`)
		})
	}
}

func TestReleaseCIRequiresSuccessfulValidation(t *testing.T) {
	for _, conclusion := range []string{"skipped", "neutral"} {
		checkout, provider := readyFixtures()
		provider.checks[0].Conclusion = conclusion
		plan, err := NewService(checkout, provider).Prepare(context.Background(), checkout.state.Repository, "1.2.3", "", true)
		require.NoError(t, err)
		assert.False(t, plan.Ready)
		assert.Contains(t, plan.Blockers, "CI has no successful validation check")
	}
}

func TestReleaseCIRequiredPolicyAndWorkflows(t *testing.T) {
	for _, tc := range []struct {
		name    string
		change  func(*fakeProvider)
		ready   bool
		blocker string
	}{
		{"required success", func(p *fakeProvider) { p.ci.RequiredChecks = []RequiredCheck{{Context: "test"}} }, true, ""},
		{"required absent", func(p *fakeProvider) { p.ci.RequiredChecks = []RequiredCheck{{Context: "missing"}} }, false, `CI check "missing": status="missing" conclusion="unavailable"`},
		{"policy unavailable", func(p *fakeProvider) { p.ci.Policy.State = "unavailable" }, false, "required-check policy is unavailable"},
		{"workflow unavailable", func(p *fakeProvider) { p.ci.Workflows.State = "unavailable" }, false, "workflow runs are unavailable"},
		{"workflow cancelled", func(p *fakeProvider) {
			p.ci.Runs = []WorkflowRun{{Name: "CI", Status: "completed", Conclusion: "cancelled"}}
		}, false, `CI check "workflow CI": status="completed" conclusion="cancelled"`},
		{"required wrong app", func(p *fakeProvider) { p.ci.RequiredChecks = []RequiredCheck{{Context: "test", AppID: 42}} }, false, `CI check "test": status="missing" conclusion="unavailable"`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			checkout, provider := readyFixtures()
			tc.change(provider)
			plan, err := NewService(checkout, provider).Prepare(context.Background(), checkout.state.Repository, "1.2.3", "", true)
			require.NoError(t, err)
			assert.Equal(t, tc.ready, plan.Ready, "%v", plan.Blockers)
			if tc.blocker != "" {
				assert.Contains(t, plan.Blockers, tc.blocker)
			}
			assert.Equal(t, provider.ci.Policy, plan.CI.Policy)
		})
	}
}
