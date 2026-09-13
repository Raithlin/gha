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

// Service coordinates branch workflows through Git.
type Service struct {
	lister Lister
}

// NewService creates a branch service backed by lister.
func NewService(lister Lister) *Service {
	return &Service{lister: lister}
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
