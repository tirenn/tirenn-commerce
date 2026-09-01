package dashboard

import (
	"context"

	"tirenn-ai-commerce/internal/logger"
)

type UseCase interface {
	GetDashboardData(ctx context.Context, days int) (*DashboardResponse, error)
}

type useCase struct {
	repo Repository
}

func NewUseCase(repo Repository) UseCase {
	return &useCase{repo: repo}
}

func (u *useCase) GetDashboardData(ctx context.Context, days int) (res *DashboardResponse, err error) {
	defer logger.Track(ctx, "usecase.dashboard", "GetDashboardData")(&err, map[string]interface{}{"days": days})
	summary, err := u.repo.GetSummary(ctx)
	if err != nil {
		return nil, err
	}

	trends, err := u.repo.GetRevenueTrends(ctx, days)
	if err != nil {
		return nil, err
	}

	topProducts, err := u.repo.GetTopSellingProducts(ctx, 5)
	if err != nil {
		return nil, err
	}

	lowStock, err := u.repo.GetLowStockAlerts(ctx)
	if err != nil {
		return nil, err
	}

	recentOrders, err := u.repo.GetRecentOrders(ctx, 6)
	if err != nil {
		return nil, err
	}

	return &DashboardResponse{
		Summary:            *summary,
		RevenueTrends:      trends,
		TopSellingProducts: topProducts,
		LowStockAlerts:     lowStock,
		RecentOrders:       recentOrders,
	}, nil
}
