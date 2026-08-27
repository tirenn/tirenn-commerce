package customer

import (
	"context"

	"gocommerce-backend/internal/domain/auth"
	"gorm.io/gorm"
)

type Repository interface {
	ListCustomers(ctx context.Context, filter CustomerFilterQuery) ([]CustomerListItem, int64, error)
	UpdateStatus(ctx context.Context, userID uint, status string) error
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

func (r *repository) ListCustomers(ctx context.Context, filter CustomerFilterQuery) ([]CustomerListItem, int64, error) {
	var total int64
	var items []CustomerListItem

	query := r.db.WithContext(ctx).Table("users").Where("role = ?", auth.RoleCustomer)

	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}

	if filter.Search != "" {
		searchTerm := "%" + filter.Search + "%"
		query = query.Where("name LIKE ? OR email LIKE ? OR phone LIKE ?", searchTerm, searchTerm, searchTerm)
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

	// Join with orders to get count and total spent
	rows, err := query.Select(`
		users.id, 
		users.name, 
		users.email, 
		users.phone, 
		users.address, 
		users.status, 
		users.created_at as registered_at,
		COALESCE((SELECT COUNT(orders.id) FROM orders WHERE orders.user_id = users.id), 0) as total_orders,
		COALESCE((SELECT SUM(orders.total_amount) FROM orders WHERE orders.user_id = users.id AND orders.status != 'CANCELLED'), 0) as total_spent
	`).Order("users.created_at DESC").Offset(offset).Limit(limit).Rows()

	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	for rows.Next() {
		var item CustomerListItem
		if err := rows.Scan(
			&item.ID,
			&item.Name,
			&item.Email,
			&item.Phone,
			&item.Address,
			&item.Status,
			&item.RegisteredAt,
			&item.TotalOrders,
			&item.TotalSpent,
		); err != nil {
			return nil, 0, err
		}
		items = append(items, item)
	}

	return items, total, nil
}

func (r *repository) UpdateStatus(ctx context.Context, userID uint, status string) error {
	return r.db.WithContext(ctx).Model(&auth.User{}).Where("id = ? AND role = ?", userID, auth.RoleCustomer).Update("status", status).Error
}
