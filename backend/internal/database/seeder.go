package database

import (
	"log"
	"time"

	"gocommerce-backend/internal/domain/auth"
	"gocommerce-backend/internal/domain/order"
	"gocommerce-backend/internal/domain/product"
	"gocommerce-backend/internal/utils"
	"gorm.io/gorm"
)

func Seed(db *gorm.DB) error {
	var userCount int64
	db.Model(&auth.User{}).Count(&userCount)
	if userCount > 0 {
		return nil // Already seeded
	}

	log.Println("Seeding initial Tirenn Commerce multi-category demo catalog...")

	// 1. Seed Users
	adminPassword, _ := utils.HashPassword("Admin@123")
	customerPassword, _ := utils.HashPassword("Shopper@123")
	sarahPassword, _ := utils.HashPassword("Sarah@123")

	admin := auth.User{
		Name:         "Tirenn Merchant Admin",
		Email:        "admin@gocommerce.com",
		PasswordHash: adminPassword,
		Role:         auth.RoleAdmin,
		Phone:        "+1-555-010-8888",
		Address:      "100 Commerce Way, San Francisco, CA",
		Status:       auth.StatusActive,
	}
	db.Create(&admin)

	shopper := auth.User{
		Name:         "Alex Rivera",
		Email:        "shopper@gocommerce.com",
		PasswordHash: customerPassword,
		Role:         auth.RoleCustomer,
		Phone:        "+1-555-019-2834",
		Address:      "742 Evergreen Terrace, Springfield, OR",
		Status:       auth.StatusActive,
	}
	db.Create(&shopper)

	sarah := auth.User{
		Name:         "Sarah Jenkins",
		Email:        "sarah.jenkins@example.com",
		PasswordHash: sarahPassword,
		Role:         auth.RoleCustomer,
		Phone:        "+1-555-014-9921",
		Address:      "200 Market Street, Seattle, WA",
		Status:       auth.StatusActive,
	}
	db.Create(&sarah)

	// 2. Seed Multi-Category Department Store Categories
	categories := []product.Category{
		{
			Name:        "Electronics & Tech",
			Slug:        "electronics-tech",
			Description: "High-performance headphones, smart wearables, computer peripherals, and smart home audio.",
			Icon:        "⚡",
		},
		{
			Name:        "Fashion & Apparel",
			Slug:        "fashion-apparel",
			Description: "Everyday streetwear, heavyweight hoodies, all-weather jackets, and comfortable sneakers.",
			Icon:        "👕",
		},
		{
			Name:        "Home & Living",
			Slug:        "home-living",
			Description: "Specialty coffee gear, ergonomic furniture, ambient smart lamps, and kitchen essentials.",
			Icon:        "🏡",
		},
		{
			Name:        "Sports & Outdoors",
			Slug:        "sports-outdoors",
			Description: "Waterproof backpacks, stainless insulated flasks, travel gear, and fitness accessories.",
			Icon:        "🎒",
		},
	}
	for i := range categories {
		db.Create(&categories[i])
	}

	// 3. Seed Multi-Category Retail Products
	products := []product.Product{
		// Electronics & Tech
		{
			CategoryID:        categories[0].ID,
			Name:              "AuraPro Active Noise-Cancelling Wireless Headphones",
			Slug:              "aurapro-anc-headphones",
			SKU:               "TECH-AP-001",
			Description:       "Premium over-ear wireless headphones with hybrid active noise cancellation, 40mm titanium dynamic drivers, 45-hour battery life, and ultra-plush memory foam earcups.",
			Price:             149.99,
			StockQuantity:     35,
			LowStockThreshold: 8,
			ImageURL:          "https://images.unsplash.com/photo-1505740420928-5e560c06d30e?w=600&auto=format&fit=crop&q=80",
			IsActive:          true,
			Badge:             "BESTSELLER",
			Rating:            4.9,
		},
		{
			CategoryID:        categories[0].ID,
			Name:              "TitanFit Ultra Smart Health & Fitness Watch",
			Slug:              "titanfit-ultra-smartwatch",
			SKU:               "TECH-TF-002",
			Description:       "AMOLED sapphire display, real-time SpO2 and heart-rate monitoring, 5ATM water resistance, built-in GPS, and seamless iOS / Android companion app synchronization.",
			Price:             89.50,
			StockQuantity:     18,
			LowStockThreshold: 5,
			ImageURL:          "https://images.unsplash.com/photo-1523275335684-37898b6baf30?w=600&auto=format&fit=crop&q=80",
			IsActive:          true,
			Badge:             "TOP RATED",
			Rating:            4.8,
		},
		{
			CategoryID:        categories[0].ID,
			Name:              "ApexCraft RGB Wireless Mechanical Keyboard",
			Slug:              "apexcraft-rgb-keyboard",
			SKU:               "TECH-KB-003",
			Description:       "75% compact hot-swappable mechanical keyboard with lubricated linear switches, CNC anodized aluminum frame, custom PBT keycaps, and tri-mode connectivity (2.4G/BT/USB-C).",
			Price:             119.00,
			StockQuantity:     12,
			LowStockThreshold: 4,
			ImageURL:          "https://images.unsplash.com/photo-1587829741301-dc798b83add3?w=600&auto=format&fit=crop&q=80",
			IsActive:          true,
			Badge:             "HOT DEAL",
			Rating:            4.9,
		},

		// Fashion & Apparel
		{
			CategoryID:        categories[1].ID,
			Name:              "UrbanCraft Heavyweight French Terry Hoodie",
			Slug:              "urbancraft-heavyweight-hoodie",
			SKU:               "FASH-HD-101",
			Description:       "420 GSM 100% organic combed cotton hoodie with double-layered hood, custom ribbed cuffs, dropped shoulders, and pre-shrunk structured streetwear fit.",
			Price:             54.00,
			StockQuantity:     45,
			LowStockThreshold: 10,
			ImageURL:          "https://images.unsplash.com/photo-1556905055-8f358a7a47b2?w=600&auto=format&fit=crop&q=80",
			IsActive:          true,
			Badge:             "NEW ARRIVAL",
			Rating:            4.7,
		},
		{
			CategoryID:        categories[1].ID,
			Name:              "AeroFlex All-Weather Commuter Windbreaker",
			Slug:              "aeroflex-commuter-windbreaker",
			SKU:               "FASH-WB-102",
			Description:       "Ultralight ripstop waterproof shell with taped seams, YKK AquaGuard zippers, hidden hood compartment, and breathable underarm ventilation mesh.",
			Price:             79.99,
			StockQuantity:     22,
			LowStockThreshold: 6,
			ImageURL:          "https://images.unsplash.com/photo-1556821840-3a63f95609a7?w=600&auto=format&fit=crop&q=80",
			IsActive:          true,
			Badge:             "TRENDING",
			Rating:            4.8,
		},

		// Home & Living
		{
			CategoryID:        categories[2].ID,
			Name:              "BaristaCraft Precision Conical Burr Coffee Grinder",
			Slug:              "baristacraft-coffee-grinder",
			SKU:               "HOME-CG-201",
			Description:       "Stainless steel 48mm conical burrs with 30 micro-step grind settings from fine Turkish espresso to coarse French press, low-RPM quiet motor, and anti-static chamber.",
			Price:             68.00,
			StockQuantity:     15,
			LowStockThreshold: 5,
			ImageURL:          "https://images.unsplash.com/photo-1514432324607-a09d9b4aefdd?w=600&auto=format&fit=crop&q=80",
			IsActive:          true,
			Badge:             "CHEF CHOICE",
			Rating:            4.9,
		},
		{
			CategoryID:        categories[2].ID,
			Name:              "Nordic Glow Smart Dimmable Ambient Table Lamp",
			Slug:              "nordic-glow-ambient-lamp",
			SKU:               "HOME-NL-202",
			Description:       "Minimalist brushed aluminum bedside lamp with touch-sensitive step-less dimming, warm 2700K to 6500K color temperature tuning, and integrated Qi wireless charging pad.",
			Price:             42.99,
			StockQuantity:     28,
			LowStockThreshold: 7,
			ImageURL:          "https://images.unsplash.com/photo-1507473885765-e6ed057f782c?w=600&auto=format&fit=crop&q=80",
			IsActive:          true,
			Badge:             "POPULAR",
			Rating:            4.7,
		},

		// Sports & Outdoors
		{
			CategoryID:        categories[3].ID,
			Name:              "Nomad 35L Waterproof Rolltop Travel Backpack",
			Slug:              "nomad-35l-travel-backpack",
			SKU:               "SPRT-BP-301",
			Description:       "Heavy-duty 900D Cordura ballistic nylon with rolltop expandable capacity, dedicated TSA-friendly 16-inch laptop compartment, ergonomic air-mesh back panel, and luggage pass-through.",
			Price:             65.00,
			StockQuantity:     4, // Low stock on purpose to test radar alert
			LowStockThreshold: 5,
			ImageURL:          "https://images.unsplash.com/photo-1553062407-98eeb64c6a62?w=600&auto=format&fit=crop&q=80",
			IsActive:          true,
			Badge:             "⚠️ LOW STOCK",
			Rating:            5.0,
		},
		{
			CategoryID:        categories[3].ID,
			Name:              "HydroShield Double-Wall Vacuum Insulated 32oz Flask",
			Slug:              "hydroshield-vacuum-flask-32oz",
			SKU:               "SPRT-FL-302",
			Description:       "Food-grade 18/8 stainless steel bottle keeping beverages iced cold for 24 hours or piping hot for 12 hours. Sweat-free powder coat finish with leakproof magnetic lid.",
			Price:             24.99,
			StockQuantity:     50,
			LowStockThreshold: 10,
			ImageURL:          "https://images.unsplash.com/photo-1602143407151-7111542de6e8?w=600&auto=format&fit=crop&q=80",
			IsActive:          true,
			Badge:             "ESSENTIAL",
			Rating:            4.8,
		},
	}
	for i := range products {
		db.Create(&products[i])
	}

	// 4. Seed Sample Orders
	order1 := order.Order{
		OrderNumber:     "TC-20260825-102938",
		UserID:          shopper.ID,
		TotalAmount:     149.99,
		Status:          order.StatusShipped,
		ShippingName:    "Alex Rivera",
		ShippingPhone:   "+1-555-019-2834",
		ShippingAddress: "742 Evergreen Terrace, Springfield, OR",
		PaymentMethod:   "SIMULATED_CARD",
		PaymentStatus:   order.PaymentSuccess,
		Notes:           "Leave package on front porch",
		CreatedAt:       time.Now().AddDate(0, 0, -2),
	}
	db.Create(&order1)

	orderItem1 := order.OrderItem{
		OrderID:      order1.ID,
		ProductID:    products[0].ID,
		ProductName:  products[0].Name,
		ProductSKU:   products[0].SKU,
		ProductImage: products[0].ImageURL,
		Quantity:     1,
		UnitPrice:    products[0].Price,
		Subtotal:     products[0].Price,
	}
	db.Create(&orderItem1)

	order2 := order.Order{
		OrderNumber:     "TC-20260826-591823",
		UserID:          sarah.ID,
		TotalAmount:     108.00,
		Status:          order.StatusPaid,
		ShippingName:    "Sarah Jenkins",
		ShippingPhone:   "+1-555-014-9921",
		ShippingAddress: "200 Market Street, Seattle, WA",
		PaymentMethod:   "SIMULATED_CARD",
		PaymentStatus:   order.PaymentSuccess,
		Notes:           "Deliver during business hours",
		CreatedAt:       time.Now().AddDate(0, 0, -1),
	}
	db.Create(&order2)

	orderItem2 := order.OrderItem{
		OrderID:      order2.ID,
		ProductID:    products[3].ID,
		ProductName:  products[3].Name,
		ProductSKU:   products[3].SKU,
		ProductImage: products[3].ImageURL,
		Quantity:     2,
		UnitPrice:    products[3].Price,
		Subtotal:     108.00,
	}
	db.Create(&orderItem2)

	log.Println("Database seeding completed successfully.")
	return nil
}
