package release

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/raithlin/gha/pkg/model"
	"github.com/stretchr/testify/require"
)

type scriptedStatusProvider struct {
	results []*Status
	reads   int
	cancel  context.CancelFunc
	err     error
}

func (p *scriptedStatusProvider) ReadStatus(_ context.Context, repository model.RepositoryRef, tag, _ string) (*Status, error) {
	if p.err != nil {
		return nil, p.err
	}
	if len(p.results) == 0 {
		return nil, nil
	}
	result := *p.results[min(p.reads, len(p.results)-1)]
	result.Repository, result.Tag = repository, tag
	p.reads++
	if p.cancel != nil && p.reads == 1 {
		p.cancel()
	}
	return &result, nil
}

func TestStatusCheckRejectsMissingProviderAndMissingResult(t *testing.T) {
	repo := model.RepositoryRef{Owner: "acme", Name: "tool"}
	_, err := NewStatusService(nil).Check(context.Background(), repo, "v1.2.3", "")
	require.ErrorContains(t, err, "not configured")
	_, err = NewStatusService(&scriptedStatusProvider{}).Check(context.Background(), repo, "v1.2.3", "")
	require.ErrorContains(t, err, "no result")
	_, err = NewStatusService(&scriptedStatusProvider{err: errors.New("provider unavailable")}).Check(context.Background(), repo, "v1.2.3", "")
	require.ErrorContains(t, err, "provider unavailable")
}

func TestStatusWaitStopsOnObservedWorkflowFailure(t *testing.T) {
	p := &scriptedStatusProvider{results: []*Status{{State: "failed", Commit: "abc"}}}
	result, err := NewStatusService(p).Wait(context.Background(), model.RepositoryRef{Owner: "acme", Name: "tool"}, "v1.2.3", "", time.Second, time.Hour)
	require.NoError(t, err)
	require.Equal(t, "completed", result.Wait.State)
	require.Equal(t, "failed", result.State)
}

func TestStatusWaitObservesPublicationAfterEventualConsistency(t *testing.T) {
	p := &scriptedStatusProvider{results: []*Status{
		{State: "undiscovered"},
		{State: "pending"},
		{State: "published"},
	}}
	result, err := NewStatusService(p).Wait(context.Background(), model.RepositoryRef{Owner: "acme", Name: "tool"}, "v1.2.3", "", time.Second, time.Nanosecond)
	require.NoError(t, err)
	require.Equal(t, "published", result.State)
	require.Equal(t, "completed", result.Wait.State)
	require.Equal(t, 3, result.Wait.Observations)
}

func TestStatusWaitReturnsPartialObservationOnCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	p := &scriptedStatusProvider{results: []*Status{{State: "pending", Commit: "abc"}}, cancel: cancel}
	result, err := NewStatusService(p).Wait(ctx, model.RepositoryRef{Owner: "acme", Name: "tool"}, "v1.2.3", "", time.Second, time.Hour)
	require.NoError(t, err)
	require.Equal(t, "cancelled", result.Wait.State)
	require.Equal(t, "abc", result.Commit)
	require.Equal(t, 1, result.Wait.Observations)
}

func TestStatusWaitReturnsPartialObservationOnTimeout(t *testing.T) {
	p := &scriptedStatusProvider{results: []*Status{{State: "pending", Commit: "abc"}}}
	result, err := NewStatusService(p).Wait(context.Background(), model.RepositoryRef{Owner: "acme", Name: "tool"}, "v1.2.3", "", time.Millisecond, time.Hour)
	require.NoError(t, err)
	require.Equal(t, "timed_out", result.Wait.State)
	require.Equal(t, "pending", result.State)
	require.Equal(t, "abc", result.Commit)
}
