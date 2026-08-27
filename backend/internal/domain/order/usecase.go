package order

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"time"

	"gocommerce-backend/internal/logger"
)

type UseCase interface {
	Checkout(ctx context.Context, userID uint, req *CheckoutRequest) (*Order, error)
	GetOrderByID(ctx context.Context, id uint, requestingUserID uint, isAdmin bool) (*Order, error)
	GetOrderByOrderNumber(ctx context.Context, orderNumber string, requestingUserID uint, isAdmin bool) (*Order, error)
	ListCustomerOrders(ctx context.Context, userID uint, page, limit int) ([]Order, int64, error)
	ListAllOrders(ctx context.Context, filter OrderFilterQuery) ([]Order, int64, error)
	UpdateOrderStatus(ctx context.Context, orderID uint, req *UpdateOrderStatusRequest) error
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
	return fmt.Sprintf("GC-%s-%d", time.Now().Format("20060102"), randomPart)
}

func (u *useCase) Checkout(ctx context.Context, userID uint, req *CheckoutRequest) (*Order, error) {
	if len(req.Items) == 0 {
		errEmpty := errors.New("cart is empty")
		logger.Warn(ctx, "usecase", "checkout attempted with empty cart", errEmpty)
		return nil, errEmpty
	}

	orderNumber := generateOrderNumber()

	paymentMethod := "SIMULATED_CARD"
	if req.PaymentMethod != "" {
		paymentMethod = req.PaymentMethod
	}

	order := &Order{
		OrderNumber:     orderNumber,
		UserID:          userID,
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
		logger.Warn(ctx, "usecase", fmt.Sprintf("order with ID %d not found", id), err)
		return nil, errors.New("order not found")
	}

	if !isAdmin && order.UserID != requestingUserID {
		errForbidden := errors.New("unauthorized to view this order")
		logger.Warn(ctx, "usecase", fmt.Sprintf("user %d attempted to view order %d belonging to user %d", requestingUserID, id, order.UserID), errForbidden)
		return nil, errForbidden
	}

	return order, nil
}

func (u *useCase) GetOrderByOrderNumber(ctx context.Context, orderNumber string, requestingUserID uint, isAdmin bool) (*Order, error) {
	order, err := u.repo.FindByOrderNumber(ctx, orderNumber)
	if err != nil {
		logger.Warn(ctx, "usecase", fmt.Sprintf("order with number %s not found", orderNumber), err)
		return nil, errors.New("order not found")
	}

	if !isAdmin && order.UserID != requestingUserID {
		errForbidden := errors.New("unauthorized to view this order")
		logger.Warn(ctx, "usecase", fmt.Sprintf("user %d attempted to view order %s belonging to user %d", requestingUserID, orderNumber, order.UserID), errForbidden)
		return nil, errForbidden
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

func (u *useCase) ListAllOrders(ctx context.Context, filter OrderFilterQuery) ([]Order, int64, error) {
	orders, total, err := u.repo.ListAll(ctx, filter)
	if err != nil {
		logger.Error(ctx, "usecase", "failed to list all orders for admin", err)
		return nil, 0, err
	}
	return orders, total, nil
}

func (u *useCase) UpdateOrderStatus(ctx context.Context, orderID uint, req *UpdateOrderStatusRequest) error {
	order, err := u.repo.FindByID(ctx, orderID)
	if err != nil {
		logger.Error(ctx, "usecase", fmt.Sprintf("order %d not found for status update", orderID), err)
		return errors.New("order not found")
	}

	if err := u.repo.UpdateStatus(ctx, orderID, req.Status, req.Notes); err != nil {
		logger.Error(ctx, "usecase", fmt.Sprintf("failed to update status of order %d to %s", orderID, req.Status), err)
		return err
	}

	logger.Info(ctx, "usecase", fmt.Sprintf("order %s status updated from %s to %s", order.OrderNumber, order.Status, req.Status))
	return nil
}
