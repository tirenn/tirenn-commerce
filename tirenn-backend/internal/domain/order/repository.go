package order

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gocommerce-backend/internal/domain/product"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Repository interface {
	CreateOrderWithItems(ctx context.Context, order *Order, items []OrderItem) error
	FindByID(ctx context.Context, id uint) (*Order, error)
	FindByOrderNumber(ctx context.Context, orderNumber string) (*Order, error)
	ListByUser(ctx context.Context, userID uint, page, limit int) ([]Order, int64, error)
	ListAll(ctx context.Context, filter OrderFilterQuery) ([]Order, int64, error)
	UpdateStatus(ctx context.Context, orderID uint, status, notes string) error
	CancelOrderAndRestock(ctx context.Context, orderID uint, notes string) error
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

func (r *repository) CreateOrderWithItems(ctx context.Context, order *Order, items []OrderItem) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var totalAmount float64

		// 1. Lock each product row & verify and decrement stock atomically
		for i := range items {
			var p product.Product
			// SELECT ... FOR UPDATE to prevent race conditions during concurrent checkouts
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&p, items[i].ProductID).Error; err != nil {
				return fmt.Errorf("product ID %d not found", items[i].ProductID)
			}

			if !p.IsActive {
				return fmt.Errorf("product '%s' is no longer active", p.Name)
			}

			if p.StockQuantity < items[i].Quantity {
				return fmt.Errorf("insufficient stock for '%s' (available: %d, requested: %d)", p.Name, p.StockQuantity, items[i].Quantity)
			}

			// Decrement product stock
			newStock := p.StockQuantity - items[i].Quantity
			if err := tx.Model(&p).Update("stock_quantity", newStock).Error; err != nil {
				return err
			}

			// Fill in item details snapshot
			items[i].ProductName = p.Name
			items[i].ProductSKU = p.SKU
			items[i].ProductImage = p.ImageURL
			items[i].UnitPrice = p.Price
			items[i].Subtotal = p.Price * float64(items[i].Quantity)
			items[i].Currency = p.Currency
			if items[i].Currency == "" {
				items[i].Currency = "IDR"
			}

			totalAmount += items[i].Subtotal

			// Log stock adjustment for checkout
			stockLog := product.StockAdjustmentLog{
				ProductID:      p.ID,
				AdjustmentType: "SUBTRACT",
				Quantity:       items[i].Quantity,
				PreviousStock:  p.StockQuantity,
				NewStock:       newStock,
				Reason:         fmt.Sprintf("Checkout Order #%s", order.OrderNumber),
				AdjustedBy:     order.UserID,
			}
			if err := tx.Create(&stockLog).Error; err != nil {
				return err
			}
		}

		order.TotalAmount = totalAmount
		if order.Currency == "" {
			order.Currency = "IDR"
		}

		// 2. Insert Order Header
		if err := tx.Create(order).Error; err != nil {
			return err
		}

		// 3. Insert Order Items
		for i := range items {
			items[i].OrderID = order.ID
			if err := tx.Create(&items[i]).Error; err != nil {
				return err
			}
		}

		return nil
	})
}

func (r *repository) FindByID(ctx context.Context, id uint) (*Order, error) {
	var o Order
	err := r.db.WithContext(ctx).Preload("User").Preload("Items").Preload("Items.Product").First(&o, id).Error
	if err != nil {
		return nil, err
	}
	return &o, nil
}

func (r *repository) FindByOrderNumber(ctx context.Context, orderNumber string) (*Order, error) {
	var o Order
	err := r.db.WithContext(ctx).Preload("User").Preload("Items").Preload("Items.Product").Where("order_number = ?", orderNumber).First(&o).Error
	if err != nil {
		return nil, err
	}
	return &o, nil
}

func (r *repository) ListByUser(ctx context.Context, userID uint, page, limit int) ([]Order, int64, error) {
	var orders []Order
	var total int64

	query := r.db.WithContext(ctx).Model(&Order{}).Where("user_id = ?", userID).Preload("Items")

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 10
	}
	offset := (page - 1) * limit

	err := query.Order("created_at DESC").Offset(offset).Limit(limit).Find(&orders).Error
	return orders, total, err
}

func (r *repository) ListAll(ctx context.Context, filter OrderFilterQuery) ([]Order, int64, error) {
	var orders []Order
	var total int64

	query := r.db.WithContext(ctx).Model(&Order{}).Preload("User").Preload("Items")

	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}

	if filter.Search != "" {
		searchTerm := "%" + filter.Search + "%"
		query = query.Where("order_number LIKE ? OR shipping_name LIKE ? OR shipping_phone LIKE ?", searchTerm, searchTerm, searchTerm)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	page := filter.Page
	if page < 1 {
		page = 1
	}
	limit := filter.Limit
	if limit < 1 || limit > 100 {
		limit = 10
	}
	offset := (page - 1) * limit

	err := query.Order("created_at DESC").Offset(offset).Limit(limit).Find(&orders).Error
	return orders, total, err
}

func (r *repository) UpdateStatus(ctx context.Context, orderID uint, status, notes string) error {
	updates := map[string]interface{}{
		"status":     status,
		"updated_at": time.Now(),
	}
	if notes != "" {
		updates["notes"] = notes
	}
	return r.db.WithContext(ctx).Model(&Order{}).Where("id = ?", orderID).Updates(updates).Error
}

func (r *repository) CancelOrderAndRestock(ctx context.Context, orderID uint, notes string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var o Order
		if err := tx.Preload("Items").First(&o, orderID).Error; err != nil {
			return err
		}

		if o.Status == StatusCancelled {
			return errors.New("order is already cancelled")
		}

		// Restock each product
		for _, item := range o.Items {
			var p product.Product
			if err := tx.First(&p, item.ProductID).Error; err == nil {
				newStock := p.StockQuantity + item.Quantity
				_ = tx.Model(&p).Update("stock_quantity", newStock).Error

				stockLog := product.StockAdjustmentLog{
					ProductID:      p.ID,
					AdjustmentType: "ADD",
					Quantity:       item.Quantity,
					PreviousStock:  p.StockQuantity,
					NewStock:       newStock,
					Reason:         fmt.Sprintf("Order #%s Cancelled / Restocked", o.OrderNumber),
					AdjustedBy:     0,
				}
				_ = tx.Create(&stockLog).Error
			}
		}

		updates := map[string]interface{}{
			"status":     StatusCancelled,
			"notes":      notes,
			"updated_at": time.Now(),
		}
		return tx.Model(&o).Updates(updates).Error
	})
}
