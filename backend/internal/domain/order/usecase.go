package order

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"time"
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
		return nil, errors.New("cart is empty")
	}

	orderNumber := generateOrderNumber()

	paymentMethod := "SIMULATED_CARD"
	if req.PaymentMethod != "" {
		paymentMethod = req.PaymentMethod
	}

	order := &Order{
		OrderNumber:     orderNumber,
		UserID:          userID,
		Status:          StatusPaid, // In our simulated checkout, it starts as PAID
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
		return nil, err
	}

	return u.repo.FindByID(ctx, order.ID)
}

func (u *useCase) GetOrderByID(ctx context.Context, id uint, requestingUserID uint, isAdmin bool) (*Order, error) {
	order, err := u.repo.FindByID(ctx, id)
	if err != nil {
		return nil, errors.New("order not found")
	}

	if !isAdmin && order.UserID != requestingUserID {
		return nil, errors.New("unauthorized to view this order")
	}

	return order, nil
}

func (u *useCase) GetOrderByOrderNumber(ctx context.Context, orderNumber string, requestingUserID uint, isAdmin bool) (*Order, error) {
	order, err := u.repo.FindByOrderNumber(ctx, orderNumber)
	if err != nil {
		return nil, errors.New("order not found")
	}

	if !isAdmin && order.UserID != requestingUserID {
		return nil, errors.New("unauthorized to view this order")
	}

	return order, nil
}

func (u *useCase) ListCustomerOrders(ctx context.Context, userID uint, page, limit int) ([]Order, int64, error) {
	return u.repo.ListByUser(ctx, userID, page, limit)
}

func (u *useCase) ListAllOrders(ctx context.Context, filter OrderFilterQuery) ([]Order, int64, error) {
	return u.repo.ListAll(ctx, filter)
}

func (u *useCase) UpdateOrderStatus(ctx context.Context, orderID uint, req *UpdateOrderStatusRequest) error {
	if req.Status == StatusCancelled {
		return u.repo.CancelOrderAndRestock(ctx, orderID, req.Notes)
	}
	return u.repo.UpdateStatus(ctx, orderID, req.Status, req.Notes)
}
