package customer

import (
	"context"

	"github.com/tirenn/commerce/backend/internal/logger"
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

func (u *useCase) ListCustomers(ctx context.Context, filter CustomerFilterQuery) (items []CustomerListItem, total int64, err error) {
	defer logger.Track(ctx, "usecase.customer", "ListCustomers")(&err, map[string]interface{}{"page": filter.Page, "limit": filter.Limit})
	items, total, err = u.repo.ListCustomers(ctx, filter)
	if err != nil {
		logger.Error(ctx, "usecase.customer", "failed to list customers in repository", err)
	}
	return items, total, err
}

func (u *useCase) UpdateStatus(ctx context.Context, userID uint, req *UpdateCustomerStatusRequest) (err error) {
	defer logger.Track(ctx, "usecase.customer", "UpdateStatus")(&err, map[string]interface{}{"user_id": userID, "status": req.Status})
	err = u.repo.UpdateStatus(ctx, userID, req.Status)
	if err != nil {
		logger.Error(ctx, "usecase.customer", "failed to update customer status in repository", err)
	}
	return err
}
