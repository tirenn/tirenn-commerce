package customer

import (
	"context"
)

type UseCase interface {
	ListCustomers(ctx context.Context, filter CustomerFilterQuery) ([]CustomerListItem, int64, error)
	UpdateStatus(ctx context.Context, userID uint, req *UpdateCustomerStatusRequest) error
}

type useCase struct {
	repo Repository
}

func NewUseCase(repo Repository) UseCase {
	return &useCase{repo: repo}
}

func (u *useCase) ListCustomers(ctx context.Context, filter CustomerFilterQuery) ([]CustomerListItem, int64, error) {
	return u.repo.ListCustomers(ctx, filter)
}

func (u *useCase) UpdateStatus(ctx context.Context, userID uint, req *UpdateCustomerStatusRequest) error {
	return u.repo.UpdateStatus(ctx, userID, req.Status)
}
