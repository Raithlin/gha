package release

import (
	"fmt"

	"github.com/raithlin/gha/pkg/model"
)

// CISignal records whether a provider policy or workflow inventory was verified.
type CISignal struct {
	State   string `json:"state" yaml:"state"`
	Message string `json:"message,omitempty" yaml:"message,omitempty"`
}

// RequiredCheck identifies a required context, optionally bound to a provider app.
type RequiredCheck struct {
	Context string `json:"context" yaml:"context"`
	AppID   int64  `json:"app_id,omitempty" yaml:"app_id,omitempty"`
}

// WorkflowRun records the selected execution and its latest attempt for the release commit.
type WorkflowRun struct {
	ID           int64  `json:"id" yaml:"id"`
	WorkflowID   int64  `json:"workflow_id" yaml:"workflow_id"`
	CheckSuiteID int64  `json:"check_suite_id" yaml:"check_suite_id"`
	RunAttempt   int    `json:"run_attempt" yaml:"run_attempt"`
	Name         string `json:"name" yaml:"name"`
	HeadSHA      string `json:"head_sha" yaml:"head_sha"`
	HeadBranch   string `json:"head_branch" yaml:"head_branch"`
	Event        string `json:"event" yaml:"event"`
	Status       string `json:"status" yaml:"status"`
	Conclusion   string `json:"conclusion" yaml:"conclusion"`
}

// CIState exposes the readiness policy and the evidence selected for evaluation.
type CIState struct {
	Blockers       []CIBlocker     `json:"blockers" yaml:"blockers"`
	Checks         CISignal        `json:"checks" yaml:"checks"`
	Policy         CISignal        `json:"policy" yaml:"policy"`
	RequiredChecks []RequiredCheck `json:"required_checks" yaml:"required_checks"`
	Workflows      CISignal        `json:"workflows" yaml:"workflows"`
	Runs           []WorkflowRun   `json:"runs" yaml:"runs"`
}

func normalizeCI(ci CIState) CIState {
	if ci.Checks.State == "" {
		ci.Checks.State = "unavailable"
	}
	if ci.Policy.State == "" {
		ci.Policy.State = "unavailable"
	}
	if ci.Workflows.State == "" {
		ci.Workflows.State = "unavailable"
	}
	if ci.RequiredChecks == nil {
		ci.RequiredChecks = []RequiredCheck{}
	}
	if ci.Runs == nil {
		ci.Runs = []WorkflowRun{}
	}
	return ci
}

func policyBlockers(ci CIState, checks []*model.CheckRun) []CIBlocker {
	blockers := []CIBlocker{}
	if ci.Policy.State != "available" {
		blockers = append(blockers, CIBlocker{Name: "required-check policy", Status: "unavailable", Conclusion: "unavailable", Message: "required-check policy is unavailable"})
	}
	if ci.Workflows.State != "available" {
		blockers = append(blockers, CIBlocker{Name: "workflow inventory", Status: "unavailable", Conclusion: "unavailable", Message: "workflow runs are unavailable"})
	}
	for _, run := range ci.Runs {
		if !nonFailing(run.Status, run.Conclusion) {
			blockers = append(blockers, CIBlocker{Name: "workflow " + run.Name, Status: run.Status, Conclusion: run.Conclusion})
		}
	}
	for _, required := range ci.RequiredChecks {
		found := false
		for _, check := range checks {
			if check != nil && check.Name == required.Context && (required.AppID <= 0 || (check.App != nil && check.App.ID == required.AppID)) {
				found = true
			}
		}
		if !found {
			blockers = append(blockers, CIBlocker{Name: required.Context, Status: "missing", Conclusion: "unavailable"})
		}
	}
	return blockers
}

func nonFailing(status, conclusion string) bool {
	return status == "completed" && (conclusion == "success" || conclusion == "skipped" || conclusion == "neutral")
}

// CIBlocker is an inspectable reason why release validation is blocked.
type CIBlocker struct {
	Name       string `json:"name" yaml:"name"`
	Status     string `json:"status" yaml:"status"`
	Conclusion string `json:"conclusion" yaml:"conclusion"`
	Message    string `json:"message,omitempty" yaml:"message,omitempty"`
}

func (b CIBlocker) String() string {
	if b.Message != "" {
		return b.Message
	}
	return fmt.Sprintf("CI check %q: status=%q conclusion=%q", b.Name, b.Status, b.Conclusion)
}
