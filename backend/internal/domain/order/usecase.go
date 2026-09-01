package order

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/tirenn/commerce/backend/internal/domain"
	"github.com/tirenn/commerce/backend/internal/logger"
)

type UseCase interface {
	Checkout(ctx context.Context, userID uint, req *CheckoutRequest) (*Order, error)
	GetOrderByID(ctx context.Context, id uint, requestingUserID uint, isAdmin bool) (*Order, error)
	GetOrderByOrderNumber(ctx context.Context, orderNumber string, requestingUserID uint, isAdmin bool) (*Order, error)
	ListCustomerOrders(ctx context.Context, userID uint, page, limit int) ([]Order, int64, error)
	AdminListOrders(ctx context.Context, filter OrderFilterQuery) ([]Order, int64, error)
	AdminUpdateStatus(ctx context.Context, orderID uint, req *UpdateOrderStatusRequest) (*Order, error)
}

type useCase struct {
	repo Repository
}

func NewUseCase(repo Repository) UseCase {
	return &useCase{repo: repo}
}

func generateOrderNumber() string {
	n, _ := rand.Int(rand.Reader, big.NewInt(900000))
	randomPart := n.Int64() + 100000
	return fmt.Sprintf("TRN-%s-%d", time.Now().Format("20060102"), randomPart)
}

func (u *useCase) Checkout(ctx context.Context, userID uint, req *CheckoutRequest) (ord *Order, err error) {
	defer logger.Track(ctx, "usecase.order", "Checkout")(&err, map[string]interface{}{"user_id": userID, "items_count": len(req.Items)})
	if len(req.Items) == 0 {
		logger.Warn(ctx, "usecase", "checkout attempted with empty cart", domain.ErrBadRequest)
		return nil, domain.ErrBadRequest
	}

	orderNumber := generateOrderNumber()

	paymentMethod := "SIMULATED_CARD"
	if req.PaymentMethod != "" {
		paymentMethod = req.PaymentMethod
	}

	currency := "IDR"
	if req.Currency != "" {
		currency = req.Currency
	}

	order := &Order{
		OrderNumber:     orderNumber,
		UserID:          userID,
		Currency:        currency,
		Status:          StatusPaid,
		ShippingName:    req.ShippingName,
		ShippingPhone:   req.ShippingPhone,
		ShippingAddress: req.ShippingAddress,
		PaymentMethod:   paymentMethod,
		PaymentStatus:   PaymentSuccess,
		Notes:           req.Notes,
	}

	items := make([]OrderItem, len(req.Items))
	for i, itemReq := range req.Items {
		items[i] = OrderItem{
			ProductID: itemReq.ProductID,
			Quantity:  itemReq.Quantity,
		}
	}

	if err := u.repo.CreateOrderWithItems(ctx, order, items); err != nil {
		logger.Error(ctx, "usecase", "failed to create order with items and stock deduction", err)
		return nil, err
	}

	logger.Info(ctx, "usecase", fmt.Sprintf("order %s created successfully for user %d", order.OrderNumber, userID))
	return u.repo.FindByID(ctx, order.ID)
}

func (u *useCase) GetOrderByID(ctx context.Context, id uint, requestingUserID uint, isAdmin bool) (*Order, error) {
	order, err := u.repo.FindByID(ctx, id)
	if err != nil {
		logger.Warn(ctx, "usecase", fmt.Sprintf("order with ID %d not found", id), domain.ErrNotFound)
		return nil, domain.ErrNotFound
	}

	if !isAdmin && order.UserID != requestingUserID {
		logger.Warn(ctx, "usecase", fmt.Sprintf("user %d attempted to view order %d belonging to user %d", requestingUserID, id, order.UserID), domain.ErrForbidden)
		return nil, domain.ErrForbidden
	}

	return order, nil
}

func (u *useCase) GetOrderByOrderNumber(ctx context.Context, orderNumber string, requestingUserID uint, isAdmin bool) (*Order, error) {
	order, err := u.repo.FindByOrderNumber(ctx, orderNumber)
	if err != nil {
		logger.Warn(ctx, "usecase", fmt.Sprintf("order with number %s not found", orderNumber), domain.ErrNotFound)
		return nil, domain.ErrNotFound
	}

	if !isAdmin && order.UserID != requestingUserID {
		logger.Warn(ctx, "usecase", fmt.Sprintf("user %d attempted to view order %s belonging to user %d", requestingUserID, orderNumber, order.UserID), domain.ErrForbidden)
		return nil, domain.ErrForbidden
	}

	return order, nil
}

func (u *useCase) ListCustomerOrders(ctx context.Context, userID uint, page, limit int) ([]Order, int64, error) {
	orders, total, err := u.repo.ListByUser(ctx, userID, page, limit)
	if err != nil {
		logger.Error(ctx, "usecase", fmt.Sprintf("failed to list orders for customer %d", userID), err)
		return nil, 0, err
	}
	return orders, total, nil
}

func (u *useCase) AdminListOrders(ctx context.Context, filter OrderFilterQuery) ([]Order, int64, error) {
	orders, total, err := u.repo.ListAll(ctx, filter)
	if err != nil {
		logger.Error(ctx, "usecase", "failed to list all orders for admin", err)
		return nil, 0, err
	}
	return orders, total, nil
}

func (u *useCase) AdminUpdateStatus(ctx context.Context, orderID uint, req *UpdateOrderStatusRequest) (*Order, error) {
	order, err := u.repo.FindByID(ctx, orderID)
	if err != nil {
		logger.Error(ctx, "usecase", fmt.Sprintf("order %d not found for status update", orderID), domain.ErrNotFound)
		return nil, domain.ErrNotFound
	}

	if strings.EqualFold(req.Status, string(StatusCancelled)) {
		if err := u.repo.CancelOrderAndRestock(ctx, orderID, req.Notes); err != nil {
			logger.Error(ctx, "usecase", fmt.Sprintf("failed to cancel and restock order %d", orderID), err)
			return nil, err
		}
	} else {
		if err := u.repo.UpdateStatus(ctx, orderID, req.Status, req.Notes); err != nil {
			logger.Error(ctx, "usecase", fmt.Sprintf("failed to update status of order %d to %s", orderID, req.Status), err)
			return nil, err
		}
	}

	logger.Info(ctx, "usecase", fmt.Sprintf("order %s status updated from %s to %s", order.OrderNumber, order.Status, req.Status))
	return u.repo.FindByID(ctx, orderID)
}
