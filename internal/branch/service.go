// Package branch provides read-only repository branch workflows.
package branch

import (
	"context"
	"fmt"

	"github.com/raithlin/gha/pkg/model"
)

// Lister reads local and origin branch state from Git.
type Lister interface {
	List(context.Context, int) (*model.BranchInventory, error)
}

// Inspector reads one branch from local and cached-origin Git state.
type Inspector interface {
	Inspect(context.Context, string) (*model.BranchInspection, error)
}

// SafetyProvider enriches a branch with signals from its code host. Each
// signal must be marked unavailable when the provider cannot retrieve it.
type SafetyProvider interface {
	InspectBranchSafety(context.Context, model.RepositoryRef, string) (model.BranchSafety, error)
}

// Service coordinates branch workflows through Git.
type Service struct {
	lister   Lister
	provider SafetyProvider
}

// NewService creates a branch service backed by lister.
func NewService(lister Lister, providers ...SafetyProvider) *Service {
	var provider SafetyProvider
	if len(providers) > 0 {
		provider = providers[0]
	}
	return &Service{lister: lister, provider: provider}
}

// Inventory returns bounded local and origin branch views without changing Git state.
func (s *Service) Inventory(ctx context.Context, limit int) (*model.BranchInventory, error) {
	if limit < 1 || limit > 100 {
		return nil, fmt.Errorf("limit must be between 1 and 100")
	}
	inventory, err := s.lister.List(ctx, limit)
	if err != nil {
		return nil, fmt.Errorf("inspect branches: %w", err)
	}
	return inventory, nil
}

// WithLister keeps provider enrichment while targeting another checkout.
func (s *Service) WithLister(lister Lister) *Service {
	if s == nil {
		return NewService(lister)
	}
	return NewService(lister, s.provider)
}

// Show inspects one branch without changing Git or provider state. Provider
// failures do not discard local Git facts; their signals are explicit.
func (s *Service) Show(ctx context.Context, name string, repository model.RepositoryRef, repositoryError error) (*model.BranchInspection, error) {
	inspector, ok := s.lister.(Inspector)
	if !ok {
		return nil, fmt.Errorf("inspect branch: local branch inspection is not supported")
	}
	inspection, err := inspector.Inspect(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("inspect branch: %w", err)
	}
	if repositoryError != nil {
		inspection.Safety = unavailableSafety(repositoryError.Error())
		return inspection, nil
	}
	if repository.Owner == "" || repository.Name == "" {
		inspection.Safety = unavailableSafety("repository is unavailable; pass --repo owner/repo")
		return inspection, nil
	}
	inspection.Repository = &repository
	if s.provider == nil {
		inspection.Safety = unavailableSafety("provider safety signals are unavailable in this build")
		return inspection, nil
	}
	safety, err := s.provider.InspectBranchSafety(ctx, repository, name)
	if err != nil {
		inspection.Safety = unavailableSafety(err.Error())
		return inspection, nil
	}
	inspection.Safety = safety
	return inspection, nil
}

func unavailableSafety(message string) model.BranchSafety {
	unavailable := model.ProviderSignal{State: "unavailable", Message: message}
	return model.BranchSafety{
		Requests:      unavailable,
		Protection:    unavailable,
		Permissions:   unavailable,
		DefaultBranch: unavailable,
		Merge:         unavailable,
	}
}
