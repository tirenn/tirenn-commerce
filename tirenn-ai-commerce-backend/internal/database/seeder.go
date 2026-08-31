package database

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/pgvector/pgvector-go"
	"tirenn-ai-commerce/internal/client/ollama"
	"tirenn-ai-commerce/internal/domain/auth"
	"tirenn-ai-commerce/internal/domain/order"
	"tirenn-ai-commerce/internal/domain/product"
	"tirenn-ai-commerce/internal/security"
	"gorm.io/gorm"
)

type productItemDef struct {
	SubCatIdx   int
	Name        string
	SKU         string
	Price       float64
	Currency    string
	Stock       int
	ImageURL    string
	Badge       string
	Rating      float64
	Description string
}

func Seed(db *gorm.DB, ollamaClients ...*ollama.Client) error {
	var productCount int64
	db.Model(&product.Product{}).Count(&productCount)
	if productCount < 560 {
		return ForceSeed(db, ollamaClients...)
	}

	var nullEmbeddingCount int64
	db.Model(&product.Product{}).Where("embedding IS NULL").Count(&nullEmbeddingCount)
	if nullEmbeddingCount > 0 && len(ollamaClients) > 0 && ollamaClients[0] != nil {
		log.Printf("🧠 Backfilling %d missing product embeddings...", nullEmbeddingCount)
		var prods []product.Product
		db.Where("embedding IS NULL").Find(&prods)

		var ollamaClient = ollamaClients[0]
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()

		numWorkers := 8
		jobs := make(chan *product.Product, len(prods))
		var wg sync.WaitGroup

		for w := 0; w < numWorkers; w++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for prod := range jobs {
					text := fmt.Sprintf("%s. %s", prod.Name, prod.Description)
					vec, err := ollamaClient.GenerateEmbedding(ctx, text)
					if err == nil && len(vec) > 0 {
						pgVec := pgvector.NewVector(vec)
						if err := db.Exec("UPDATE products SET embedding = ? WHERE id = ?", pgVec, prod.ID).Error; err != nil {
							log.Printf("⚠️ Failed to update embedding for product %d: %v", prod.ID, err)
						}
					}
				}
			}()
		}

		for i := range prods {
			jobs <- &prods[i]
		}
		close(jobs)
		wg.Wait()
		log.Println("✅ Missing vector embeddings successfully populated!")
	}
	return nil
}

// ForceSeed clears existing tables and seeds all 560 products, users, categories, and vector embeddings
func ForceSeed(db *gorm.DB, ollamaClients ...*ollama.Client) error {
	// Clean existing products, subcategories, categories to do a fresh 560 product reset (20 ID + 20 EN per subcategory)
	log.Println("🔄 Resetting database tables for fresh 560-product interleaved multi-currency seeding...")
	if err := db.Exec("TRUNCATE TABLE order_items, orders, stock_adjustment_logs, products, sub_categories, categories, users RESTART IDENTITY CASCADE;").Error; err != nil {
		return fmt.Errorf("truncate tables failed: %w", err)
	}

	// 1. Seed Users
	adminPassword, _ := security.HashPassword("Admin@123")
	customerPassword, _ := security.HashPassword("Shopper@123")
	sarahPassword, _ := security.HashPassword("Sarah@123")

	admin := auth.User{
		Name:         "Tirenn Merchant Admin",
		Email:        "admin@gocommerce.com",
		PasswordHash: adminPassword,
		Role:         auth.RoleAdmin,
		Phone:        "+62 811-9876-5432",
		Address:      "Gedung Menara Sudirman Lt. 18, Jl. Jend. Sudirman Kav. 60, Jakarta Selatan",
		Status:       auth.StatusActive,
	}
	db.Create(&admin)

	shopper := auth.User{
		Name:         "Budi Santoso",
		Email:        "shopper@gocommerce.com",
		PasswordHash: customerPassword,
		Role:         auth.RoleCustomer,
		Phone:        "+62 812-3456-7890",
		Address:      "Jl. Menteng Raya No. 24, RT 02/RW 05, Jakarta Pusat",
		Status:       auth.StatusActive,
	}
	db.Create(&shopper)

	sarah := auth.User{
		Name:         "Siti Rahmawati",
		Email:        "sarah.jenkins@example.com",
		PasswordHash: sarahPassword,
		Role:         auth.RoleCustomer,
		Phone:        "+62 813-7890-1234",
		Address:      "Jl. Dago Asri No. 12, Coblong, Bandung, Jawa Barat",
		Status:       auth.StatusActive,
	}
	db.Create(&sarah)

	// 2. Seed 5 Main Categories
	categories := []product.Category{
		{
			Name:        "Elektronik & Gadget",
			Slug:        "elektronik-gadget",
			Description: "Headphone nirkabel, smartwatch kesehatan, keyboard mekanikal, dan aksesoris teknologi terkini.",
			Icon:        "⚡",
		},
		{
			Name:        "Fashion Pria",
			Slug:        "fashion-pria",
			Description: "Koleksi pakaian pria modern, kemeja kasual, jaket streetwear, celana denim, dan sepatu.",
			Icon:        "👔",
		},
		{
			Name:        "Fashion Wanita",
			Slug:        "fashion-wanita",
			Description: "Busana wanita elegan, dress kasual, tas selempang kulit premium, dan sepatu hak/sneakers.",
			Icon:        "👗",
		},
		{
			Name:        "Makanan & Minuman",
			Slug:        "makanan-minuman",
			Description: "Biji kopi sangrai specialty nusantara, teh artisan organik, dan camilan sehat alami.",
			Icon:        "☕",
		},
		{
			Name:        "Kecantikan & Perawatan",
			Slug:        "kecantikan-perawatan",
			Description: "Perawatan kulit wajah natural, serum pencerah, tabir surya SPF 50+, dan parfum aromatik.",
			Icon:        "✨",
		},
	}

	for i := range categories {
		db.Create(&categories[i])
	}

	// 3. Seed 14 Subcategories
	subCatDefs := []struct {
		CategoryIdx int
		Name        string
		Slug        string
		Description string
		Icon        string
	}{
		// Elektronik & Gadget (Cat 0)
		{0, "Audio & Headphone", "audio-headphone", "Headphone bluetooth, TWS earbuds, dan speaker portable.", "🎧"},
		{0, "Smartwatch & Wearables", "smartwatch-wearables", "Jam tangan pintar, fitness tracker, dan gelang kesehatan.", "⌚"},
		{0, "Aksesoris Komputer & Gaming", "aksesoris-komputer-gaming", "Keyboard mekanikal, mouse wireless, webcam, dan perlengkapan meja.", "🎮"},

		// Fashion Pria (Cat 1)
		{1, "Pakaian & Kaos Pria", "pakaian-kaos-pria", "Kaos oversized, kemeja flanel, hoodie, dan polo shirt.", "👕"},
		{1, "Celana & Jeans Pria", "celana-jeans-pria", "Celana denim slim fit, celana chino, dan celana pendek tactical.", "👖"},
		{1, "Sepatu & Sandal Pria", "sepatu-sandal-pria", "Sneakers kasual, sepatu lari ergonomis, dan sandal slop santai.", "👟"},
		{1, "Aksesoris & Dompet Pria", "aksesoris-dompet-pria", "Dompet kulit bifold, ikat pinggang, kacamata, dan topi baseball.", "💼"},

		// Fashion Wanita (Cat 2)
		{2, "Pakaian & Dress Wanita", "pakaian-dress-wanita", "Blouse katun linen, midi dress elegan, cardigan knitwear, dan celana kulot.", "👗"},
		{2, "Tas & Dompet Wanita", "tas-dompet-wanita", "Tas selempang crossbody, tote bag kanvas, shoulder bag, dan clutch.", "👜"},
		{2, "Sepatu & Sandal Wanita", "sepatu-sandal-wanita", "Flat shoes, heels pesta, sneakers fashion, dan sandal slide.", "👠"},

		// Makanan & Minuman (Cat 3)
		{3, "Kopi & Teh Nusantara", "kopi-teh-nusantara", "Biji kopi arabika specialty, cold brew, matcha jepang, dan teh herbal.", "☕"},
		{3, "Camilan & Snack Sehat", "camilan-snack-sehat", "Granola organik, keripik tempe oven, kacang almond panggang, dan madu.", "🍪"},

		// Kecantikan & Perawatan (Cat 4)
		{4, "Skincare & Perawatan Wajah", "skincare-perawatan-wajah", "Serum vitamin C, pelembab ceramide, toner hidrasi, dan sunscreen gel.", "🧴"},
		{4, "Parfum & Body Care", "parfum-body-care", "Eau de Parfum aroma kayu/bunga, sabun mandi aromaterapi, dan body lotion.", "🌸"},
	}

	subCategories := make([]product.SubCategory, len(subCatDefs))
	for i, def := range subCatDefs {
		subCategories[i] = product.SubCategory{
			CategoryID:  categories[def.CategoryIdx].ID,
			Name:        def.Name,
			Slug:        def.Slug,
			Description: def.Description,
			Icon:        def.Icon,
		}
		db.Create(&subCategories[i])
	}

	log.Println("🌱 Seeding 560 interleaved products (alternating categories & ID/EN bilingual products)...")

	// 4. 560 Interleaved Products
	allProductsList := []productItemDef{
		{0, "AuraSound ANC-700 Headphone Nirkabel Peredam Bising", "ID-AUD-001", 1299000.0, "IDR", 35, "https://images.unsplash.com/photo-1505740420928-5e560c06d30e?w=600", "Terlaris", 4.9, "Headphone peredam bising aktif dengan driver 40mm beresolusi tinggi dan baterai tahan hingga 45 jam pemakaian."},
		{0, "Vanguard Elite Hybrid Active Noise-Cancelling Headphones", "EN-AUD-001", 81.19, "USD", 35, "https://images.unsplash.com/photo-1505740420928-5e560c06d30e?w=600", "Best Seller", 4.9, "Engineered with 40mm beryllium drivers and 4-stage adaptive noise cancellation for acoustic purity."},
		{1, "NusantaraFit Pro Jam Tangan Pintar AMOLED GPS", "ID-WCH-001", 899000.0, "IDR", 30, "https://images.unsplash.com/photo-1523275335684-37898b6baf30?w=600", "Terlaris", 4.8, "Smartwatch layar sentuh AMOLED cerah dengan pemantau detak jantung, SpO2, dan GPS rute outdoor presisi."},
		{1, "AeroChronos AMOLED Multisport GPS Smartwatch", "EN-WCH-001", 56.19, "USD", 30, "https://images.unsplash.com/photo-1523275335684-37898b6baf30?w=600", "Best Seller", 4.8, "Ultra-vibrant AMOLED touchscreen smartwatch with all-day biometric health tracking and route GPS."},
		{2, "NusantaraKey Keyboard Mekanikal TKL Switch Merah Linier", "ID-CMP-001", 649000.0, "IDR", 30, "https://images.unsplash.com/photo-1587829741301-dc798b83add3?w=600", "Terlaris", 4.9, "Keyboard mekanikal 87 tombol tanpa keypad angka dengan switch linier halus dan lampu latar RGB."},
		{2, "ApexTypist Tenkeyless Linear Red Switch Mechanical Keyboard", "EN-CMP-001", 40.56, "USD", 30, "https://images.unsplash.com/photo-1587829741301-dc798b83add3?w=600", "Best Seller", 4.9, "Compact 87-key aluminum top-plate mechanical keyboard with smooth hot-swappable red switches."},
		{3, "Kaos Polos Heavyweight Katun Combed 24s Hitam Solid", "ID-MCL-001", 99000.0, "IDR", 60, "https://images.unsplash.com/photo-1521572267360-ee0c2909d518?w=600", "Terlaris", 4.8, "Kaos katun combed 24s tebal anti-terawang berkerah bulat kokoh untuk gaya santai kasual."},
		{3, "Heavyweight 280 GSM Ring-Spun Cotton Crewneck Tee", "EN-MCL-001", 6.19, "USD", 60, "https://images.unsplash.com/photo-1521572267360-ee0c2909d518?w=600", "Best Seller", 4.8, "Durable heavyweight cotton boxy t-shirt with reinforced ribbed collar and seamless sides."},
		{4, "Celana Jeans Pria Slim Fit Denim Melar Indigo Gelap", "ID-MPN-001", 249000.0, "IDR", 40, "https://images.unsplash.com/photo-1542272604-780c96856592?w=600", "Terlaris", 4.8, "Denim melar 13oz warna indigo gelap jahitan rantai kuat tahan robek untuk aktivitas harian."},
		{4, "Tailored Slim-Fit Stretch Denim Jeans in Deep Indigo", "EN-MPN-001", 15.56, "USD", 40, "https://images.unsplash.com/photo-1542272604-780c96856592?w=600", "Best Seller", 4.8, "Classic five-pocket slim indigo jeans engineered with 2% elastane for unrestricted flex."},
		{5, "Sepatu Sneakers Kanvas Pria Sol Vulkanisir Hitam Putih", "ID-MSH-001", 289000.0, "IDR", 40, "https://images.unsplash.com/photo-1549298916-b41d501d3772?w=600", "Terlaris", 4.8, "Sneakers kasual sol karet vulkanisir tahan lama dengan insole busa empuk nyaman dipakai harian."},
		{5, "Classic Low-Top Vulcanized Canvas Skate Sneakers Black/White", "EN-MSH-001", 18.06, "USD", 40, "https://images.unsplash.com/photo-1549298916-b41d501d3772?w=600", "Best Seller", 4.8, "Timeless low-top canvas sneakers featuring heavy-duty foxing tape and durable gum rubber tread."},
		{6, "Dompet Pria Kulit Sapi Asli Lipat Dua Cokelat Kopi", "ID-MAC-001", 149000.0, "IDR", 50, "https://images.unsplash.com/photo-1627123424574-724758594e93?w=600", "Terlaris", 4.8, "Dompet kulit sapi asli dengan 8 slot kartu, 2 ruang uang kertas, dan lapisan pelindung RFID anti-pencurian data."},
		{6, "Full-Grain Leather Bifold Wallet with RFID Blocking", "EN-MAC-001", 9.31, "USD", 50, "https://images.unsplash.com/photo-1627123424574-724758594e93?w=600", "Best Seller", 4.8, "Handmade vegetable-tanned leather bifold featuring built-in RFID frequency shield and dual cash pockets."},
		{7, "Dress Midi Brokat Katun Lengan Balon Warna Lilac", "ID-WCL-001", 259000.0, "IDR", 30, "https://images.unsplash.com/photo-1515886657613-9f3515b0c78f?w=600", "Terlaris", 4.9, "Gaun midi wanita dengan brokat halus, furing katun adem, dan lengan balon manis untuk pesta dan kondangan."},
		{7, "Floral Wrap Midi Dress with Flounce Ruffle Hem", "EN-WCL-001", 16.19, "USD", 30, "https://images.unsplash.com/photo-1515886657613-9f3515b0c78f?w=600", "Best Seller", 4.9, "Romantic wrap dress in breathable chiffon featuring adjustable waist tie and flowing tiered hem."},
		{8, "Tas Selempang Wanita Kulit Sintetis Tekstur Jeruk Moka", "ID-WBG-001", 189000.0, "IDR", 40, "https://images.unsplash.com/photo-1584917865442-de89df76afd3?w=600", "Terlaris", 4.8, "Tas selempang crossbody kulit sintetis lembut dengan tali panjang lepas pasang dan resleting emas."},
		{8, "Structured Leather Crossbody Bag with Turnlock Gold", "EN-WBG-001", 11.81, "USD", 40, "https://images.unsplash.com/photo-1584917865442-de89df76afd3?w=600", "Best Seller", 4.8, "Refined pebble-grain crossbody bag with twist-lock closure, interior slip pockets, and adjustable shoulder strap."},
		{9, "Flat Shoes Balet Lipat Kulit Sintetis Busa Empuk Nude", "ID-WSH-001", 159000.0, "IDR", 50, "https://images.unsplash.com/photo-1543163521-1bf539c55dd2?w=600", "Terlaris", 4.8, "Sepatu teplek balet dengan insole busa empuk elastis yang tidak membuat telapak kaki lelah saat kerja."},
		{9, "Pointed-Toe Knit Ballet Flats with Memory Foam Insole", "EN-WSH-001", 9.94, "USD", 50, "https://images.unsplash.com/photo-1543163521-1bf539c55dd2?w=600", "Best Seller", 4.8, "Eco-conscious knit pointed-toe ballet flats featuring padded arch support and flexible rubber sole."},
		{10, "Biji Kopi Arabika Gayo Aceh Single Origin Sangrai Sedang 250g", "ID-COF-001", 89000.0, "IDR", 60, "https://images.unsplash.com/photo-1514432324607-a09d9b4aefdd?w=600", "Terlaris", 4.9, "Biji kopi arabika dataran tinggi Gayo Aceh dengan notes rasa rempah herbal, cokelat hitam, dan keasaman seimbang."},
		{10, "Ethiopian Yirgacheffe Light Roast Washed Whole Bean 12oz", "EN-COF-001", 5.56, "USD", 60, "https://images.unsplash.com/photo-1514432324607-a09d9b4aefdd?w=600", "Best Seller", 4.9, "Renowned single-origin coffee offering floral jasmine aromatics, vibrant bergamot notes, and clean finish."},
		{11, "Granola Madu Hutan Tropis Kacang Mete & Biji Bunga Matahari", "ID-SNK-001", 65000.0, "IDR", 60, "https://images.unsplash.com/photo-1599599810769-bcde5a160d32?w=600", "Terlaris", 4.9, "Granola gandum utuh panggang madu lebah liar dengan taburan kacang mete Bali dan kismis manis."},
		{11, "Honey Toasted Pecan & Cranberry Crunchy Granola 12oz", "EN-SNK-001", 4.06, "USD", 60, "https://images.unsplash.com/photo-1599599810769-bcde5a160d32?w=600", "Best Seller", 4.9, "Small-batch baked rolled oats tossed with wild wildflower honey, roasted pecans, and dried cranberries."},
		{12, "Serum Pencerah Wajah Vitamin C 10% Ekstrak Jeruk Kakadu", "ID-SKN-001", 139000.0, "IDR", 45, "https://images.unsplash.com/photo-1556228720-195a672e8a03?w=600", "Terlaris", 4.9, "Serum antioksidan pencerah noda hitam dengan turunan vitamin C stabil dan ekstrak botani alami."},
		{12, "Radiance Glow Vitamin C 15% Brightening Face Serum 30ml", "EN-SKN-001", 8.69, "USD", 45, "https://images.unsplash.com/photo-1556228720-195a672e8a03?w=600", "Best Seller", 4.9, "Potent L-Ascorbic Acid formula stabilized with Ferulic Acid and Vitamin E for luminous tone correction."},
		{13, "Eau de Parfum Cendana Jawa Aroma Kayu Hangat Mewah 50ml", "ID-BDY-001", 219000.0, "IDR", 35, "https://images.unsplash.com/photo-1592945403244-b3fbafd7f539?w=600", "Terlaris", 4.9, "Parfum aroma kayu cendana khas Jawa berpadu kapulaga dan amber hangat tahan hingga 10 jam."},
		{13, "Smoked Sandalwood & Cedarwood Eau de Parfum 50ml", "EN-BDY-001", 13.69, "USD", 35, "https://images.unsplash.com/photo-1592945403244-b3fbafd7f539?w=600", "Best Seller", 4.9, "Sophisticated unisex woody perfume blending dry cedar, smoky sandalwood, and warm amber resin."},
		{0, "SuaraNusantara TWS Earbuds Nirkabel Bass Menggelegar", "ID-AUD-002", 349000.0, "IDR", 50, "https://images.unsplash.com/photo-1590658268037-6bf12165a8df?w=600", "Paling Dicari", 4.8, "Earbuds bluetooth 5.3 berdesain ergonomis dengan dynamic bass booster dan sertifikasi tahan keringat IPX5."},
		{0, "PulseBeat Stealth True Wireless Earbuds with Low-Latency Mode", "EN-AUD-002", 21.81, "USD", 50, "https://images.unsplash.com/photo-1590658268037-6bf12165a8df?w=600", "Trending", 4.8, "Compact in-ear wireless earphones featuring custom graph-diaphragm drivers and gaming sound sync."},
		{1, "RagaSehat Gelang Kebugaran Tahan Air Berenang 50m", "ID-WCH-002", 249000.0, "IDR", 60, "https://images.unsplash.com/photo-1575311373937-040b8e1fd5b6?w=600", "Populer", 4.7, "Smartband ramping penghitung langkah, kualitas tidur, dan 30 mode latihan fisik dengan ketahanan 5 ATM."},
		{1, "CorePulse 5ATM Waterproof Swim & Fitness Wristband", "EN-WCH-002", 15.56, "USD", 60, "https://images.unsplash.com/photo-1575311373937-040b8e1fd5b6?w=600", "Popular", 4.7, "Sleek fitness tracker featuring 30 workout profiles, continuous heart monitoring, and 14-day battery life."},
		{2, "KilatGerak Mouse Gaming Nirkabel Ringan Sarang Lebah", "ID-CMP-002", 429000.0, "IDR", 40, "https://images.unsplash.com/photo-1615663245857-ac93bb7c39e7?w=600", "Sedang Tren", 4.8, "Mouse nirkabel seringan 58 gram dengan sensor optik 16000 DPI untuk respon membidik cepat."},
		{2, "ViperGlide Ultra-Lightweight Honeycomb Shell Gaming Mouse", "EN-CMP-002", 26.81, "USD", 40, "https://images.unsplash.com/photo-1615663245857-ac93bb7c39e7?w=600", "Trending", 4.8, "Ergonomic 58-gram tournament wireless mouse powered by a flawless 16,000 DPI tracking sensor."},
		{3, "Kemeja Batik Tulis Solo Motif Parang Rusak Katun Halus", "ID-MCL-002", 289000.0, "IDR", 25, "https://images.unsplash.com/photo-1596755094514-f87e34085b2c?w=600", "Pilihan Editor", 4.9, "Batik tulis khas Solo dengan pewarnaan alami dan potongan kemeja pria berfuring sutra dingin."},
		{3, "Heritage Buffalo Plaid Brushed Cotton Overshirt", "EN-MCL-002", 12.44, "USD", 35, "https://images.unsplash.com/photo-1596755094514-f87e34085b2c?w=600", "Staff Pick", 4.8, "Heavy brushed flannel button-up shirt featuring dual chest flap pockets and horn buttons."},
		{4, "Celana Chino Pria Slim Fit Katun Twill Stretch Khaki", "ID-MPN-002", 199000.0, "IDR", 50, "https://images.unsplash.com/photo-1624378439575-d8705ad7ae80?w=600", "Populer", 4.8, "Celana bahan katun twill elastis nyaman dipakai untuk bekerja di kantor maupun nongkrong kasual."},
		{4, "Smart-Casual Wrinkle-Free Flat-Front Chino Pants Khaki", "EN-MPN-002", 12.44, "USD", 50, "https://images.unsplash.com/photo-1624378439575-d8705ad7ae80?w=600", "Popular", 4.8, "Pre-washed breathable cotton twill trousers tailored with clean lines for desk-to-dinner transitions."},
		{5, "Sepatu Lari Ringan Bantalan Udara Responsif Abu Perak", "ID-MSH-002", 399000.0, "IDR", 35, "https://images.unsplash.com/photo-1542291026-7eec264c27ff?w=600", "Sedang Tren", 4.9, "Sepatu lari sol responsif busa EVA dengan rajutan atas bernapas dan cengkeraman anti-selip."},
		{5, "Performance Cushioning Mesh Road Running Shoes Silver", "EN-MSH-002", 24.94, "USD", 35, "https://images.unsplash.com/photo-1542291026-7eec264c27ff?w=600", "Trending", 4.9, "Engineered open-weave road runners boasting high-rebound EVA foam for energetic toe-offs."},
		{6, "Sabuk Kulit Formal Kepala Gesper Logam Krom Hitam", "ID-MAC-002", 119000.0, "IDR", 60, "https://images.unsplash.com/photo-1627123424574-724758594e93?w=600", "Populer", 4.8, "Ikat pinggang kulit pria formal dengan kepala gesper tusuk logam krom mengkilap anti-karat."},
		{6, "Heavy-Duty Reversible Leather Dress Belt Gunmetal", "EN-MAC-002", 7.44, "USD", 60, "https://images.unsplash.com/photo-1627123424574-724758594e93?w=600", "Popular", 4.8, "Dual-sided black/brown leather belt with 360-degree rotating gunmetal prong buckle."},
		{7, "Blouse Katun Rayon Adem Kerah V Warna Putih Bersih", "ID-WCL-002", 149000.0, "IDR", 45, "https://images.unsplash.com/photo-1515886657613-9f3515b0c78f?w=600", "Populer", 4.8, "Atasan wanita kerah V berbahan katun rayon jatuh yang sangat sejuk dan cocok untuk kerja kantor."},
		{7, "Relaxed Linen Button-Down Resort Blouse Coral", "EN-WCL-002", 9.31, "USD", 45, "https://images.unsplash.com/photo-1515886657613-9f3515b0c78f?w=600", "Popular", 4.8, "Airy pre-washed European linen shirt tailored with dropped shoulders and mother-of-pearl buttons."},
		{8, "Tote Bag Kanvas Tebal Muat Laptop 14 Inci Aksen Kulit", "ID-WBG-002", 159000.0, "IDR", 50, "https://images.unsplash.com/photo-1584917865442-de89df76afd3?w=600", "Populer", 4.8, "Tas jinjing kanvas katun tebal dengan lapisan busa pelindung laptop dan saku tumbler botol minum."},
		{8, "Canvas Utility Tote Bag with Padded 15-Inch Laptop Compartment", "EN-WBG-002", 9.94, "USD", 50, "https://images.unsplash.com/photo-1584917865442-de89df76afd3?w=600", "Popular", 4.8, "Heavyweight cotton canvas work tote featuring interior water bottle holder and reinforced leather handles."},
		{9, "Sepatu Hak Tahu Block Heels 5cm Pesta Warna Krem", "ID-WSH-002", 229000.0, "IDR", 35, "https://images.unsplash.com/photo-1543163521-1bf539c55dd2?w=600", "Populer", 4.8, "Sepatu hak kokoh 5cm dengan tali pergelangan kaki anggun cocok untuk pesta pernikahan dan kantor."},
		{9, "Block-Heel Ankle-Strap Evening Pumps 2-Inch Nude", "EN-WSH-002", 14.31, "USD", 35, "https://images.unsplash.com/photo-1543163521-1bf539c55dd2?w=600", "Popular", 4.8, "Versatile 2-inch block heel dress pumps with slender buckle ankle strap and cushioned footbed."},
		{10, "Biji Kopi Arabika Toraja Enrekang Aroma Rempah Buah 250g", "ID-COF-002", 95000.0, "IDR", 50, "https://images.unsplash.com/photo-1514432324607-a09d9b4aefdd?w=600", "Pilihan Editor", 4.9, "Kopi specialty dari lereng pegunungan Sesean Sulawesi dengan sentuhan rasa buah manis dan karamel tebal."},
		{10, "Colombian Supremo Medium Roast Single-Origin Coffee 12oz", "EN-COF-002", 5.94, "USD", 50, "https://images.unsplash.com/photo-1514432324607-a09d9b4aefdd?w=600", "Popular", 4.8, "Smooth, balanced high-altitude coffee featuring notes of sweet brown sugar, toasted walnut, and milk chocolate."},
		{11, "Keripik Tempe Oven Bumbu Bawang Ketumbar Gurih Renyah", "ID-SNK-002", 28000.0, "IDR", 90, "https://images.unsplash.com/photo-1599599810769-bcde5a160d32?w=600", "Paling Dicari", 4.8, "Keripik tempe kedelai non-GMO dipanggang oven bebas minyak jenuh dengan bumbu bawang putih asli."},
		{11, "Dry-Roasted Whole Unsalted California Almonds 16oz", "EN-SNK-002", 4.69, "USD", 50, "https://images.unsplash.com/photo-1599599810769-bcde5a160d32?w=600", "Top Rated", 4.9, "Oven-roasted supreme-sized whole almonds offering satisfying natural crunch without added sodium."},
		{12, "Pelembab Wajah Ceramide & Asam Hialuronat Pelindung Kulit", "ID-SKN-002", 159000.0, "IDR", 40, "https://images.unsplash.com/photo-1556228720-195a672e8a03?w=600", "Paling Dicari", 4.9, "Gel pelembab bertekstur seringan air yang mengunci hidrasi 24 jam dan memperbaiki skin barrier rusak."},
		{12, "Barrier Repair Barrier-Building Ceramide Moisturizer 50ml", "EN-SKN-002", 9.94, "USD", 40, "https://images.unsplash.com/photo-1556228720-195a672e8a03?w=600", "Top Rated", 4.9, "Triple ceramide barrier repair cream restoring moisture retention and calming compromised skin."},
		{13, "Eau de Parfum Melati Keraton Bunga Putih Elegan 50ml", "ID-BDY-002", 199000.0, "IDR", 40, "https://images.unsplash.com/photo-1592945403244-b3fbafd7f539?w=600", "Paling Dicari", 4.8, "Aroma klasik semerbak bunga melati putih segar dengan sentuhan teh hijau dan vanila lembut."},
		{13, "Velvet Rose & Amber Luxury Artisan Fragrance 50ml", "EN-BDY-002", 12.44, "USD", 40, "https://images.unsplash.com/photo-1592945403244-b3fbafd7f539?w=600", "Top Rated", 4.8, "Deep romantic blend of Damask rose petals, spiced clove, patchouli, and golden amber."},
		{0, "NadaStudio Open-Back Headphone Monitor Audiophile", "ID-AUD-003", 1850000.0, "IDR", 20, "https://images.unsplash.com/photo-1546435770-a3e426bf472b?w=600", "Pilihan Editor", 4.9, "Headphone akustik terbuka untuk mixing audio rekaman dan mastering musik dengan detail suara transparan."},
		{0, "ReferenceLine Acoustic Open-Back Studio Mastering Headphones", "EN-AUD-003", 115.62, "USD", 20, "https://images.unsplash.com/photo-1546435770-a3e426bf472b?w=600", "Staff Pick", 4.9, "Precision-engineered planar magnetic studio monitor headphones with transparent staging."},
		{1, "SatriaChrono Jam Tangan Pintar Bezel Baja Kaca Safir", "ID-WCH-003", 1450000.0, "IDR", 20, "https://images.unsplash.com/photo-1546868871-7041f2a55e12?w=600", "Pilihan Editor", 4.9, "Desain formal klasik eksekutif berbalut baja tahan karat dan tali kulit sapi asli anti-gores."},
		{1, "ImperialSteel Sapphire Crystal Luxury Smart Timepiece", "EN-WCH-003", 90.62, "USD", 20, "https://images.unsplash.com/photo-1546868871-7041f2a55e12?w=600", "Staff Pick", 4.9, "Executive dress smartwatch enclosed in 316L stainless steel with sapphire crystal glass dial."},
		{2, "SorotLensa Kamera Siaran Langsung USB Full HD 60FPS", "ID-CMP-003", 389000.0, "IDR", 35, "https://images.unsplash.com/photo-1587829741301-dc798b83add3?w=600", "Populer", 4.7, "Webcam fokus otomatis dengan mikrofon ganda penyaring kebisingan untuk rapat Zoom dan streaming."},
		{2, "StreamVision Full HD 1080p 60FPS Broadcaster Webcam", "EN-CMP-003", 24.31, "USD", 35, "https://images.unsplash.com/photo-1587829741301-dc798b83add3?w=600", "Popular", 4.7, "Crystal clear autofocus streaming webcam equipped with dual noise-reduction microphones."},
		{3, "Kemeja Flanel Katun Kotak-Kotak Merah Marun Hangat", "ID-MCL-003", 199000.0, "IDR", 35, "https://images.unsplash.com/photo-1596755094514-f87e34085b2c?w=600", "Paling Dicari", 4.8, "Kemeja flanel berbulu lembut dengan saku ganda di dada cocok untuk nongkrong di kafe."},
		{3, "Classic Oxford Cloth Button-Down Work Shirt Navy", "EN-MCL-003", 11.81, "USD", 40, "https://images.unsplash.com/photo-1596755094514-f87e34085b2c?w=600", "Popular", 4.8, "Timeless breathable cotton Oxford shirt tailored with a neat curved hem for smart-casual wear."},
		{4, "Celana Kargo Panjang Taktikal 6 Saku Ripstop Hitam", "ID-MPN-003", 229000.0, "IDR", 35, "https://images.unsplash.com/photo-1542272604-780c96856592?w=600", "Sedang Tren", 4.7, "Celana kargo bahan ripstop anti-robek dengan banyak kantong serbaguna untuk aktivitas outdoor."},
		{4, "Tactical Multi-Pocket Ripstop Cargo Pants Matte Black", "EN-MPN-003", 14.31, "USD", 35, "https://images.unsplash.com/photo-1542272604-780c96856592?w=600", "Trending", 4.7, "Durable weather-treated ripstop cargo trousers featuring dual bellowed utility thigh pockets."},
		{5, "Sepatu Formal Kulit Derby Kilap Ujung Polos Hitam", "ID-MSH-003", 459000.0, "IDR", 25, "https://images.unsplash.com/photo-1533867617858-e7b97e060509?w=600", "Pilihan Editor", 4.9, "Sepatu kerja formal kulit sintetis premium dengan jahitan welt rapi dan sol kayu bertumit."},
		{5, "Formal Polished Leather Plain-Toe Derby Shoes Black", "EN-MSH-003", 28.69, "USD", 25, "https://images.unsplash.com/photo-1533867617858-e7b97e060509?w=600", "Staff Pick", 4.9, "Executive dress derbies constructed with cushioned insole and clean blind eyelet lacing."},
		{6, "Kacamata Hitam Polarized Lensa Gelap Anti Sinar UV400", "ID-MAC-003", 139000.0, "IDR", 45, "https://images.unsplash.com/photo-1511499767150-a48a237f0083?w=600", "Sedang Tren", 4.7, "Kacamata hitam gaya wayfarer klasik dengan lensa polarisasi peredam silau matahari saat menyetir."},
		{6, "Polarized UV400 Wayfarer Sunglasses Matte Black", "EN-MAC-003", 8.69, "USD", 45, "https://images.unsplash.com/photo-1511499767150-a48a237f0083?w=600", "Trending", 4.7, "Classic retro wayfarer silhouette frames fitted with anti-glare TAC polarized dark tint lenses."},
		{7, "Cardigan Rajut Crop Top Kancing Batok Kelapa Mustard", "ID-WCL-003", 179000.0, "IDR", 35, "https://images.unsplash.com/photo-1515886657613-9f3515b0c78f?w=600", "Sedang Tren", 4.8, "Outer rajut model crop kasual dengan kancing batok kelapa alami untuk paduan gaya santai."},
		{7, "Chunky Cable-Knit Open-Front Longline Cardigan Camel", "EN-WCL-003", 11.19, "USD", 35, "https://images.unsplash.com/photo-1515886657613-9f3515b0c78f?w=600", "Trending", 4.8, "Cozy textured oversized knit duster cardigan detailed with deep front patch pockets."},
		{8, "Shoulder Bag Klasik Tali Rantai Emas Hitam Elegan", "ID-WBG-003", 219000.0, "IDR", 30, "https://images.unsplash.com/photo-1584917865442-de89df76afd3?w=600", "Sedang Tren", 4.9, "Tas bahu quilted kulit sintetis empuk dengan tali rantai emas mewah cocok untuk pesta dan kerja."},
		{8, "Quilted Chain-Strap Evening Shoulder Bag Caviar Black", "EN-WBG-003", 13.69, "USD", 30, "https://images.unsplash.com/photo-1584917865442-de89df76afd3?w=600", "Trending", 4.9, "Diamond-quilted shoulder bag accented with polished gold-tone chain link strap and magnetic snap flap."},
		{9, "Sneakers Kasual Wanita Sol Tebal Ringan Putih Pink Pastel", "ID-WSH-003", 269000.0, "IDR", 35, "https://images.unsplash.com/photo-1543163521-1bf539c55dd2?w=600", "Sedang Tren", 4.9, "Sneakers platform wanita sol tebal 4cm yang empuk dan ringan menambah tinggi badan secara natural."},
		{9, "Chunky Retro Platform Leather Sneakers White/Pastel", "EN-WSH-003", 16.81, "USD", 35, "https://images.unsplash.com/photo-1543163521-1bf539c55dd2?w=600", "Trending", 4.9, "90s-inspired platform fashion sneakers with lightweight EVA midsole and pastel suede color blocking."},
		{10, "Kopi Robusta Temanggung Sangrai Gelap Mantap 250g", "ID-COF-003", 65000.0, "IDR", 70, "https://images.unsplash.com/photo-1514432324607-a09d9b4aefdd?w=600", "Populer", 4.8, "Robusta petik merah murni dari lereng Gunung Sindoro dengan krema tebal gurih dan kafein mantap."},
		{10, "Dark Roast French Roast Organic Whole Bean Coffee 16oz", "EN-COF-003", 4.06, "USD", 70, "https://images.unsplash.com/photo-1514432324607-a09d9b4aefdd?w=600", "Top Rated", 4.8, "Intense full-bodied dark roast delivering smoky dark chocolate richness and heavy crema."},
		{11, "Kacang Almond Panggang Tanpa Garam Oven Matang 250g", "ID-SNK-003", 75000.0, "IDR", 50, "https://images.unsplash.com/photo-1599599810769-bcde5a160d32?w=600", "Pilihan Editor", 4.9, "Kacang almond utuh California dipanggang kering tanpa minyak dan garam tambahan, tinggi vitamin E."},
		{11, "Organic Raw Wildflower Honey Pure Glass Jar 16oz", "EN-SNK-003", 7.19, "USD", 35, "https://images.unsplash.com/photo-1599599810769-bcde5a160d32?w=600", "Premium", 4.9, "Unfiltered pure honey harvested from mountain bee apiaries retaining natural pollen and enzymes."},
		{12, "Tabir Surya Gel Sunscreen SPF 50+ PA++++ Tekstur Air", "ID-SKN-003", 129000.0, "IDR", 50, "https://images.unsplash.com/photo-1556228720-195a672e8a03?w=600", "Terlaris", 4.9, "Sunscreen ringan tanpa whitecast tidak menyumbat pori diperkaya ekstrak lidah buaya penyejuk."},
		{12, "Ultra-Light Watery Sunscreen Gel Broad Spectrum SPF 50+", "EN-SKN-003", 8.06, "USD", 50, "https://images.unsplash.com/photo-1556228720-195a672e8a03?w=600", "Best Seller", 4.9, "Zero-cast weightless water-gel sun shield protecting against UVA/UVB rays with matte finish."},
		{13, "Sabun Mandi Cair Aromaterapi Minyak Serai dan Jahe 300ml", "ID-BDY-003", 79000.0, "IDR", 60, "https://images.unsplash.com/photo-1592945403244-b3fbafd7f539?w=600", "Populer", 4.8, "Body wash busa melimpah dengan minyak esensial serai alami menghangatkan tubuh dan meredakan lelah."},
		{13, "Revitalizing Eucalyptus & Spearmint Aromatherapy Body Wash 300ml", "EN-BDY-003", 4.94, "USD", 60, "https://images.unsplash.com/photo-1592945403244-b3fbafd7f539?w=600", "Popular", 4.8, "Invigorating botanical shower gel enriched with pure eucalyptus essential oils clearing the senses."},
		{0, "DentumMini Speaker Portabel Bluetooth Tahan Air", "ID-AUD-004", 289000.0, "IDR", 45, "https://images.unsplash.com/photo-1608043152269-423dbba4e7e1?w=600", "Sedang Tren", 4.7, "Speaker mini ringkas bertenaga suara lantang dengan radiator pasif ganda dan tali gantung fleksibel."},
		{0, "RoverShield Rugged Waterproof Outdoor Bluetooth Speaker", "EN-AUD-004", 18.06, "USD", 45, "https://images.unsplash.com/photo-1608043152269-423dbba4e7e1?w=600", "Top Rated", 4.7, "Shockproof all-weather portable speaker delivering 360-degree expansive bass and 16-hour playtime."},
		{1, "CincinPintar Vital Titanium Pemantau Suhu Tubuh", "ID-WCH-004", 1199000.0, "IDR", 15, "https://images.unsplash.com/photo-1605100804763-247f67b3557e?w=600", "Produk Baru", 4.8, "Cincin pintar bahan titanium ultra-ringan pemantau fase tidur REM dan variabilitas detak jantung."},
		{1, "BioRing Titanium Sleep Architecture & HRV Health Ring", "EN-WCH-004", 74.94, "USD", 15, "https://images.unsplash.com/photo-1605100804763-247f67b3557e?w=600", "New Arrival", 4.8, "Featherweight titanium finger ring measuring sleep cycles, temperature fluctuations, and recovery indexes."},
		{2, "HamparanMeja Alas Mouse Jumbo Meja Kantor 90x40cm", "ID-CMP-004", 119000.0, "IDR", 70, "https://images.unsplash.com/photo-1615663245857-ac93bb7c39e7?w=600", "Wajib Punya", 4.8, "Deskmat permukaan kain mikro lembut dengan jahitan tepi obras rapat dan dasar karet anti-selip."},
		{2, "DeskMatrix Extended XL Micro-Weave Gaming Surface 90x40cm", "EN-CMP-004", 7.44, "USD", 70, "https://images.unsplash.com/photo-1615663245857-ac93bb7c39e7?w=600", "Essential", 4.8, "Spacious water-resistant desk mat featuring reinforced perimeter stitching and anti-slip natural rubber base."},
		{3, "Hoodie Streetwear Katun Fleece Tebal Abu-Abu Misty", "ID-MCL-004", 259000.0, "IDR", 30, "https://images.unsplash.com/photo-1556905055-8f358a7a47b2?w=600", "Sedang Tren", 4.9, "Hoodie pullover bertali tebal dengan saku kangguru lapang dan rib elastis di ujung lengan."},
		{3, "Vintage Washed 14oz Denim Trucker Jacket Indigo", "EN-MCL-004", 15.56, "USD", 30, "https://images.unsplash.com/photo-1596755094514-f87e34085b2c?w=600", "Top Rated", 4.8, "Authentic rigid denim jacket featuring copper shank buttons, chest pockets, and waist adjusters."},
		{4, "Celana Ankle Pants Pria Semi-Wool Potongan Rapi Abu", "ID-MPN-004", 279000.0, "IDR", 30, "https://images.unsplash.com/photo-1624378439575-d8705ad7ae80?w=600", "Pilihan Editor", 4.9, "Celana panjang formal potongan mata kaki modern tanpa lipatan bawah berkesan bersih rapi."},
		{4, "Modern Cropped Ankle Trousers in Charcoal Wool Blend", "EN-MPN-004", 17.44, "USD", 30, "https://images.unsplash.com/photo-1624378439575-d8705ad7ae80?w=600", "Staff Pick", 4.9, "Sharp tailored ankle-length trousers with pressed center creases and internal gripper waistband."},
		{5, "Sepatu Bot Gunung Sol Bergerigi Tahan Air Cokelat Tua", "ID-MSH-004", 499000.0, "IDR", 20, "https://images.unsplash.com/photo-1542291026-7eec264c27ff?w=600", "Paling Dicari", 4.8, "Sepatu hiking tahan air dengan pelindung ujung jari karet tebal dan sol cengkeram lumpur."},
		{5, "All-Terrain Waterproof Lugged Trail Hiking Boots Brown", "EN-MSH-004", 31.19, "USD", 20, "https://images.unsplash.com/photo-1542291026-7eec264c27ff?w=600", "Top Rated", 4.8, "Weatherproof ankle-height trail boots equipped with rock protection toe bumper and deep lugs."},
		{6, "Topi Baseball Katun Aksen Bordir Tali Logam Hitam", "ID-MAC-004", 79000.0, "IDR", 75, "https://images.unsplash.com/photo-1588850561407-ed78c282e89b?w=600", "Terlaris", 4.7, "Topi baseball katun twill tebal dengan gesper pengatur lingkar kepala kuningan vintage."},
		{6, "Washed Cotton Vintage Dad Hat with Brass Buckle Navy", "EN-MAC-004", 4.94, "USD", 75, "https://images.unsplash.com/photo-1588850561407-ed78c282e89b?w=600", "Best Seller", 4.7, "Unstructured 6-panel low-profile dad cap made from soft garment-washed cotton twill."},
		{7, "Celana Kulot Katun Linen Pinggang Karet Warna Moka", "ID-WCL-004", 169000.0, "IDR", 50, "https://images.unsplash.com/photo-1515886657613-9f3515b0c78f?w=600", "Favorit", 4.7, "Celana kulot wanita potongan longgar bahan katun linen sejuk dengan saku samping fungsional."},
		{7, "High-Waisted Wide-Leg Pleated Palazzo Pants Beige", "EN-WCL-004", 10.56, "USD", 50, "https://images.unsplash.com/photo-1515886657613-9f3515b0c78f?w=600", "Top Rated", 4.7, "Flowy high-rise trousers featuring knife pleats and an elastic back waistband for tailored elegance."},
		{8, "Dompet Lipat Tiga Wanita Slot Koin Ritsleting Salem", "ID-WBG-004", 99000.0, "IDR", 70, "https://images.unsplash.com/photo-1584917865442-de89df76afd3?w=600", "Hemat", 4.7, "Dompet lipat mungil muat banyak kartu dengan slot koin beritsleting dan tempat foto transparan."},
		{8, "Envelope Leather Trifold Wallet with Coin Zip Pocket Berry", "EN-WBG-004", 6.19, "USD", 70, "https://images.unsplash.com/photo-1584917865442-de89df76afd3?w=600", "Budget Pick", 4.7, "Compact trifold wallet featuring billfold compartment, 6 card slots, and an exterior zippered change pocket."},
		{9, "Sandal Slide Wanita Tali Silang Busa Empuk Warna Moka", "ID-WSH-004", 119000.0, "IDR", 60, "https://images.unsplash.com/photo-1543163521-1bf539c55dd2?w=600", "Favorit", 4.7, "Sandal selop santai wanita dengan tali silang kulit sintetis empuk dan sol anti-licin."},
		{9, "Padded Crisscross Band Leather Slide Sandals Tan", "EN-WSH-004", 7.44, "USD", 60, "https://images.unsplash.com/photo-1543163521-1bf539c55dd2?w=600", "Top Rated", 4.7, "Supple padded leather slide sandals offering effortless slip-on comfort and non-skid traction."},
		{10, "Kopi Arabika Flores Bajawa Cita Rasa Cokelat Manis 250g", "ID-COF-004", 92000.0, "IDR", 45, "https://images.unsplash.com/photo-1514432324607-a09d9b4aefdd?w=600", "Sedang Tren", 4.8, "Biji kopi arabika tanah vulkanik Flores dengan aroma kacang panggang, tembakau manis, dan cokelat legit."},
		{10, "Costa Rican Tarrazu Honey Process Specialty Beans 12oz", "EN-COF-004", 5.75, "USD", 45, "https://images.unsplash.com/photo-1514432324607-a09d9b4aefdd?w=600", "Staff Pick", 4.9, "Honey-processed Arabica beans showcasing silky stone fruit sweetness and honeyed aftertaste."},
		{11, "Madu Hutan Murni Sumbawa Asli Botol Kaca 500g", "ID-SNK-004", 115000.0, "IDR", 35, "https://images.unsplash.com/photo-1599599810769-bcde5a160d32?w=600", "Produk Unggulan", 4.9, "Madu mentah lebah hutan liar pohon bidara Sumbawa tanpa pemanasan buatan menjaga enzim alami tubuh."},
		{11, "Freeze-Dried Crunchy Organic Strawberry Slices 3oz", "EN-SNK-004", 2.81, "USD", 55, "https://images.unsplash.com/photo-1599599810769-bcde5a160d32?w=600", "Popular", 4.8, "100% pure ripe strawberry slices gently freeze-dried to preserve natural flavor and vitamin C."},
		{12, "Sabun Cuci Muka Busa Lembut Ekstrak Centella Asiatica", "ID-SKN-004", 89000.0, "IDR", 65, "https://images.unsplash.com/photo-1556228720-195a672e8a03?w=600", "Populer", 4.8, "Facial wash pH rendah 5.5 membersihkan minyak dan kotoran tanpa membuat kulit terasa kering tertarik."},
		{12, "Gentle Hydrating Amino Acid Foam Cleanser 150ml", "EN-SKN-004", 5.56, "USD", 65, "https://images.unsplash.com/photo-1556228720-195a672e8a03?w=600", "Popular", 4.8, "Sulfate-free micro-bubble facial wash removing impurities while preserving natural lipid moisture barrier."},
		{13, "Losion Tubuh Pencerah Niacinamide Ekstrak Susu Kambing 250ml", "ID-BDY-004", 89000.0, "IDR", 55, "https://images.unsplash.com/photo-1592945403244-b3fbafd7f539?w=600", "Terlaris", 4.8, "Body lotion cepat meresap dengan kandungan susu kambing melembabkan kulit kering dan mencerahkan."},
		{13, "Ultra-Hydrating Raw Shea Butter & Oat Body Lotion 250ml", "EN-BDY-004", 5.56, "USD", 55, "https://images.unsplash.com/photo-1592945403244-b3fbafd7f539?w=600", "Best Seller", 4.8, "Fast-absorbing restorative body moisturizer locking in deep hydration for dry, parched skin."},
		{0, "BioskopRumah Soundbar Dolby Atmos 2.1 dengan Subwoofer", "ID-AUD-005", 1499000.0, "IDR", 15, "https://images.unsplash.com/photo-1545454675-3531b543be5d?w=600", "Produk Unggulan", 4.8, "Sistem tata suara sinematik 120W untuk televisi ruang keluarga dengan konektivitas optik dan bluetooth."},
		{0, "HorizonTheater 3.1 Channel Dolby Spatial Audio Soundbar", "EN-AUD-005", 93.69, "USD", 15, "https://images.unsplash.com/photo-1545454675-3531b543be5d?w=600", "Premium", 4.8, "Slimline TV home audio soundbar system paired with high-output wireless long-throw subwoofer."},
		{1, "RimbaKompas Smartwatch Taktikal Militer Tangguh IP68", "ID-WCH-005", 799000.0, "IDR", 25, "https://images.unsplash.com/photo-1508685096489-7aacd43bd3b1?w=600", "Paling Dicari", 4.8, "Smartwatch tahan benturan dilengkapi altimeter barometrik, kompas digital, dan senter darurat."},
		{1, "TactixForce Military Shockproof Compass Smartwatch", "EN-WCH-005", 49.94, "USD", 25, "https://images.unsplash.com/photo-1508685096489-7aacd43bd3b1?w=600", "Trending", 4.8, "Combat-grade outdoor rugged smartwatch featuring barometric altimeter, storm alerts, and flashlight."},
		{2, "TegakPostur Dudukan Laptop Aluminium Lipat Pengatur Sudut", "ID-CMP-005", 219000.0, "IDR", 50, "https://images.unsplash.com/photo-1587829741301-dc798b83add3?w=600", "Terlaris", 4.9, "Stand laptop aluminium kokoh pembuang panas menjaga layar sejajar mata untuk mencegah sakit leher."},
		{2, "ErgoElevate Anodized Aluminum Folding Laptop Riser", "EN-CMP-005", 13.69, "USD", 50, "https://images.unsplash.com/photo-1587829741301-dc798b83add3?w=600", "Best Seller", 4.9, "Heavy-gauge ergonomic laptop riser promoting posture alignment and passive airflow cooling."},
		{3, "Kaos Oversize Streetwear Grafis Tipografi Putih Bersih", "ID-MCL-005", 129000.0, "IDR", 50, "https://images.unsplash.com/photo-1583743814966-8936f5b7be1a?w=600", "Populer", 4.7, "Kaos longgar bahu turun dengan sablon plastisol halus bertema perkotaan modern."},
		{3, "French Terry Drop-Shoulder Relaxed Streetwear Hoodie", "EN-MCL-005", 16.19, "USD", 30, "https://images.unsplash.com/photo-1556905055-8f358a7a47b2?w=600", "Trending", 4.9, "Cozy brushed cotton fleece pullover with double-layer drawstring hood and ribbed trim."},
		{4, "Celana Jogger Katun Baby Terry Pinggang Karet Abu Misty", "ID-MPN-005", 169000.0, "IDR", 45, "https://images.unsplash.com/photo-1542272604-780c96856592?w=600", "Favorit", 4.8, "Celana jogger santai berujung rib elastis dengan tali serut pinggang dan saku ritsleting dalam."},
		{4, "French Terry Cuffed Drawstring Jogger Sweatpants Heather", "EN-MPN-005", 10.56, "USD", 45, "https://images.unsplash.com/photo-1542272604-780c96856592?w=600", "Top Rated", 4.8, "Midweight cotton terry sweatpants with heavy-duty drawstring waistband and tapered ribbed cuffs."},
		{5, "Sandal Slop Busa EVA Empuk Pemulihan Kaki Hitam", "ID-MSH-005", 99000.0, "IDR", 80, "https://images.unsplash.com/photo-1549298916-b41d501d3772?w=600", "Terlaris", 4.7, "Sandal slide santai satu cetakan tanpa lem anti-air dan sangat ringan untuk santai di rumah."},
		{5, "Ergonomic Molded EVA Recovery Slide Sandals Onyx", "EN-MSH-005", 6.19, "USD", 80, "https://images.unsplash.com/photo-1549298916-b41d501d3772?w=600", "Best Seller", 4.7, "Ultra-cushioned single-piece EVA foam slide sandals providing restorative arch support after workouts."},
		{6, "Tas Pinggang Taktikal Selempang Kanvas Tahan Air", "ID-MAC-005", 169000.0, "IDR", 35, "https://images.unsplash.com/photo-1627123424574-724758594e93?w=600", "Paling Dicari", 4.8, "Waistbag kasual pria bahan kordura tahan percikan air dengan colokan kabel earphone dan banyak sekat."},
		{6, "Tactical Crossbody Sling Bag with Waterproof YKK Zips", "EN-MAC-005", 10.56, "USD", 35, "https://images.unsplash.com/photo-1627123424574-724758594e93?w=600", "Top Rated", 4.8, "Ballistic nylon EDC shoulder sling pack equipped with internal organizer dividers and headphone port."},
		{7, "Gamis Muslimah Katun Ceruti Motif Bunga Lembut Pastel", "ID-WCL-005", 299000.0, "IDR", 25, "https://images.unsplash.com/photo-1515886657613-9f3515b0c78f?w=600", "Pilihan Editor", 4.9, "Busana gamis syari wanita berlapis furing lembut dengan ritsleting depan busui friendly."},
		{7, "Tiered Bohemian Maxi Dress in Soft Cotton Voile Sage", "EN-WCL-005", 18.69, "USD", 25, "https://images.unsplash.com/photo-1515886657613-9f3515b0c78f?w=600", "Staff Pick", 4.9, "Floor-length boho prairie dress with smocked bodice, flutter sleeves, and sweeping tiered skirt."},
		{8, "Tas Jinjing Mini Selempang Satchel Warna Cokelat Tan", "ID-WBG-005", 239000.0, "IDR", 25, "https://images.unsplash.com/photo-1584917865442-de89df76afd3?w=600", "Pilihan Editor", 4.9, "Tas satchel formal wanita berstruktur kokoh dengan handle jinjing atas dan pengunci putar logam."},
		{8, "Soft Vegan Leather Slouchy Hobo Bag Chestnut Brown", "EN-WBG-005", 14.94, "USD", 25, "https://images.unsplash.com/photo-1584917865442-de89df76afd3?w=600", "Staff Pick", 4.9, "Supple crescent hobo bag designed with comfortable wide shoulder strap and spacious slouchy interior."},
		{9, "Sepatu Slip-On Rajut Wanita Bernapas Anti-Lecet Abu", "ID-WSH-005", 189000.0, "IDR", 45, "https://images.unsplash.com/photo-1543163521-1bf539c55dd2?w=600", "Terlaris", 4.8, "Sepatu jalan santai bahan rajutan rajut fleksibel dengan bantalan tumit busa anti-lecet."},
		{9, "Slip-On Stretch Knit Loafers with Cushioned Arch Support", "EN-WSH-005", 11.81, "USD", 45, "https://images.unsplash.com/photo-1543163521-1bf539c55dd2?w=600", "Best Seller", 4.8, "Breathable woven mesh slip-on loafers engineered to hug the foot with pillowy all-day support."},
		{10, "Teh Hitam Kayu Aro Artisan Daun Utuh Organik 100g", "ID-COF-005", 59000.0, "IDR", 60, "https://images.unsplash.com/photo-1576092768241-dec231879fc3?w=600", "Paling Dicari", 4.8, "Teh hitam ortodoks kualitas ekspor dari kebun teh tertua kaki Gunung Kerinci beraroma harum legendaris."},
		{10, "Imperial Earl Grey Loose Leaf Tea with Bergamot Oil 100g", "EN-COF-005", 3.69, "USD", 60, "https://images.unsplash.com/photo-1576092768241-dec231879fc3?w=600", "Best Seller", 4.8, "Classic orthodox black tea blend scented with natural cold-pressed Italian bergamot essential oil."},
		{11, "Keripik Pisang Sale Lumer Cokelat Batang Manis Legit", "ID-SNK-005", 35000.0, "IDR", 75, "https://images.unsplash.com/photo-1599599810769-bcde5a160d32?w=600", "Sedang Tren", 4.8, "Pisang sale panggang asap diselimuti cokelat leleh tebal dengan kerenyahan tahan lama."},
		{11, "Natural Creamy Peanut Butter with Sea Salt Only 16oz", "EN-SNK-005", 3.44, "USD", 60, "https://images.unsplash.com/photo-1599599810769-bcde5a160d32?w=600", "Best Seller", 4.9, "Velvety smooth peanut butter made exclusively with dry roasted runner peanuts and pinch of sea salt."},
		{12, "Toner Hidrasi Air Mawar Alami Menyeimbangkan pH Kulit", "ID-SKN-005", 79000.0, "IDR", 60, "https://images.unsplash.com/photo-1556228720-195a672e8a03?w=600", "Favorit", 4.7, "Toner penyegar destilasi kelopak mawar murni mengembalikan kesegaran dan kelembaban alami wajah."},
		{12, "Soothing Centella Asiatica Balancing Facial Toner 200ml", "EN-SKN-005", 4.94, "USD", 60, "https://images.unsplash.com/photo-1556228720-195a672e8a03?w=600", "Favorite", 4.7, "Cica-infused calming liquid toner soothing redness, balancing skin pH, and prepping for hydration."},
		{13, "Lulur Mandi Tradisional Kopi dan Beras Sangrai Halus 200g", "ID-BDY-005", 49000.0, "IDR", 80, "https://images.unsplash.com/photo-1592945403244-b3fbafd7f539?w=600", "Favorit", 4.9, "Scrub badan butiran kopi arabika asli dan tepung beras mengangkat sel kulit mati seketika."},
		{13, "Himalayan Pink Salt & Sweet Almond Oil Body Polish Scrub 200g", "EN-BDY-005", 3.06, "USD", 80, "https://images.unsplash.com/photo-1592945403244-b3fbafd7f539?w=600", "Staff Pick", 4.9, "Mineral-rich pink salt exfoliating body scrub buffing away dead skin cells to reveal satin softness."},
		{0, "SuaraJernih Mikrofon Kondensor USB Podcast & Vokal", "ID-AUD-006", 529000.0, "IDR", 30, "https://images.unsplash.com/photo-1590658006821-04f4008d5717?w=600", "Terlaris", 4.9, "Mikrofon rekaman pola kardioid dengan penyaring hembusan nafas dan pemantau suara langsung tanpa jeda."},
		{0, "ClearCast Pro Studio Broadcast USB Condenser Microphone", "EN-AUD-006", 33.06, "USD", 30, "https://images.unsplash.com/photo-1590658006821-04f4008d5717?w=600", "Best Seller", 4.9, "Broadcast-grade cardioid recording microphone with integrated pop filter and zero-latency monitoring."},
		{1, "WarnaWarni Smartband Kebugaran Pastel Ceria", "ID-WCH-006", 199000.0, "IDR", 45, "https://images.unsplash.com/photo-1575311373937-040b8e1fd5b6?w=600", "Hemat", 4.6, "Gelang pintar tali silikon warna pastel ceria dengan notifikasi WhatsApp dan kontrol musik ponsel."},
		{1, "ZenithBand Pastel Lightweight Everyday Activity Band", "EN-WCH-006", 12.44, "USD", 45, "https://images.unsplash.com/photo-1575311373937-040b8e1fd5b6?w=600", "Budget Pick", 4.6, "Casual colorful fitness band with phone notifications, camera shutter remote, and hydration alerts."},
		{2, "SerbaColok Hub USB-C 8-in-1 Port HDMI 4K & Pengisian Cepat", "ID-CMP-006", 299000.0, "IDR", 45, "https://images.unsplash.com/photo-1587829741301-dc798b83add3?w=600", "Wajib Punya", 4.8, "Dongle multifungsi port HDMI, pembaca kartu SD, USB 3.0, dan colokan charger laptop 100W PD."},
		{2, "OmniPort 8-in-1 USB-C Power Delivery Expansion Hub", "EN-CMP-006", 18.69, "USD", 45, "https://images.unsplash.com/photo-1587829741301-dc798b83add3?w=600", "Essential", 4.8, "Comprehensive USB-C dock hosting 4K HDMI, Gigabit Ethernet, SD reader, and 100W PD charging."},
		{3, "Polo Shirt Pria Katun Pique Kerah Rajut Biru Dongker", "ID-MCL-006", 149000.0, "IDR", 45, "https://images.unsplash.com/photo-1521572267360-ee0c2909d518?w=600", "Terlaris", 4.7, "Baju polo berkerah berpori sirkulasi udara baik dengan belahan samping rapi."},
		{3, "Breathable Pique Tailored Fit Polo Shirt Charcoal", "EN-MCL-006", 9.31, "USD", 45, "https://images.unsplash.com/photo-1521572267360-ee0c2909d518?w=600", "Popular", 4.7, "Honeycomb textured cotton pique polo with structured knit collar and two-button placket."},
		{4, "Celana Jeans Selvedge Tepi Merah Raw Denim Kaku 14oz", "ID-MPN-006", 489000.0, "IDR", 15, "https://images.unsplash.com/photo-1542272604-780c96856592?w=600", "Produk Unggulan", 4.9, "Denim kaku tepi merah ditenun mesin shuttle klasik untuk penggemar efek pudar alami fadding."},
		{4, "Authentic 14oz Japanese Red-Line Selvedge Raw Denim", "EN-MPN-006", 30.56, "USD", 15, "https://images.unsplash.com/photo-1542272604-780c96856592?w=600", "Premium", 4.9, "Unwashed shuttle-loom woven heritage denim jeans developing high-contrast personalized wear patterns."},
		{5, "Sepatu Bot Chelsea Kulit Suede Panel Karet Warna Tan", "ID-MSH-006", 479000.0, "IDR", 20, "https://images.unsplash.com/photo-1533867617858-e7b97e060509?w=600", "Sedang Tren", 4.8, "Sepatu bot chelsea suede elegan dengan panel elastis samping yang mudah dilepas pasang."},
		{5, "Suede Elastic-Gusset Ankle Chelsea Boots Desert Tan", "EN-MSH-006", 29.94, "USD", 20, "https://images.unsplash.com/photo-1533867617858-e7b97e060509?w=600", "Trending", 4.8, "Handcrafted water-treated suede Chelsea boots fitted with durable elastic side gores and pull loops."},
		{6, "Dompet Kartu Ramping Slot Depan Kulit Sintetis Moka", "ID-MAC-006", 59000.0, "IDR", 90, "https://images.unsplash.com/photo-1627123424574-724758594e93?w=600", "Hemat", 4.6, "Cardholder ramping tipis muat 6 kartu dan uang lipat sangat pas di saku celana depan."},
		{6, "Minimalist Aluminum Cardholder with Quick-Eject Trigger", "EN-MAC-006", 3.69, "USD", 90, "https://images.unsplash.com/photo-1627123424574-724758594e93?w=600", "Budget Pick", 4.6, "Ultra-thin RFID blocking anodized aluminum card case with bottom card pop-up slider."},
		{7, "Rok Plisket Panjang Rempel Halus Warna Khaki Anggun", "ID-WCL-006", 139000.0, "IDR", 55, "https://images.unsplash.com/photo-1515886657613-9f3515b0c78f?w=600", "Terlaris", 4.8, "Rok plisket lipit mikro bahan hyget premium jatuh elastis tidak mudah kusut."},
		{7, "Structured Double-Breasted Tailored Blazer Toffee", "EN-WCL-006", 20.56, "USD", 20, "https://images.unsplash.com/photo-1515886657613-9f3515b0c78f?w=600", "Editor's Choice", 4.9, "Modern suiting blazer crafted with peaked lapels, tortoise buttons, and full satin interior lining."},
		{8, "Tas Ransel Mini Kulit Wanita Slot Botol Samping Krem", "ID-WBG-006", 199000.0, "IDR", 35, "https://images.unsplash.com/photo-1584917865442-de89df76afd3?w=600", "Terlaris", 4.8, "Backpack mini wanita berbahan kulit sintetis anti-air dengan banyak kompartemen rahasia."},
		{8, "Geometric Acrylic Minaudiere Evening Clutch Bag Pearl", "EN-WBG-006", 17.44, "USD", 20, "https://images.unsplash.com/photo-1584917865442-de89df76afd3?w=600", "Premium", 4.9, "Hard-shell marbleized acrylic evening box clutch with optional drop-in gold snake chain."},
		{9, "Sepatu Hak Stiletto Lancip 7cm Kulit Mengkilap Hitam", "ID-WSH-006", 289000.0, "IDR", 20, "https://images.unsplash.com/photo-1543163521-1bf539c55dd2?w=600", "Pilihan Editor", 4.9, "Sepatu hak tinggi lancip 7cm berkesan mewah elegan dengan sol bantalan lateks empuk."},
		{9, "Classic Suede Almond-Toe Stiletto Pumps 3-Inch Navy", "EN-WSH-006", 18.06, "USD", 20, "https://images.unsplash.com/photo-1543163521-1bf539c55dd2?w=600", "Staff Pick", 4.9, "Timeless 3-inch stiletto high heels crafted in velvety faux-suede with latex cushioned insoles."},
		{10, "Teh Hijau Daun Puncak Ciwidey Wangi Alami 100g", "ID-COF-006", 52000.0, "IDR", 65, "https://images.unsplash.com/photo-1576092768241-dec231879fc3?w=600", "Terlaris", 4.7, "Pucuk daun teh hijau pegunungan Bandung Selatan tanpa pewarna buatan kaya zat antioksidan polifenol."},
		{10, "Japanese Ceremonial Grade Uji Matcha Green Tea Powder 50g", "EN-COF-006", 8.44, "USD", 20, "https://images.unsplash.com/photo-1576092768241-dec231879fc3?w=600", "Premium", 4.9, "First-harvest shade-grown green tea ground stone-milled in Kyoto boasting vivid green umami."},
		{11, "Keripik Singkong Balado Renyah Asli Daun Jeruk Purut", "ID-SNK-006", 25000.0, "IDR", 100, "https://images.unsplash.com/photo-1599599810769-bcde5a160d32?w=600", "Terlaris", 4.7, "Irisan singkong renyah dengan racikan cabai merah segar, gula tebu asli, dan daun jeruk wangi."},
		{11, "Raw Organic Chia Seeds High Omega-3 Fiber 16oz", "EN-SNK-006", 3.06, "USD", 70, "https://images.unsplash.com/photo-1599599810769-bcde5a160d32?w=600", "Essential", 4.8, "Nutrient-dense whole black chia seeds ideal for smoothies, protein puddings, and overnight oats."},
		{12, "Serum Retinol 0.5% Anti-Penuaan dan Kerutan Halus Malam", "ID-SKN-006", 169000.0, "IDR", 35, "https://images.unsplash.com/photo-1556228720-195a672e8a03?w=600", "Pilihan Editor", 4.8, "Serum regenerasi sel kulit malam hari untuk memudarkan garis halus dan memperbaiki tekstur kulit."},
		{12, "Encapsulated Retinol 1% Night Anti-Aging Treatment 30ml", "EN-SKN-006", 10.56, "USD", 35, "https://images.unsplash.com/photo-1556228720-195a672e8a03?w=600", "Staff Pick", 4.8, "Time-released micro-encapsulated retinoid serum smoothing deep wrinkles with minimal irritation."},
		{13, "Minyak Pijat Relaksasi Aromaterapi Lavender dan Kenanga 100ml", "ID-BDY-006", 65000.0, "IDR", 60, "https://images.unsplash.com/photo-1592945403244-b3fbafd7f539?w=600", "Pilihan Editor", 4.8, "Massage oil minyak kelapa murni dengan keharuman bunga kenanga menenangkan ketegangan otot."},
		{13, "Relaxing French Lavender Botanical Sleep Massage Oil 100ml", "EN-BDY-006", 4.06, "USD", 60, "https://images.unsplash.com/photo-1592945403244-b3fbafd7f539?w=600", "Editor's Choice", 4.8, "Calming therapeutic body oil blended with pure Provençal lavender and sweet almond base."},
		{0, "LariAktif Earphone Konduksi Tulang Telinga Terbuka", "ID-AUD-007", 699000.0, "IDR", 25, "https://images.unsplash.com/photo-1577174881658-0f30ed549adc?w=600", "Produk Baru", 4.7, "Earphone titanium tanpa menyumbat liang telinga agar tetap waspada saat bersepeda atau lari pagi."},
		{0, "AeroWave Titanium Frame Open-Ear Bone Conduction Headset", "EN-AUD-007", 43.69, "USD", 25, "https://images.unsplash.com/photo-1577174881658-0f30ed549adc?w=600", "New Arrival", 4.7, "Ergonomic bone conduction sports headphones keeping joggers aware of ambient traffic noise."},
		{1, "GolfNusantara Smartwatch Peta Lapangan Multi-Satelit", "ID-WCH-007", 1899000.0, "IDR", 10, "https://images.unsplash.com/photo-1523275335684-37898b6baf30?w=600", "Produk Unggulan", 4.9, "Jam pintar navigasi outdoor akurat dengan sensor penerima satelit ganda GPS, GLONASS, dan Galileo."},
		{1, "FairwayPro Precision Golf & Multi-Satellite Sports Watch", "EN-WCH-007", 118.69, "USD", 10, "https://images.unsplash.com/photo-1523275335684-37898b6baf30?w=600", "Premium", 4.9, "Specialized sports watch preloaded with global golf course layouts and pinpoint multi-band GNSS."},
		{2, "BantalKetik Sandaran Pergelangan Tangan Busa Memori Lembut", "ID-CMP-007", 89000.0, "IDR", 60, "https://images.unsplash.com/photo-1587829741301-dc798b83add3?w=600", "Hemat", 4.7, "Wrist rest empuk berbalut kulit sintetis halus mencegah nyeri sendi saat mengetik berjam-jam."},
		{2, "CloudRest Ergonomic Memory Foam Keyboard Wrist Support", "EN-CMP-007", 5.56, "USD", 60, "https://images.unsplash.com/photo-1587829741301-dc798b83add3?w=600", "Budget Pick", 4.7, "Contoured plush memory foam wrist rest covered in smooth cooling leatherette for typing comfort."},
		{3, "Sweater Rajut Halus Pria Kerah Bulat Warna Pasir", "ID-MCL-007", 219000.0, "IDR", 25, "https://images.unsplash.com/photo-1556905055-8f358a7a47b2?w=600", "Produk Baru", 4.8, "Knitwear rajutan lembut gaya kasual santai cocok dipakai di ruangan ber-AC."},
		{3, "Hawaiian Floral Vacation Camp Collar Shirt Botanical", "EN-MCL-007", 8.69, "USD", 40, "https://images.unsplash.com/photo-1596755094514-f87e34085b2c?w=600", "Trending", 4.7, "Flowy viscose summer shirt with vintage tropical print and casual open notch lapel."},
		{4, "Celana Pendek Chino Santai Katun Halus Biru Navy", "ID-MPN-007", 129000.0, "IDR", 60, "https://images.unsplash.com/photo-1624378439575-d8705ad7ae80?w=600", "Populer", 4.7, "Celana pendek kasual di atas lutut berbahan katun lembut cocok untuk bersantai di akhir pekan."},
		{4, "7-Inch Inseam Tailored Cotton Chino Summer Shorts Navy", "EN-MPN-007", 8.06, "USD", 60, "https://images.unsplash.com/photo-1624378439575-d8705ad7ae80?w=600", "Popular", 4.7, "Clean flat-front walking shorts hit right above the knee in garment-washed cotton."},
		{5, "Sneakers Basket Retro Sol Tebal Warna Putih Hijau Klasik", "ID-MSH-007", 349000.0, "IDR", 35, "https://images.unsplash.com/photo-1549298916-b41d501d3772?w=600", "Populer", 4.8, "Sneakers basket retro kulit sintetis tebal dengan bantalan pergelangan empuk gaya 90-an."},
		{5, "Heritage Court Basketball Leather Sneakers White/Green", "EN-MSH-007", 21.81, "USD", 35, "https://images.unsplash.com/photo-1549298916-b41d501d3772?w=600", "Popular", 4.8, "80s inspired court silhouette layered in full-grain synthetic leather with stitched cupsole."},
		{6, "Gelang Kulit Anyam Pria Gesper Magnet Baja Tahan Karat", "ID-MAC-007", 89000.0, "IDR", 60, "https://images.unsplash.com/photo-1627123424574-724758594e93?w=600", "Sedang Tren", 4.8, "Gelang tangan anyaman kulit sintetis hitam ganda dengan pengunci magnetik baja titanium."},
		{6, "Braided Leather Charm Bracelet with Magnetic Clasp", "EN-MAC-007", 5.56, "USD", 60, "https://images.unsplash.com/photo-1627123424574-724758594e93?w=600", "Trending", 4.8, "Double-wrap genuine braided leather wristband secured by a brushed titanium magnetic locking clasp."},
		{7, "Kemeja Wanita Katun Linen Oversize Warna Hijau Matcha", "ID-WCL-007", 189000.0, "IDR", 40, "https://images.unsplash.com/photo-1515886657613-9f3515b0c78f?w=600", "Sedang Tren", 4.8, "Kemeja santai wanita potongan longgar dengan saku dada tunggal cocok untuk hangout."},
		{7, "A-Line Pleated Swing Skirt with Hidden Pockets Olive", "EN-WCL-007", 8.69, "USD", 55, "https://images.unsplash.com/photo-1515886657613-9f3515b0c78f?w=600", "Best Seller", 4.8, "High-waist midi circle skirt made from stretch cotton twill with deep functional side pockets."},
		{8, "Clutch Pesta Mewah Aksen Manik Kristal Berkilau Perak", "ID-WBG-007", 279000.0, "IDR", 20, "https://images.unsplash.com/photo-1584917865442-de89df76afd3?w=600", "Produk Unggulan", 4.9, "Tas genggam pesta bertabur kristal swarovski imitasi dengan rantai selempang tipis."},
		{8, "Water-Resistant Nylon Multi-Pocket Travel Backpack Plum", "EN-WBG-007", 12.44, "USD", 35, "https://images.unsplash.com/photo-1584917865442-de89df76afd3?w=600", "Top Rated", 4.8, "Lightweight commuter backpack with anti-theft hidden back pocket and water-resistant coated zippers."},
		{9, "Sandal Tali Gladiator Tali Anyam Kasual Cokelat Muda", "ID-WSH-007", 149000.0, "IDR", 40, "https://images.unsplash.com/photo-1543163521-1bf539c55dd2?w=600", "Sedang Tren", 4.7, "Sandal gladiator tali ikat betis bahan kulit sintetis lembut cocok untuk liburan ke pantai."},
		{9, "Strappy Lace-Up Gladiator Summer Sandals Camel", "EN-WSH-007", 9.31, "USD", 40, "https://images.unsplash.com/photo-1543163521-1bf539c55dd2?w=600", "Trending", 4.7, "Boho wraparound gladiator lace sandals with gold aglets and flexible flat outsole."},
		{10, "Teh Melati Keraton Tradisional Batang dan Bunga Kering 150g", "ID-COF-007", 48000.0, "IDR", 80, "https://images.unsplash.com/photo-1576092768241-dec231879fc3?w=600", "Favorit", 4.8, "Teh wangi melati tubruk racikan klasik Jawa Tengah dengan aroma wangi semerbak menenangkan pikiran."},
		{10, "Dragon Well Longjing Hand-Fired Green Tea Leaves 75g", "EN-COF-007", 3.25, "USD", 65, "https://images.unsplash.com/photo-1576092768241-dec231879fc3?w=600", "Editor's Choice", 4.8, "Flat jade tea spears pan-roasted by hand yielding a smooth, savory toasted chestnut fragrance."},
		{11, "Keripik Nangka Organik Pengeringan Suhu Dingin Vakum 100g", "ID-SNK-007", 45000.0, "IDR", 55, "https://images.unsplash.com/photo-1599599810769-bcde5a160d32?w=600", "Populer", 4.8, "Keripik buah nangka manis asli diproses vacuum frying menjaga rasa manis alami buah tanpa gula."},
		{11, "Mountain Trekker Trail Mix with Cashews & Dark Chocolate", "EN-SNK-007", 5.31, "USD", 45, "https://images.unsplash.com/photo-1599599810769-bcde5a160d32?w=600", "Staff Pick", 4.8, "Energy-boosting trail blend featuring jumbo cashews, California raisins, and 70% cacao chunks."},
		{12, "Masker Lumpur Laut Mati Pembersih Pori dan Komedo 100g", "ID-SKN-007", 99000.0, "IDR", 50, "https://images.unsplash.com/photo-1556228720-195a672e8a03?w=600", "Sedang Tren", 4.8, "Clay mask lumpur mineral alami mengangkat kotoran pori-pori tersumbat dan mengontrol sebum wajah."},
		{12, "Clarifying Dead Sea Mineral Detox Mud Clay Mask 100g", "EN-SKN-007", 6.19, "USD", 50, "https://images.unsplash.com/photo-1556228720-195a672e8a03?w=600", "Trending", 4.8, "Kaolin and natural Dead Sea silt clay mask drawing out pore impurities and refining surface texture."},
		{13, "Deodoran Roll-On Alami Tawas dan Ekstrak Mentimun 50ml", "ID-BDY-007", 45000.0, "IDR", 90, "https://images.unsplash.com/photo-1592945403244-b3fbafd7f539?w=600", "Hemat", 4.7, "Deodoran non-alkohol berbahan mineral tawas alami mencegah bau ketiak 24 jam tanpa noda baju."},
		{13, "Aluminum-Free Natural Deodorant Stick with Bergamot 50g", "EN-BDY-007", 2.81, "USD", 90, "https://images.unsplash.com/photo-1592945403244-b3fbafd7f539?w=600", "Essential", 4.7, "Plant-based baking soda free underarm deodorant providing all-day odor neutralizing protection."},
		{0, "KayuKlasik Speaker Meja Vintage Kenop Putar Analog", "ID-AUD-008", 459000.0, "IDR", 20, "https://images.unsplash.com/photo-1520523839898-5071212a4e23?w=600", "Sedang Tren", 4.8, "Speaker estetik balutan kayu kenari alami dengan suara hangat dan tombol pengatur bass treble analog."},
		{0, "TimberAcoustics Handcrafted Walnut Bookshelf Bluetooth Speaker", "EN-AUD-008", 28.69, "USD", 20, "https://images.unsplash.com/photo-1520523839898-5071212a4e23?w=600", "Editor's Choice", 4.8, "Artisanal wooden desk speaker featuring dual silk-dome tweeters and brass analog control knobs."},
		{1, "TenangJiwa Gelang Pintar Sensor Stres & Latihan Nafas", "ID-WCH-008", 389000.0, "IDR", 35, "https://images.unsplash.com/photo-1508685096489-7aacd43bd3b1?w=600", "Sedang Tren", 4.7, "Smart tracker pemantau beban stres harian dan pengingat hidrasi air minum berkala."},
		{1, "MindfulTrack EDA Stress Bio-Feedback Sensor Wristband", "EN-WCH-008", 24.31, "USD", 35, "https://images.unsplash.com/photo-1508685096489-7aacd43bd3b1?w=600", "Editor's Choice", 4.7, "Smart lifestyle wristband analyzing electrodermal stress responses and delivering guided breathing routines."},
		{2, "StereoMeja Speaker Komputer Bilah Suara RGB Depan", "ID-CMP-008", 249000.0, "IDR", 40, "https://images.unsplash.com/photo-1545454675-3531b543be5d?w=600", "Sedang Tren", 4.7, "Speaker soundbar kompak bawah layar komputer dengan pencahayaan dinamis dan colokan headset."},
		{2, "SoundPod Desktop Stereo Soundbar with Chroma RGB Accents", "EN-CMP-008", 15.56, "USD", 40, "https://images.unsplash.com/photo-1545454675-3531b543be5d?w=600", "Trending", 4.7, "Under-monitor low-profile audio soundbar featuring touch-sensitive RGB ambient modes."},
		{3, "Jaket Bomber Parasut Tahan Angin Hijau Army", "ID-MCL-008", 279000.0, "IDR", 20, "https://images.unsplash.com/photo-1596755094514-f87e34085b2c?w=600", "Favorit", 4.8, "Jaket pilot bahan taslan anti-angin dengan saku ritsleting lengan dan furing oranye."},
		{3, "Merino Wool Blend V-Neck Knit Cardigan Heather Gray", "EN-MCL-008", 14.94, "USD", 25, "https://images.unsplash.com/photo-1556905055-8f358a7a47b2?w=600", "Editor's Choice", 4.8, "Fine-gauge lightweight cardigan offering natural thermal regulation and tortoise buttons."},
		{4, "Celana Bahan Formal Kantor Reguler Hitam Anti Kusut", "ID-MPN-008", 219000.0, "IDR", 40, "https://images.unsplash.com/photo-1624378439575-d8705ad7ae80?w=600", "Terlaris", 4.8, "Celana kain kantor potongan lurus dengan lapisan kantong furing halus tidak mudah kusut."},
		{4, "All-Day Performance Commuter Stretch Slacks Navy", "EN-MPN-008", 13.69, "USD", 40, "https://images.unsplash.com/photo-1624378439575-d8705ad7ae80?w=600", "Best Seller", 4.8, "Four-way stretch technical dress pants engineered with hidden zipper security passport pocket."},
		{5, "Sandal Tali Kulit Pria Sol Karet Kasual Cokelat Kopi", "ID-MSH-008", 189000.0, "IDR", 50, "https://images.unsplash.com/photo-1549298916-b41d501d3772?w=600", "Favorit", 4.7, "Sandal kasual tali kulit dengan bantalan telapak kaki berkontur anatomis mencegah pegal."},
		{5, "Dual-Strap Adjustable Footbed Cork Slide Sandals Brown", "EN-MSH-008", 11.81, "USD", 50, "https://images.unsplash.com/photo-1549298916-b41d501d3772?w=600", "Editor's Choice", 4.7, "Anatomical cork-latex footbed slide sandals with adjustable metal buckles and suede lining."},
		{6, "Topi Beanie Rajut Musim Dingin Warna Abu Misty Hangat", "ID-MAC-008", 65000.0, "IDR", 70, "https://images.unsplash.com/photo-1588850561407-ed78c282e89b?w=600", "Populer", 4.7, "Kupluk rajut elastis bahan wol sintetis lembut penahan dingin saat berkendara malam."},
		{6, "Chunky Ribbed Knit Beanie Hat Heather Charcoal", "EN-MAC-008", 4.06, "USD", 70, "https://images.unsplash.com/photo-1588850561407-ed78c282e89b?w=600", "Popular", 4.7, "Folded-cuff fisherman winter beanie knit from thermal acrylic yarn providing itch-free warmth."},
		{7, "Dress Kasual Floral Tali Bahu Silang Pantai Bali", "ID-WCL-008", 199000.0, "IDR", 30, "https://images.unsplash.com/photo-1515886657613-9f3515b0c78f?w=600", "Populer", 4.7, "Summer dress motif bunga tropis bahan rayon sejuk dengan aksen tali silang di punggung."},
		{7, "Ribbed Mock-Neck Long Sleeve Base-Layer Top Black", "EN-WCL-008", 6.19, "USD", 65, "https://images.unsplash.com/photo-1515886657613-9f3515b0c78f?w=600", "Essential", 4.7, "Form-fitting modal rib knit top providing sleek layering under dresses and blazers."},
		{8, "Tas Serut Ramping Bucket Bag Tali Panjang Maroon", "ID-WBG-008", 179000.0, "IDR", 35, "https://images.unsplash.com/photo-1584917865442-de89df76afd3?w=600", "Favorit", 4.7, "Tas serut model silinder dengan dasar bulat kokoh dan penutup tali serut berumbai rumbai."},
		{8, "Drawstring Leather Bucket Bag with Braided Strap Cognac", "EN-WBG-008", 11.19, "USD", 35, "https://images.unsplash.com/photo-1584917865442-de89df76afd3?w=600", "Trending", 4.7, "Structured round-bottom bucket bag with cinch cord closure and detachable woven shoulder strap."},
		{9, "Sepatu Loafers Wanita Aksen Gesper Logam Emas Maroon", "ID-WSH-008", 249000.0, "IDR", 30, "https://images.unsplash.com/photo-1543163521-1bf539c55dd2?w=600", "Paling Dicari", 4.8, "Sepatu loafers wanita formal dengan aksen ring gesper logam keemasan gaya preppy vintage."},
		{9, "Polished Penny Loafers with Metallic Bit Hardware Burgundy", "EN-WSH-008", 15.56, "USD", 30, "https://images.unsplash.com/photo-1543163521-1bf539c55dd2?w=600", "Editor's Choice", 4.8, "Preppy tailored leather loafers embellished with a polished equestrian gold-tone snaffle bar."},
		{10, "Biji Kopi Arabika Bali Kintamani Cita Rasa Jeruk Sitrus 250g", "ID-COF-008", 89000.0, "IDR", 50, "https://images.unsplash.com/photo-1514432324607-a09d9b4aefdd?w=600", "Populer", 4.8, "Kopi arabika proses basah dari Kintamani Bali dengan aftertaste jeruk segar dan bunga melati."},
		{10, "Pure Organic Egyptian Chamomile Flower Herbal Tea 50g", "EN-COF-008", 4.31, "USD", 45, "https://images.unsplash.com/photo-1576092768241-dec231879fc3?w=600", "Popular", 4.8, "Whole golden chamomile blossoms delivering caffeine-free apple-honey tranquility before bedtime."},
		{11, "Biji Chia Hitam Organik Kaya Serat dan Omega-3 200g", "ID-SNK-008", 49000.0, "IDR", 70, "https://images.unsplash.com/photo-1599599810769-bcde5a160d32?w=600", "Wajib Punya", 4.8, "Biji chia organik superfood untuk topping oatmeal, pudding susu chia, dan minuman detoks harian."},
		{11, "Gluten-Free Cinnamon Rolled Oats & Raisin Cookies", "EN-SNK-008", 3.0, "USD", 65, "https://images.unsplash.com/photo-1599599810769-bcde5a160d32?w=600", "Trending", 4.8, "Artisanal bakery cookies made with ancient grains, sweet sun-cured raisins, and Ceylon cinnamon."},
		{12, "Essence Niacinamide 5% Memudarkan Bekas Jerawat 100ml", "ID-SKN-008", 145000.0, "IDR", 40, "https://images.unsplash.com/photo-1556228720-195a672e8a03?w=600", "Terlaris", 4.8, "Cairan essence ringan mencerahkan warna kulit kusam dan meratakan tekstur bekas jerawat."},
		{12, "Niacinamide 10% + Zinc 1% Pore Refining Serum 30ml", "EN-SKN-008", 9.06, "USD", 40, "https://images.unsplash.com/photo-1556228720-195a672e8a03?w=600", "Best Seller", 4.8, "High-strength vitamin and mineral blemish formula regulating sebum production and minimizing pores."},
		{13, "Sabun Batang Minyak Kelapa Organik Buatan Tangan Bali 100g", "ID-BDY-008", 35000.0, "IDR", 100, "https://images.unsplash.com/photo-1592945403244-b3fbafd7f539?w=600", "Hemat", 4.8, "Handmade soap batang cold process alami wangi bunga kamboja lembut berbusa halus."},
		{13, "Handcrafted Cold-Process Coconut Milk Soap Bar 100g", "EN-BDY-008", 2.19, "USD", 100, "https://images.unsplash.com/photo-1592945403244-b3fbafd7f539?w=600", "Budget Pick", 4.8, "Gentle nourishing artisan soap bar infused with organic coconut cream and olive oil."},
		{0, "MutiaraPods Earbuds Ramping ANC 4-Mikrofon Panggilan Jernih", "ID-AUD-009", 489000.0, "IDR", 40, "https://images.unsplash.com/photo-1572536147248-ac59a8abfa4b?w=600", "Favorit", 4.8, "Earbuds peredam bising hibrida warna putih mutiara dengan mikrofon berteknologi beamforming."},
		{0, "CloudBuds Pro Ceramic White Hybrid ANC Earbuds", "EN-AUD-009", 30.56, "USD", 40, "https://images.unsplash.com/photo-1572536147248-ac59a8abfa4b?w=600", "Popular", 4.8, "Gloss white earbuds with adaptive environmental noise reduction and quad-mic call enhancement."},
		{1, "AnyamBaja Tali Jam Milanese Magnetik 22mm Anti Karat", "ID-WCH-009", 129000.0, "IDR", 70, "https://images.unsplash.com/photo-1546868871-7041f2a55e12?w=600", "Wajib Punya", 4.7, "Tali pengganti smartwatch bahan stainless steel rajut jaring halus dengan gesper magnet kuat."},
		{1, "MilaneseMesh Stainless Steel Magnetic Watch Band 22mm", "EN-WCH-009", 8.06, "USD", 70, "https://images.unsplash.com/photo-1546868871-7041f2a55e12?w=600", "Essential", 4.7, "Breathable woven steel mesh strap with infinitely adjustable strong magnetic lock."},
		{2, "BersihTotal Set Pembersih Keyboard & Layar Gadget 7-in-1", "ID-CMP-009", 49000.0, "IDR", 90, "https://images.unsplash.com/photo-1587829741301-dc798b83add3?w=600", "Hemat", 4.8, "Pembersih portabel lengkap dengan penarik keycaps, kuas debu mikro, dan semprotan pembersih layar."},
		{2, "TechCare 7-Piece Precision Electronics Cleaning Kit", "EN-CMP-009", 3.06, "USD", 90, "https://images.unsplash.com/photo-1587829741301-dc798b83add3?w=600", "Budget Pick", 4.8, "All-in-one gadget hygiene kit with keycap puller, fine nylon brushes, and optical lens solution."},
		{3, "Kaos Raglan Lengan 3/4 Katun Kombinasi Putih Hitam", "ID-MCL-009", 119000.0, "IDR", 55, "https://images.unsplash.com/photo-1521572267360-ee0c2909d518?w=600", "Hemat", 4.6, "Kaos lengan tanggung dua warna bahan katun adem cocok untuk kuliah dan santai."},
		{3, "Urban Utility Flight Bomber Jacket with Arm Pocket", "EN-MCL-009", 17.44, "USD", 20, "https://images.unsplash.com/photo-1596755094514-f87e34085b2c?w=600", "Best Seller", 4.9, "Satin nylon flight jacket with utility sleeve pencil pocket and ribbed knit varsity collar."},
		{4, "Celana Pendek Olahraga Lari 2-in-1 Saku Ritsleting HP", "ID-MPN-009", 99000.0, "IDR", 70, "https://images.unsplash.com/photo-1624378439575-d8705ad7ae80?w=600", "Hemat", 4.6, "Celana lari ringan dengan celana kompresi ketat di dalam dan saku anti-guncang ponsel."},
		{4, "Lightweight 2-in-1 Lined Running Shorts with Zipper Pocket", "EN-MPN-009", 6.19, "USD", 70, "https://images.unsplash.com/photo-1624378439575-d8705ad7ae80?w=600", "Budget Pick", 4.6, "Athletic split-hem running shorts with built-in anti-chafing compression liner and rear pocket."},
		{5, "Sepatu Santai Slip-On Rajut Fleksibel Tanpa Tali Abu Gelap", "ID-MSH-009", 219000.0, "IDR", 45, "https://images.unsplash.com/photo-1542291026-7eec264c27ff?w=600", "Hemat", 4.7, "Sepatu slip-on tanpa tali bahan rajut elastis yang sangat praktis langsung pakai untuk kuliah."},
		{5, "Hands-Free Breathable Stretch Knit Walking Slip-Ons", "EN-MSH-009", 13.69, "USD", 45, "https://images.unsplash.com/photo-1542291026-7eec264c27ff?w=600", "Budget Pick", 4.7, "Featherweight step-in athletic loafers featuring seamless sock knit upper and flexible sole."},
		{6, "Kacamata Baca Lensa Anti Radiasi Cahaya Biru Komputer", "ID-MAC-009", 99000.0, "IDR", 55, "https://images.unsplash.com/photo-1511499767150-a48a237f0083?w=600", "Wajib Punya", 4.8, "Kacamata frame polikarbonat ringan dengan lensa penyaring blue-light pelindung mata dari layar monitor."},
		{6, "Blue-Light Blocking Computer Glasses Tortoise Frame", "EN-MAC-009", 6.19, "USD", 55, "https://images.unsplash.com/photo-1511499767150-a48a237f0083?w=600", "Essential", 4.8, "Anti-eyestrain optical clarity glasses filtering 90% of harmful digital blue rays from monitors."},
		{7, "Outer Blazer Formal Wanita Berfuring Pundak Busa Navy", "ID-WCL-009", 329000.0, "IDR", 20, "https://images.unsplash.com/photo-1515886657613-9f3515b0c78f?w=600", "Produk Unggulan", 4.9, "Blazer kantor wanita bahan semi-wool berfuring sutra dengan kancing tunggal elegan."},
		{7, "Casual Chambray Denim Shirt Dress with Belt Blue", "EN-WCL-009", 12.44, "USD", 30, "https://images.unsplash.com/photo-1515886657613-9f3515b0c78f?w=600", "Popular", 4.7, "Lightweight cotton chambray midi dress equipped with matching fabric sash belt and chest pockets."},
		{8, "Dompet Panjang Wanita Kulit Aksen Kancing Jepret Emas", "ID-WBG-009", 129000.0, "IDR", 50, "https://images.unsplash.com/photo-1584917865442-de89df76afd3?w=600", "Populer", 4.8, "Dompet panjang elegan muat uang tanpa terlipat, 12 slot kartu debit, dan kantong ritsleting."},
		{8, "Slim Pebble-Grain Zip-Around Continental Wallet Taupe", "EN-WBG-009", 8.06, "USD", 50, "https://images.unsplash.com/photo-1584917865442-de89df76afd3?w=600", "Popular", 4.8, "Full zip-around long wallet featuring 12 credit card slots, central zip divider, and full currency sleeves."},
		{9, "Sandal Selop Mules Ujung Runcing Hak Pendek Lilac", "ID-WSH-009", 179000.0, "IDR", 35, "https://images.unsplash.com/photo-1543163521-1bf539c55dd2?w=600", "Produk Baru", 4.7, "Sepatu mules selop wanita berujung runcing dengan hak tahu pendek 3cm yang nyaman."},
		{9, "Pointed Mules in Crocodile-Embossed Vegan Leather Lilac", "EN-WSH-009", 11.19, "USD", 35, "https://images.unsplash.com/photo-1543163521-1bf539c55dd2?w=600", "New Arrival", 4.7, "Backless pointed slip-on mules featuring an embossed croc texture and comfortable 1.5-inch block heel."},
		{10, "Konsentrat Cold Brew Kopi Susu Aren Botol Kaca 500ml", "ID-COF-009", 75000.0, "IDR", 40, "https://images.unsplash.com/photo-1514432324607-a09d9b4aefdd?w=600", "Sedang Tren", 4.9, "Seduhan dingin kopi arabika murni 16 jam siap campur susu segar dan gula aren organik."},
		{10, "Single-Origin Cold Brew Coffee Steep Bags 6-Pack", "EN-COF-009", 4.69, "USD", 40, "https://images.unsplash.com/photo-1514432324607-a09d9b4aefdd?w=600", "Trending", 4.8, "Convenient pitcher brew pouches extracting smooth, ultra-low acidity concentrate in 12 hours."},
		{11, "Trail Mix Buah Kering Kismis Almond Kenari Sehat 250g", "ID-SNK-009", 85000.0, "IDR", 45, "https://images.unsplash.com/photo-1599599810769-bcde5a160d32?w=600", "Favorit", 4.8, "Campuran energi kacang almond panggang, kenari walnut, biji labu, dan buah kranberi kering."},
		{11, "Crispy Baked Sea Salt & Rosemary Pita Chips 8oz", "EN-SNK-009", 1.75, "USD", 90, "https://images.unsplash.com/photo-1599599810769-bcde5a160d32?w=600", "Popular", 4.7, "Twice-baked authentic hearth flatbread chips seasoned with aromatic rosemary and Mediterranean sea salt."},
		{12, "Pembersih Minyak Cleansing Oil Peluruh Make-up Waterproof", "ID-SKN-009", 119000.0, "IDR", 45, "https://images.unsplash.com/photo-1556228720-195a672e8a03?w=600", "Wajib Punya", 4.8, "Minyak pembersih wajah emulsifikasi air meluruhkan maskara tahan air dan sunblock seketika."},
		{12, "Nourishing Camellia Deep Cleansing Face Oil 150ml", "EN-SKN-009", 7.44, "USD", 45, "https://images.unsplash.com/photo-1556228720-195a672e8a03?w=600", "Essential", 4.8, "Silky plant-based oil cleanser melting long-wear foundation, waterproof eyeliner, and sunblock."},
		{13, "Body Mist Semprotan Tubuh Kesegaran Bunga Sakura 100ml", "ID-BDY-009", 59000.0, "IDR", 70, "https://images.unsplash.com/photo-1592945403244-b3fbafd7f539?w=600", "Populer", 4.7, "Semprotan wangi tubuh ringan manis floral untuk kesegaran setelah beraktivitas atau berolahraga."},
		{13, "Crisp Sea Salt & Driftwood Refreshing Body Mist 100ml", "EN-BDY-009", 3.69, "USD", 70, "https://images.unsplash.com/photo-1592945403244-b3fbafd7f539?w=600", "Trending", 4.7, "Airy coastal fragrance mist capturing refreshing ocean spray, white cedar, and mineral notes."},
		{0, "PestaRia Speaker Bluetooth Lampu LED RGB Mengikuti Ketukan", "ID-AUD-010", 799000.0, "IDR", 18, "https://images.unsplash.com/photo-1545454675-3531b543be5d?w=600", "Populer", 4.7, "Speaker pesta bertenaga besar dengan pertunjukan lampu warna-warni dan colokan mikrofon karaoke."},
		{0, "BassBlast MegaParty Wireless Tailgate Boombox with RGB Lights", "EN-AUD-010", 49.94, "USD", 18, "https://images.unsplash.com/photo-1545454675-3531b543be5d?w=600", "Hot Item", 4.7, "High-wattage portable PA party system with rhythm-reactive party lights and dual instrument inputs."},
		{1, "DokMagnet Pengisi Daya Cepat Nirkabel Smartwatch", "ID-WCH-010", 149000.0, "IDR", 50, "https://images.unsplash.com/photo-1523275335684-37898b6baf30?w=600", "Wajib Punya", 4.8, "Kabel charger magnetik universal dengan proteksi panas berlebih dan pengisian daya stabil."},
		{1, "SnapCharge Fast Magnetic Snap Smartwatch Charging Cable", "EN-WCH-010", 9.31, "USD", 50, "https://images.unsplash.com/photo-1523275335684-37898b6baf30?w=600", "Essential", 4.8, "Overheat-protected magnetic wireless charging cable compatible with leading smartwatch lines."},
		{2, "KabelRapi Kotak Penyimpan Colokan Listrik Anti-Debu Meja", "ID-CMP-010", 79000.0, "IDR", 60, "https://images.unsplash.com/photo-1587829741301-dc798b83add3?w=600", "Wajib Punya", 4.6, "Kotak pengorganisir stop kontak dan adaptor kabel kusut agar meja kerja rapi dan aman dari anak."},
		{2, "CableVault Flame-Retardant Wire Organizer Management Box", "EN-CMP-010", 4.94, "USD", 60, "https://images.unsplash.com/photo-1587829741301-dc798b83add3?w=600", "Essential", 4.6, "Matte finish power strip organizer concealing adapters and clutter under desks."},
		{3, "Kemeja Pantai Rayon Motif Daun Palem Tropis Bali", "ID-MCL-010", 139000.0, "IDR", 40, "https://images.unsplash.com/photo-1596755094514-f87e34085b2c?w=600", "Sedang Tren", 4.7, "Kemeja kerah terbuka bahan rayon jatuh yang sangat sejuk untuk jalan-jalan ke pantai."},
		{3, "Minimalist Typography Streetwear Graphic T-Shirt", "EN-MCL-010", 8.06, "USD", 50, "https://images.unsplash.com/photo-1583743814966-8936f5b7be1a?w=600", "Hot Item", 4.7, "Silkscreen typography printed loose-cut graphic tee made from pre-shrunk combed jersey."},
		{4, "Celana Panjang Katun Linen Bernapas Warna Pasir Pantai", "ID-MPN-010", 189000.0, "IDR", 35, "https://images.unsplash.com/photo-1624378439575-d8705ad7ae80?w=600", "Sedang Tren", 4.8, "Bahan campuran katun linen alami yang sangat sejuk dan berpori cocok untuk cuaca panas."},
		{4, "Breathable Drawstring Linen-Blend Beach Trousers Oatmeal", "EN-MPN-010", 11.81, "USD", 35, "https://images.unsplash.com/photo-1624378439575-d8705ad7ae80?w=600", "Trending", 4.8, "Airy natural linen blend trousers featuring an elasticized drawstring waist for tropical getaways."},
		{5, "Sepatu Loafers Kulit Asli Gesper Logam Elegan Hitam", "ID-MSH-010", 529000.0, "IDR", 18, "https://images.unsplash.com/photo-1533867617858-e7b97e060509?w=600", "Produk Unggulan", 4.9, "Sepatu loafers kulit asli lembut berkesan mewah dengan aksen gesper logam keemasan."},
		{5, "Handcrafted Full-Grain Leather Horsebit Loafers Mahogany", "EN-MSH-010", 33.06, "USD", 18, "https://images.unsplash.com/photo-1533867617858-e7b97e060509?w=600", "Premium", 4.9, "Supple burnished calfskin dress loafers detailed with a polished gold-tone horsebit bridle."},
		{6, "Sarung Tangan Kulit Motor Jahitan Kuat Nyaman Hitam", "ID-MAC-010", 129000.0, "IDR", 40, "https://images.unsplash.com/photo-1627123424574-724758594e93?w=600", "Favorit", 4.7, "Sarung tangan berkendara motor dengan ujung jari sentuh layar sentuh smartphone."},
		{6, "Breathable Touchscreen Leather Riding Gloves", "EN-MAC-010", 8.06, "USD", 40, "https://images.unsplash.com/photo-1627123424574-724758594e93?w=600", "Staff Pick", 4.7, "Perforated goatskin leather motorcycle gloves with conductive index finger pads for GPS screens."},
		{7, "Kaos Wanita Garis-Garis Katun Organik Pink Pastel", "ID-WCL-010", 119000.0, "IDR", 60, "https://images.unsplash.com/photo-1515886657613-9f3515b0c78f?w=600", "Hemat", 4.6, "Kaos santai motif garis pelaut bahan katun organik lembut adem dan tidak menyusut."},
		{7, "Organic Cotton Striped Breton Long Sleeve Tee Navy/White", "EN-WCL-010", 7.44, "USD", 60, "https://images.unsplash.com/photo-1515886657613-9f3515b0c78f?w=600", "Budget Pick", 4.6, "French-inspired nautical boatneck top crafted from 100% GOTS-certified combed cotton jersey."},
		{8, "Tas Belanja Lipat Parasut Ramah Lingkungan Motif Bunga", "ID-WBG-010", 45000.0, "IDR", 100, "https://images.unsplash.com/photo-1584917865442-de89df76afd3?w=600", "Hemat", 4.8, "Tas belanja serbaguna bahan parasut kuat yang bisa dilipat kecil menjadi gantungan kunci."},
		{8, "Handwoven Straw Market Basket Bag with Leather Handles", "EN-WBG-010", 14.31, "USD", 25, "https://images.unsplash.com/photo-1584917865442-de89df76afd3?w=600", "Editor's Choice", 4.9, "Artisan woven natural palm straw tote bag finished with durable flat leather shoulder handles."},
		{9, "Sneakers Platform Sol Tebal Kanvas Kasual Putih Polos", "ID-WSH-010", 219000.0, "IDR", 40, "https://images.unsplash.com/photo-1543163521-1bf539c55dd2?w=600", "Populer", 4.8, "Sepatu kanvas wanita sol platform tebal gaya santai retro mudah dipadukan dengan baju apa saja."},
		{9, "Casual Canvas Low-Top Sneakers with Gum Soles Off-White", "EN-WSH-010", 13.69, "USD", 40, "https://images.unsplash.com/photo-1543163521-1bf539c55dd2?w=600", "Popular", 4.8, "Everyday breathable canvas lace-up shoes built with contrast stitching and natural gum rubber outsoles."},
		{10, "Teh Bunga Telang Kering Organik Pewarna Alami 50g", "ID-COF-010", 39000.0, "IDR", 90, "https://images.unsplash.com/photo-1576092768241-dec231879fc3?w=600", "Hemat", 4.7, "Kuntum bunga telang biru kering murni kaya antioksidan antosianin untuk seduhan teh herbal biru ungu."},
		{10, "Masala Chai Spiced Black Tea with Cardamom & Ginger 100g", "EN-COF-010", 3.0, "USD", 80, "https://images.unsplash.com/photo-1576092768241-dec231879fc3?w=600", "Favorite", 4.8, "Robust Assam black tea hand-blended with crushed cinnamon sticks, green cardamom pods, and clove."},
		{11, "Selai Kacang Panggang Murni Tanpa Gula dan Minyak 250g", "ID-SNK-010", 55000.0, "IDR", 60, "https://images.unsplash.com/photo-1599599810769-bcde5a160d32?w=600", "Terlaris", 4.9, "100% selai kacang tanah panggang alami tekstur creamy tanpa pengawet atau minyak kelapa sawit."},
		{11, "Non-GMO Organic Coconut Flakes Lightly Toasted 8oz", "EN-SNK-010", 2.19, "USD", 75, "https://images.unsplash.com/photo-1599599810769-bcde5a160d32?w=600", "Budget Pick", 4.7, "Crisp unsweetened coconut smiles gently roasted for snacking or smoothie bowl garnishes."},
		{12, "Serum Asam Salisilat 2% Perawat Jerawat Meradang 30ml", "ID-SKN-010", 125000.0, "IDR", 45, "https://images.unsplash.com/photo-1556228720-195a672e8a03?w=600", "Populer", 4.7, "BHA cair eksfoliasi lembut membersihkan komedo hitam dan meredakan kemerahan jerawat aktif."},
		{12, "BHA Salicylic Acid 2% Exfoliating Liquid Solution 100ml", "EN-SKN-010", 7.81, "USD", 45, "https://images.unsplash.com/photo-1556228720-195a672e8a03?w=600", "Popular", 4.7, "Leave-on beta hydroxy acid exfoliant dissolving blackheads and unclogging deep pore debris."},
		{13, "Garam Rendam Kaki Epsom Salt Relaksasi Minyak Kayu Putih 250g", "ID-BDY-010", 49000.0, "IDR", 65, "https://images.unsplash.com/photo-1592945403244-b3fbafd7f539?w=600", "Wajib Punya", 4.8, "Garam magnesium murni pereda kaki pegal dan tumit pecah-pecah dengan sensasi hangat kayu putih."},
		{13, "Therapeutic Epsom Foot Soak Crystals with Tea Tree Oil 250g", "EN-BDY-010", 3.06, "USD", 65, "https://images.unsplash.com/photo-1592945403244-b3fbafd7f539?w=600", "Favorite", 4.8, "Sore muscle soaking salts infused with purifying Australian tea tree and peppermint oils."},
		{0, "PanggungPro Mikrofon Dinamis XLR Vokal Utama", "ID-AUD-011", 649000.0, "IDR", 22, "https://images.unsplash.com/photo-1590658006821-04f4008d5717?w=600", "Pilihan Editor", 4.9, "Mikrofon genggam panggung berbodi logam kokoh dengan peredam getaran internal anti-feedback."},
		{0, "StageCraft Live Dynamic Vocalist Microphone with Shock Mount", "EN-AUD-011", 40.56, "USD", 22, "https://images.unsplash.com/photo-1590658006821-04f4008d5717?w=600", "Staff Pick", 4.9, "Heavy-duty zinc die-cast handheld vocal microphone tuned for feedback rejection on live stages."},
		{1, "PelariCepat Smartwatch Marathon Bodi Polimer Ringan", "ID-WCH-011", 999000.0, "IDR", 20, "https://images.unsplash.com/photo-1508685096489-7aacd43bd3b1?w=600", "Terlaris", 4.8, "Smartwatch khusus pelari jarak jauh dengan indikator VO2 Max, irama langkah, dan waktu pemulihan."},
		{1, "StaminaPacer Lightweight Marathon Runner Smartwatch", "EN-WCH-011", 62.44, "USD", 20, "https://images.unsplash.com/photo-1508685096489-7aacd43bd3b1?w=600", "Best Seller", 4.8, "Optimized distance running watch tracking VO2 max estimates, running cadence, and recovery metrics."},
		{2, "BungeeKabel Penahan Kabel Mouse Silikon Lentur", "ID-CMP-011", 69000.0, "IDR", 55, "https://images.unsplash.com/photo-1615663245857-ac93bb7c39e7?w=600", "Populer", 4.7, "Lengan pegas fleksibel penahan kabel mouse kabel agar sensasi menggeser mouse terasa tanpa hambatan."},
		{2, "ZeroDrag Flexible Spring-Arm Mouse Bungee Controller", "EN-CMP-011", 4.31, "USD", 55, "https://images.unsplash.com/photo-1615663245857-ac93bb7c39e7?w=600", "Popular", 4.7, "Weighted silicone anchor keeping mouse cables elevated for snag-free competitive flick shots."},
		{3, "Kaos V-Neck Katun Stretch Hitam Pas Badan", "ID-MCL-011", 89000.0, "IDR", 70, "https://images.unsplash.com/photo-1583743814966-8936f5b7be1a?w=600", "Wajib Punya", 4.7, "Kaos kerah lancip elastis pas badan yang sangat cocok dijadikan dalaman kemeja atau blazer."},
		{3, "Preppy Ribbed Cable-Knit Crewneck Sweater Oatmeal", "EN-MCL-011", 13.69, "USD", 25, "https://images.unsplash.com/photo-1556905055-8f358a7a47b2?w=600", "New Arrival", 4.8, "Traditional textured cable-knit sweater delivering cozy warmth for autumn layering."},
		{4, "Celana Corduroy Vintage Garis Timbul Cokelat Karamel", "ID-MPN-011", 239000.0, "IDR", 25, "https://images.unsplash.com/photo-1624378439575-d8705ad7ae80?w=600", "Produk Baru", 4.8, "Celana korduroi tekstur garis timbul tebal dan hangat dengan potongan santai gaya 80-an."},
		{4, "Vintage Wide-Wale Corduroy Trousers in Caramel Tan", "EN-MPN-011", 14.94, "USD", 25, "https://images.unsplash.com/photo-1624378439575-d8705ad7ae80?w=600", "New Arrival", 4.8, "Plush 8-wale textured cotton corduroy trousers cut in an effortless relaxed straight leg."},
		{5, "Sepatu Olahraga Gym Sol Datar Latihan Beban Merah Marun", "ID-MSH-011", 369000.0, "IDR", 25, "https://images.unsplash.com/photo-1542291026-7eec264c27ff?w=600", "Produk Baru", 4.8, "Sepatu fitness sol karet datar tanpa sudut miring menjaga pijakan kaki kokoh saat angkat beban."},
		{5, "Zero-Drop Flat Sole Weightlifting Strength Shoes Crimson", "EN-MSH-011", 23.06, "USD", 25, "https://images.unsplash.com/photo-1542291026-7eec264c27ff?w=600", "New Arrival", 4.8, "Wide toe box deadlift trainers with zero heel elevation providing unmatched ground feedback."},
		{6, "Ikat Pinggang Kanvas Militer Gesper Jepit Otomatis Hijau", "ID-MAC-011", 69000.0, "IDR", 80, "https://images.unsplash.com/photo-1627123424574-724758594e93?w=600", "Hemat", 4.7, "Sabuk nilon kanvas anyam padat dengan gesper jepit logam bebas diatur tanpa lubang."},
		{6, "Military Webbing Tactical Belt with Quick-Release Buckle", "EN-MAC-011", 4.31, "USD", 80, "https://images.unsplash.com/photo-1627123424574-724758594e93?w=600", "Budget Pick", 4.7, "Heavy-duty 1000D nylon duty belt featuring a zinc alloy rapid-release cobra-style buckle."},
		{7, "Tunik Panjang Katun Toyobo Aksen Kerut Dada Cokelat", "ID-WCL-011", 219000.0, "IDR", 35, "https://images.unsplash.com/photo-1515886657613-9f3515b0c78f?w=600", "Favorit", 4.8, "Tunik santai berpotongan anggun hingga bawah lutut dengan detail rempel di dada."},
		{7, "Embroidered Eyelet Cotton Peplum Summer Blouse White", "EN-WCL-011", 11.81, "USD", 40, "https://images.unsplash.com/photo-1515886657613-9f3515b0c78f?w=600", "New Arrival", 4.8, "Feminine broderie anglaise blouse with scalloped hem and delicate puff sleeves."},
		{8, "Tas Selempang Moon Bag Bentuk Bulan Sabit Hitam Polos", "ID-WBG-011", 169000.0, "IDR", 40, "https://images.unsplash.com/photo-1584917865442-de89df76afd3?w=600", "Sedang Tren", 4.8, "Saddle bag bentuk setengah lingkaran yang sedang tren dengan tali bahu lebar yang nyaman."},
		{8, "Crescent Moon Nylon Crossbody Saddle Bag Khaki", "EN-WBG-011", 10.56, "USD", 40, "https://images.unsplash.com/photo-1584917865442-de89df76afd3?w=600", "Hot Item", 4.8, "Sleek half-moon ergonomic nylon sling bag engineered for hands-free city exploration."},
		{9, "Sandal Jepit Karet Jelly Anti-Air Nyaman Warna Peach", "ID-WSH-011", 69000.0, "IDR", 90, "https://images.unsplash.com/photo-1543163521-1bf539c55dd2?w=600", "Hemat", 4.6, "Sandal jepit bahan karet jelly lentur tahan basah mudah dicuci untuk santai harian."},
		{9, "Waterproof Cushioned EVA Pool Slide Sandals Peach", "EN-WSH-011", 4.31, "USD", 90, "https://images.unsplash.com/photo-1543163521-1bf539c55dd2?w=600", "Budget Pick", 4.6, "Ultra-lightweight molded waterproof pool slides designed for beach days and quick errands."},
		{10, "Teh Herbal Chamomile Bunga Kering Penenang Tidur 50g", "ID-COF-011", 69000.0, "IDR", 45, "https://images.unsplash.com/photo-1576092768241-dec231879fc3?w=600", "Pilihan Editor", 4.8, "Bunga kamomil utuh tanpa kafein yang memberikan efek relaksasi mendalam sebelum tidur malam."},
		{10, "Jasmine Pearls Hand-Rolled Green Tea Leaves Tin 75g", "EN-COF-011", 5.56, "USD", 50, "https://images.unsplash.com/photo-1576092768241-dec231879fc3?w=600", "Top Rated", 4.8, "Tightly hand-rolled green tea pearls scented over fresh night-blooming jasmine flowers."},
		{11, "Keripik Ubi Ungu Renyah Tanpa Pengawet Buatan 150g", "ID-SNK-011", 29000.0, "IDR", 80, "https://images.unsplash.com/photo-1599599810769-bcde5a160d32?w=600", "Hemat", 4.7, "Camilan ubi ungu lokal gurih manis diproses higienis kaya serat pangan dan antioksidan antosianin."},
		{11, "Roasted Salted Pumpkin Seeds Pepitas 12oz", "EN-SNK-011", 2.62, "USD", 60, "https://images.unsplash.com/photo-1599599810769-bcde5a160d32?w=600", "Favorite", 4.8, "Shelled green pumpkin seeds dry-roasted and lightly sprinkled with mineral-rich Himalayan rock salt."},
		{12, "Krim Mata Peptida Peredam Kantung Mata dan Mata Panda", "ID-SKN-011", 135000.0, "IDR", 40, "https://images.unsplash.com/photo-1556228720-195a672e8a03?w=600", "Favorit", 4.8, "Eye cream dengan aplikator ujung keramik dingin merelaksasi mata lelah dan memudarkan lingkaran hitam."},
		{12, "Triple Peptide Firming Eye Cream with Caffeine 15ml", "EN-SKN-011", 8.44, "USD", 40, "https://images.unsplash.com/photo-1556228720-195a672e8a03?w=600", "Staff Pick", 4.8, "De-puffing cooling eye treatment enriched with signal peptides and green tea caffeine."},
		{13, "Krim Tangan Pelembab Shea Butter dan Minyak Zaitun 50ml", "ID-BDY-011", 49000.0, "IDR", 75, "https://images.unsplash.com/photo-1592945403244-b3fbafd7f539?w=600", "Favorit", 4.7, "Hand cream tidak lengket melembabkan telapak tangan kering akibat sering cuci tangan."},
		{13, "Intensive Hand & Cuticle Recovery Cream with Vitamin E 50ml", "EN-BDY-011", 3.06, "USD", 75, "https://images.unsplash.com/photo-1592945403244-b3fbafd7f539?w=600", "Essential", 4.7, "Non-greasy rapid repair hand balm soothing cracked knuckles and restoring dry nail cuticles."},
		{0, "RinganSuara Earbuds Nirkabel Ultra-Ringan 3.5 Gram", "ID-AUD-012", 189000.0, "IDR", 60, "https://images.unsplash.com/photo-1590658268037-6bf12165a8df?w=600", "Hemat", 4.6, "Earbuds silikon lembut yang sangat ringan dan nyaman dipakai seharian saat belajar daring atau bekerja."},
		{0, "FeatherTouch Minimalist Ergonomic Wireless In-Ear Earphones", "EN-AUD-012", 11.81, "USD", 60, "https://images.unsplash.com/photo-1590658268037-6bf12165a8df?w=600", "Budget Pick", 4.6, "Ultra-lightweight 3.5g contoured earbuds providing pressure-free comfort for commute and study."},
		{1, "DewiAnggun Smartwatch Elegan Berlian Kilau Rose Gold", "ID-WCH-012", 679000.0, "IDR", 30, "https://images.unsplash.com/photo-1523275335684-37898b6baf30?w=600", "Sedang Tren", 4.8, "Smartwatch wanita ramping warna rose gold dengan pelacak siklus bulanan dan tema wallpaper mewah."},
		{1, "EleganceGlow Slim Rose Gold Crystal Smartwatch for Women", "EN-WCH-012", 42.44, "USD", 30, "https://images.unsplash.com/photo-1523275335684-37898b6baf30?w=600", "Top Rated", 4.8, "Jewelry-inspired slim smartwatch featuring women's health cycle sync and customizable analog watchfaces."},
		{2, "HembusDingin Kipas Pendingin Laptop 5 Kipas Sunyi LED Biru", "ID-CMP-012", 189000.0, "IDR", 35, "https://images.unsplash.com/photo-1587829741301-dc798b83add3?w=600", "Sedang Tren", 4.7, "Cooling pad bertenaga hembusan angin sejuk menjaga laptop kerja atau gaming tidak cepat panas."},
		{2, "FrostGuard 5-Fan Whisper-Quiet Notebook Cooling Pad", "EN-CMP-012", 11.81, "USD", 35, "https://images.unsplash.com/photo-1587829741301-dc798b83add3?w=600", "Trending", 4.7, "High-velocity mesh laptop cooler with adjustable angle incline and auxiliary USB pass-through."},
		{3, "Jaket Parka Musim Hujan Lapisan Parasut Biru Navy", "ID-MCL-012", 329000.0, "IDR", 20, "https://images.unsplash.com/photo-1556905055-8f358a7a47b2?w=600", "Pilihan Editor", 4.8, "Jaket parka pelindung gerimis dengan penutup kepala bertali serut dan kancing jepret kuningan."},
		{3, "Micro-Stripe Yarn-Dyed Casual Pocket Tee Navy/White", "EN-MCL-012", 7.81, "USD", 45, "https://images.unsplash.com/photo-1521572267360-ee0c2909d518?w=600", "Popular", 4.7, "Horizontal yarn-dyed stripe t-shirt with reinforced chest pocket and split side hem."},
		{4, "Celana Jeans Biru Terang Aksen Robek Halus Lutut", "ID-MPN-012", 269000.0, "IDR", 25, "https://images.unsplash.com/photo-1542272604-780c96856592?w=600", "Paling Dicari", 4.7, "Jeans cuci warna biru muda cerah dengan detail sobek mikro di bagian tempurung lutut."},
		{4, "Relaxed Wide-Leg 90s Skater Jeans in Bleach Light Wash", "EN-MPN-012", 16.81, "USD", 25, "https://images.unsplash.com/photo-1542272604-780c96856592?w=600", "Hot Item", 4.7, "Baggy skater fit denim jeans finished with subtle vintage abrasions along the knee seams."},
		{5, "Sandal Gunung Tali Webbing Nilon Gesper Cepat Hitam", "ID-MSH-012", 159000.0, "IDR", 60, "https://images.unsplash.com/photo-1549298916-b41d501d3772?w=600", "Wajib Punya", 4.8, "Sandal gunung tali nilon tebal dengan pengunci gesper klik praktis dan sol karet tebal anti-licin."},
		{5, "Heavy-Duty Webbing Harness River Trekking Sandals Black", "EN-MSH-012", 9.94, "USD", 60, "https://images.unsplash.com/photo-1549298916-b41d501d3772?w=600", "Essential", 4.8, "Hydrophobic webbing adventure sandals featuring quick-cam buckles and river grip tread."},
		{6, "Dompet Panjang Pria Kulit Tempat Buku Tabungan & HP", "ID-MAC-012", 189000.0, "IDR", 30, "https://images.unsplash.com/photo-1627123424574-724758594e93?w=600", "Pilihan Editor", 4.8, "Long wallet pria berkapasitas besar dengan slot ritsleting koin dan penahan handphone."},
		{6, "Travel Passport Holder & Document Organizer Leather", "EN-MAC-012", 11.81, "USD", 30, "https://images.unsplash.com/photo-1627123424574-724758594e93?w=600", "Editor's Choice", 4.8, "Zip-around travel wallet housing passport pockets, boarding pass sleeves, and pen holder."},
		{7, "Baju Tidur Piyama Sutra Satin Lembut Kerah Piyama Mutiara", "ID-WCL-012", 179000.0, "IDR", 40, "https://images.unsplash.com/photo-1515886657613-9f3515b0c78f?w=600", "Populer", 4.7, "Setelan piyama lengan pendek dan celana panjang bahan satin sutra halus sejuk saat tidur."},
		{7, "Mulberry Silk Satin Cami and Pajama Lounge Set Blush", "EN-WCL-012", 11.19, "USD", 40, "https://images.unsplash.com/photo-1515886657613-9f3515b0c78f?w=600", "Trending", 4.7, "Silky breathable sleepwear set featuring adjustable spaghetti straps and drawstring shorts."},
		{8, "Pouch Kosmetik Make-up Bahan Beludru Anti-Air Pink Pastel", "ID-WBG-012", 69000.0, "IDR", 80, "https://images.unsplash.com/photo-1584917865442-de89df76afd3?w=600", "Wajib Punya", 4.7, "Tas kosmetik lembut berfuring satin dengan bukaan lebar untuk menampung lipstik dan bedak."},
		{8, "Quilted Velvet Cosmetic Pouch with Brass Hardware Rose", "EN-WBG-012", 4.31, "USD", 80, "https://images.unsplash.com/photo-1584917865442-de89df76afd3?w=600", "Essential", 4.7, "Plush water-repellent lined makeup organizer pouch with interior elastic brush pockets."},
		{9, "Sepatu Bot Semata Kaki Suede Tali Depan Cokelat Pasir", "ID-WSH-012", 349000.0, "IDR", 20, "https://images.unsplash.com/photo-1543163521-1bf539c55dd2?w=600", "Produk Unggulan", 4.9, "Ankle boots wanita bahan suede lembut bertali depan dengan hak balok kayu kokoh."},
		{9, "Suede Side-Zip Ankle Booties with Stacked Heel Sand", "EN-WSH-012", 21.81, "USD", 20, "https://images.unsplash.com/photo-1543163521-1bf539c55dd2?w=600", "Premium", 4.9, "Western-inspired ankle booties made from soft microfiber suede with a sturdy 2.5-inch stacked wooden heel."},
		{10, "Biji Kopi Arabika Papua Wamena Alami Dataran Tinggi 250g", "ID-COF-012", 110000.0, "IDR", 30, "https://images.unsplash.com/photo-1514432324607-a09d9b4aefdd?w=600", "Produk Unggulan", 4.9, "Kopi organik langka dari Lembah Baliem Papua yang ditanam tanpa pupuk kimia pada ketinggian 1800 mdpl."},
		{10, "Swiss Water Process Decaf Whole Bean Roast 12oz", "EN-COF-012", 5.5, "USD", 45, "https://images.unsplash.com/photo-1514432324607-a09d9b4aefdd?w=600", "Staff Pick", 4.8, "100% chemical-free decaffeinated coffee maintaining rich dark chocolate flavor integrity."},
		{11, "Abon Ikan Cakalang Asli Manado Gurih Tabur Nasi 150g", "ID-SNK-012", 59000.0, "IDR", 50, "https://images.unsplash.com/photo-1599599810769-bcde5a160d32?w=600", "Populer", 4.8, "Abon suwir daging ikan cakalang asap bumbu rempah tradisional gurih pedas siap santap."},
		{11, "Organic Deglet Noor Pitted Sun-Dried Dates 14oz", "EN-SNK-012", 3.69, "USD", 50, "https://images.unsplash.com/photo-1599599810769-bcde5a160d32?w=600", "Editor's Choice", 4.8, "Caramel-flavored soft pitted dates harvested from California desert oases, zero preservatives."},
		{12, "Masker Lembaran Sheet Mask Kolagen Mencerahkan Wajah Isi 5", "ID-SKN-012", 65000.0, "IDR", 90, "https://images.unsplash.com/photo-1556228720-195a672e8a03?w=600", "Hemat", 4.8, "Paket isi 5 sheet mask serat tencel tipis mengandung serum kolagen laut dan ekstrak mutiara."},
		{12, "Hydrogel Bio-Cellulose Collagen Infused Face Masks 5-Pack", "EN-SKN-012", 4.06, "USD", 90, "https://images.unsplash.com/photo-1556228720-195a672e8a03?w=600", "Budget Pick", 4.8, "Second-skin adherence facial sheet masks drenched in marine collagen and multi-molecular hyaluronic acid."},
		{13, "Mentega Tubuh Body Butter Cokelat Lemak Nabati Kental 200g", "ID-BDY-012", 89000.0, "IDR", 45, "https://images.unsplash.com/photo-1592945403244-b3fbafd7f539?w=600", "Sedang Tren", 4.8, "Body butter pelembab ekstra pekat aroma cokelat lezat untuk kulit bersisik dan siku kering."},
		{13, "Whipped Coconut Cocoa Butter Deep Moisture Souffle 200g", "EN-BDY-012", 5.56, "USD", 45, "https://images.unsplash.com/photo-1592945403244-b3fbafd7f539?w=600", "Popular", 4.8, "Decadent whipped body butter melting on contact to replenish rough elbows and knees."},
		{0, "HiFiMaster Konverter Audio DAC USB-C 32-Bit Lossless", "ID-AUD-013", 219000.0, "IDR", 40, "https://images.unsplash.com/photo-1546435770-a3e426bf472b?w=600", "Wajib Punya", 4.8, "Dongle DAC audio beresolusi tinggi 384kHz untuk mendengarkan file musik kualitas studio di ponsel."},
		{0, "SonicPurity 32-Bit USB Type-C High-Resolution Audio Adapter", "EN-AUD-013", 13.69, "USD", 40, "https://images.unsplash.com/photo-1546435770-a3e426bf472b?w=600", "Essential", 4.8, "Ultra-clean portable DAC converter unlocking high-res lossless streaming for modern smartphones."},
		{1, "AnakAman Smartwatch GPS Pelacak Lokasi 4G & Panggilan Video", "ID-WCH-013", 399000.0, "IDR", 40, "https://images.unsplash.com/photo-1575311373937-040b8e1fd5b6?w=600", "Paling Dicari", 4.7, "Jam pintar anak dengan tombol panggilan darurat SOS, kamera foto jarak jauh, dan zona batas aman."},
		{1, "GuardianKid 4G GPS Geofencing Children Smartwatch", "EN-WCH-013", 24.94, "USD", 40, "https://images.unsplash.com/photo-1575311373937-040b8e1fd5b6?w=600", "Popular", 4.7, "Safety smartwatch for children with instant SOS calling, two-way HD video, and safe-zone notifications."},
		{2, "PuddingGlow Tutup Tombol Keyboard PBT Tembus Cahaya", "ID-CMP-013", 149000.0, "IDR", 45, "https://images.unsplash.com/photo-1587829741301-dc798b83add3?w=600", "Produk Baru", 4.8, "Keycaps bahan PBT tebal dua lapis mempercantik pendaran cahaya lampu latar RGB keyboard mekanikal."},
		{2, "AuraCaps Double-Shot PBT Translucent Pudding Keycaps", "EN-CMP-013", 9.31, "USD", 45, "https://images.unsplash.com/photo-1587829741301-dc798b83add3?w=600", "New Arrival", 4.8, "Textured oil-resistant PBT keycaps with frosted sidewalls amplifying mechanical RGB brilliance."},
		{3, "Kemeja Kantor Formal Katun Dobby Putih Anti Kusut", "ID-MCL-013", 209000.0, "IDR", 40, "https://images.unsplash.com/photo-1596755094514-f87e34085b2c?w=600", "Terlaris", 4.9, "Kemeja kerja pria bermotif anyam dobby mikro berkerah kaku rapi siap pakai ke kantor."},
		{3, "Technical Weather-Shield Hooded Parka Jacket Olive", "EN-MCL-013", 20.56, "USD", 20, "https://images.unsplash.com/photo-1556905055-8f358a7a47b2?w=600", "Staff Pick", 4.8, "Water-resistant windproof storm parka equipped with deep storm flap cargo pockets."},
		{4, "Celana Pendek Kargo Drill Tebal Hijau Tentara", "ID-MPN-013", 149000.0, "IDR", 50, "https://images.unsplash.com/photo-1542272604-780c96856592?w=600", "Populer", 4.7, "Celana pendek kargo kokoh berkancing besi dengan kantong samping penutup velcro."},
		{4, "Heavy Cotton Drill Utility Workwear Cargo Shorts Olive", "EN-MPN-013", 9.31, "USD", 50, "https://images.unsplash.com/photo-1542272604-780c96856592?w=600", "Popular", 4.7, "Rugged utility shorts featuring reinforced seat stitching and deep velcro-secured cargo flaps."},
		{5, "Sepatu Skate Kanvas Sol Karet Waffle Cengkeram Kuat", "ID-MSH-013", 319000.0, "IDR", 30, "https://images.unsplash.com/photo-1549298916-b41d501d3772?w=600", "Sedang Tren", 4.7, "Sepatu skate kanvas bertumpuk ganda dengan sol waffle karet mentah mencengkeram papan erat."},
		{5, "Suede Chukka Ankle Boots with Crepe Rubber Sole Sand", "EN-MSH-013", 24.31, "USD", 25, "https://images.unsplash.com/photo-1533867617858-e7b97e060509?w=600", "Staff Pick", 4.8, "Classic desert chukkas made from velvety sand suede paired with soft natural crepe rubber sole."},
		{6, "Klip Penjepit Uang Logam Stainless Steel Ramping Perak", "ID-MAC-013", 45000.0, "IDR", 100, "https://images.unsplash.com/photo-1627123424574-724758594e93?w=600", "Hemat", 4.7, "Money clip logam pegas fleksibel penahan tumpukan uang kertas agar saku tetap tipis."},
		{6, "Stainless Steel Spring-Loaded Slim Money Clip Silver", "EN-MAC-013", 2.81, "USD", 100, "https://images.unsplash.com/photo-1627123424574-724758594e93?w=600", "Essential", 4.7, "Brushed stainless steel tension money clip holding up to 30 folded bills flat in pockets."},
		{7, "Celana High-Waist Loose Trousers Bahan Scuba Hitam", "ID-WCL-013", 189000.0, "IDR", 45, "https://images.unsplash.com/photo-1515886657613-9f3515b0c78f?w=600", "Paling Dicari", 4.8, "Celana bahan scuba jatuh tebal berpinggang tinggi membuat kaki tampak lebih jenjang."},
		{7, "Raw-Hem Straight-Leg High-Rise Cropped Jeans Light Indigo", "EN-WCL-013", 14.31, "USD", 30, "https://images.unsplash.com/photo-1515886657613-9f3515b0c78f?w=600", "Popular", 4.8, "Flattering vintage fit rigid denim jeans cut with a classic high waist and cropped raw hemline."},
		{8, "Tas Anyaman Rotan Alami Bulat Khas Pengrajin Bali", "ID-WBG-013", 229000.0, "IDR", 25, "https://images.unsplash.com/photo-1584917865442-de89df76afd3?w=600", "Paling Dicari", 4.9, "Tas rotan bundar anyaman tangan asli dengan tali kulit sapi asli dan kain batik dalam."},
		{8, "Structured Top-Handle Satchel with Crossbody Strap Emerald", "EN-WBG-013", 18.06, "USD", 20, "https://images.unsplash.com/photo-1584917865442-de89df76afd3?w=600", "Premium", 4.9, "Executive architectural structured satchel with protective metal bottom feet and divided interior."},
		{9, "Flat Shoes Ujung Kotak Aksen Pita Manis Warna Sage", "ID-WSH-013", 169000.0, "IDR", 45, "https://images.unsplash.com/photo-1543163521-1bf539c55dd2?w=600", "Favorit", 4.8, "Sepatu flat model square toe modern dengan hiasan pita mungil di ujung depan."},
		{9, "Square-Toe Mary Jane Pumps with Dual Instep Straps Black", "EN-WSH-013", 16.19, "USD", 30, "https://images.unsplash.com/photo-1543163521-1bf539c55dd2?w=600", "Staff Pick", 4.9, "Vintage patent leather Mary Jane shoes featuring double slender buckle straps and modern square toe."},
		{10, "Kopi Celup Praktis Kantong Saring Filter Drip 5 Sachet", "ID-COF-013", 45000.0, "IDR", 85, "https://images.unsplash.com/photo-1514432324607-a09d9b4aefdd?w=600", "Wajib Punya", 4.7, "Kopi giling arabika single-serve dalam kantong saring gantung mudah diseduh air panas saat bepergian."},
		{10, "Pour-Over Biodegradable Coffee Drip Filters Box 50ct", "EN-COF-013", 2.81, "USD", 85, "https://images.unsplash.com/photo-1514432324607-a09d9b4aefdd?w=600", "Essential", 4.7, "Unbleached natural wood pulp conical filters designed for clean cup extraction."},
		{11, "Rempeyek Kacang Tanah Renyah Daun Jeruk Purut 200g", "ID-SNK-013", 32000.0, "IDR", 75, "https://images.unsplash.com/photo-1599599810769-bcde5a160d32?w=600", "Favorit", 4.7, "Peyek kacang renyah tipis dengan santan kelapa asli dan potongan daun jeruk purut beraroma wangi."},
		{11, "Artisanal 70% Dark Chocolate Bar with Sea Salt Crystals", "EN-SNK-013", 2.38, "USD", 70, "https://images.unsplash.com/photo-1599599810769-bcde5a160d32?w=600", "Top Rated", 4.9, "Fair-trade certified bean-to-bar dark chocolate infused with hand-harvested flaky French sea salt."},
		{12, "Salep Bibir Lip Sleeping Mask Ekstrak Buah Beri Lembut", "ID-SKN-013", 59000.0, "IDR", 80, "https://images.unsplash.com/photo-1556228720-195a672e8a03?w=600", "Wajib Punya", 4.7, "Masker bibir malam hari melembutkan bibir kering pecah-pecah dengan ekstrak beri segar."},
		{12, "Hydrating Overnight Berry Lip Butter Treatment 20g", "EN-SKN-013", 3.69, "USD", 80, "https://images.unsplash.com/photo-1556228720-195a672e8a03?w=600", "Essential", 4.7, "Intensive berry wax and shea butter bedtime balm healing dry, flaky lips by morning."},
		{13, "Scrub Pembersih Kulit Kepala Ekstrak Daun Mint Segar 150g", "ID-BDY-013", 79000.0, "IDR", 50, "https://images.unsplash.com/photo-1592945403244-b3fbafd7f539?w=600", "Produk Baru", 4.8, "Scalp scrub butiran garam laut membersihkan ketombe dan residu minyak rambut di kulit kepala."},
		{13, "Purifying Himalayan Salt Exfoliating Scalp Scrub 150g", "EN-BDY-013", 4.94, "USD", 50, "https://images.unsplash.com/photo-1592945403244-b3fbafd7f539?w=600", "New Arrival", 4.8, "Pre-shampoo clarifying scrub removing product buildup and excess sebum from hair roots."},
		{0, "ArenaGame Headset Gaming Suara Sekitar 7.1 dengan Mic Busa", "ID-AUD-014", 379000.0, "IDR", 35, "https://images.unsplash.com/photo-1505740420928-5e560c06d30e?w=600", "Sedang Tren", 4.7, "Headset gaming dengan bantalan telinga busa memori sejuk dan mikrofon fleksibel peredam bising."},
		{0, "CyberStrike 7.1 Virtual Surround Sound Tactical Gaming Headset", "EN-AUD-014", 23.69, "USD", 35, "https://images.unsplash.com/photo-1505740420928-5e560c06d30e?w=600", "Trending", 4.7, "Immersive multi-channel PC gaming headphones with detachable noise-filtering boom microphone."},
		{1, "SilikonVent Tali Jam Pengganti Berpori Keringat Gym", "ID-WCH-014", 59000.0, "IDR", 90, "https://images.unsplash.com/photo-1508685096489-7aacd43bd3b1?w=600", "Hemat", 4.6, "Tali silikon elastis dengan lubang ventilasi udara maksimal agar pergelangan tetap kering saat berolahraga."},
		{1, "AeroGrip Perforated Sweatproof Silicone Watch Strap", "EN-WCH-014", 3.69, "USD", 90, "https://images.unsplash.com/photo-1508685096489-7aacd43bd3b1?w=600", "Budget Pick", 4.6, "Soft hypoallergenic silicone band with air channel perforations designed for gym cross-training."},
		{2, "LenganPenyangga Boom Arm Mikrofon Rekaman Jepit Meja", "ID-CMP-014", 179000.0, "IDR", 30, "https://images.unsplash.com/photo-1590658006821-04f4008d5717?w=600", "Pilihan Editor", 4.8, "Lengan mekanis penopang mikrofon meja putar 360 derajat dengan pegas peredam getaran meja."},
		{2, "ProBroadcast Heavy-Duty Swivel Desk Microphone Boom Arm", "EN-CMP-014", 11.19, "USD", 30, "https://images.unsplash.com/photo-1590658006821-04f4008d5717?w=600", "Staff Pick", 4.8, "Studio-grade counterbalanced boom arm featuring internal tension springs and integrated cable channels."},
		{3, "Singlet Olahraga Gym Poliester Cepat Kering Hitam", "ID-MCL-014", 69000.0, "IDR", 80, "https://images.unsplash.com/photo-1521572267360-ee0c2909d518?w=600", "Hemat", 4.6, "Tanktop latihan angkat beban berpori mikro yang menyerap dan menguapkan keringat seketika."},
		{3, "Athletic Moisture-Wicking Performance Training Singlet", "EN-MCL-014", 4.31, "USD", 80, "https://images.unsplash.com/photo-1521572267360-ee0c2909d518?w=600", "Budget Pick", 4.6, "Ultra-breathable perforated mesh gym tank top engineered for high-mobility workout sets."},
		{4, "Celana Sirwal Katun Kasual Potongan Longgar Hitam", "ID-MPN-014", 159000.0, "IDR", 40, "https://images.unsplash.com/photo-1624378439575-d8705ad7ae80?w=600", "Favorit", 4.8, "Celana panjang longgar di atas mata kaki bahan katun sejuk dengan karet elastis di pinggang."},
		{4, "Easy-Fit Cotton Drawstring Lounge Pants Jet Black", "EN-MPN-014", 9.94, "USD", 40, "https://images.unsplash.com/photo-1624378439575-d8705ad7ae80?w=600", "Editor's Choice", 4.8, "Minimalist loose cotton leisure pants providing all-day comfort around the home or running errands."},
		{5, "Sepatu Bot Gurun Suede Tali Dua Lubang Cokelat Pasir", "ID-MSH-014", 389000.0, "IDR", 25, "https://images.unsplash.com/photo-1533867617858-e7b97e060509?w=600", "Pilihan Editor", 4.8, "Sepatu bot chukka klasik suede lembut dengan sol crepe elastis bergaya kasual maskulin."},
		{5, "Traditional British Perforated Wingtip Brogue Shoes Tan", "EN-MSH-014", 30.56, "USD", 20, "https://images.unsplash.com/photo-1533867617858-e7b97e060509?w=600", "Premium", 4.9, "Sophisticated dress shoes featuring ornate medallion brogue perforations and stacked heel."},
		{6, "Topi Rimba Nelayan Tali Dagu Pelindung Panas Matahari", "ID-MAC-014", 85000.0, "IDR", 50, "https://images.unsplash.com/photo-1588850561407-ed78c282e89b?w=600", "Sedang Tren", 4.7, "Bucket hat rimba dengan ventilasi samping dan tali dagu pengencang saat mendaki gunung."},
		{6, "Wide-Brim Safari Boonie Hat with Neck Drawstring Khaki", "EN-MAC-014", 5.31, "USD", 50, "https://images.unsplash.com/photo-1588850561407-ed78c282e89b?w=600", "Trending", 4.7, "UPF 50+ sun protection outdoor safari hat with breathable mesh side vents and chin cord."},
		{7, "Rok Jeans Denim Span Belahan Depan Washed Blue", "ID-WCL-014", 229000.0, "IDR", 30, "https://images.unsplash.com/photo-1515886657613-9f3515b0c78f?w=600", "Sedang Tren", 4.8, "Rok jeans panjang wanita potongan span dengan aksen belahan depan modis."},
		{7, "Sleeveless Square-Neck Knit Summer Jumpsuit Black", "EN-WCL-014", 15.56, "USD", 25, "https://images.unsplash.com/photo-1515886657613-9f3515b0c78f?w=600", "Hot Item", 4.8, "One-step chic wide-leg jumpsuit made from stretchy ribbed modal knit with side slash pockets."},
		{8, "Dompet Kartu Mini Resleting Gantungan Kunci Pastel", "ID-WBG-014", 59000.0, "IDR", 85, "https://images.unsplash.com/photo-1584917865442-de89df76afd3?w=600", "Hemat", 4.6, "Cardholder resleting mini untuk menyimpan koin, e-toll, dan kunci rumah dalam satu genggaman."},
		{8, "Minimalist Leather Card Case with Keychain D-Ring Mustard", "EN-WBG-014", 3.69, "USD", 85, "https://images.unsplash.com/photo-1584917865442-de89df76afd3?w=600", "Budget Pick", 4.6, "Slim card sleeve holding essential ID cards and keys on a sturdy brass spring ring."},
		{9, "Sandal Tali Belakang Slingback Hak Kaca Transparan", "ID-WSH-014", 239000.0, "IDR", 25, "https://images.unsplash.com/photo-1543163521-1bf539c55dd2?w=600", "Sedang Tren", 4.8, "Sandal heels wanita dengan tali belakang elastis dan hak kaca akrilik bening yang unik."},
		{9, "Transparent Lucite Sculptural Heel Slingbacks Clear/Nude", "EN-WSH-014", 14.94, "USD", 25, "https://images.unsplash.com/photo-1543163521-1bf539c55dd2?w=600", "Trending", 4.8, "Contemporary cocktail slingbacks featuring see-through PVC straps and architectural crystal lucite heel."},
		{10, "Teh Daun Kelor Organik Kering Kaya Antioksidan 75g", "ID-COF-014", 42000.0, "IDR", 75, "https://images.unsplash.com/photo-1576092768241-dec231879fc3?w=600", "Hemat", 4.6, "Daun kelor moringa organik murni kaya nutrisi mineral untuk membantu menjaga daya tahan tubuh harian."},
		{10, "Moroccan Mint Spearmint & Gunpowder Green Tea 100g", "EN-COF-014", 2.62, "USD", 75, "https://images.unsplash.com/photo-1576092768241-dec231879fc3?w=600", "Budget Pick", 4.6, "Traditional crisp blend of rolled green gunpowder pellets and aromatic spearmint leaves."},
		{11, "Cookies Oat Gandum Utuh Kurma Cokelat Bebas Gluten", "ID-SNK-014", 48000.0, "IDR", 65, "https://images.unsplash.com/photo-1599599810769-bcde5a160d32?w=600", "Pilihan Editor", 4.8, "Kue kering gandum oat panggang dengan pemanis alami buah kurma dan butiran dark chocolate."},
		{11, "Crunchy Baked Apple Cinnamon Crisps Zero Added Sugar", "EN-SNK-014", 2.19, "USD", 70, "https://images.unsplash.com/photo-1599599810769-bcde5a160d32?w=600", "Budget Pick", 4.7, "Crispy dehydrated Washington Red Delicious apple chips dusted with pure ground cinnamon."},
		{12, "Peeling Serum AHA BHA Eksfoliasi Kulit Mati Halus 30ml", "ID-SKN-014", 119000.0, "IDR", 45, "https://images.unsplash.com/photo-1556228720-195a672e8a03?w=600", "Sedang Tren", 4.8, "Serum eksfoliasi bilas mingguan mengangkat sel kulit mati agar wajah glowing bercahaya."},
		{12, "AHA 30% + BHA 2% Weekly Peeling Solution 30ml", "EN-SKN-014", 7.44, "USD", 45, "https://images.unsplash.com/photo-1556228720-195a672e8a03?w=600", "Trending", 4.8, "Ten-minute salon facial exfoliating peel smoothing coarse texture and revitalizing dull tone."},
		{13, "Minyak Rambut Kemiri Asli Menghitamkan dan Menyuburkan 100ml", "ID-BDY-014", 55000.0, "IDR", 80, "https://images.unsplash.com/photo-1592945403244-b3fbafd7f539?w=600", "Terlaris", 4.9, "Minyak biji kemiri bakar murni warisan leluhur untuk menebalkan dan mengkilapkan rambut."},
		{13, "100% Organic Cold-Pressed Argan Hair Shine Oil 100ml", "EN-BDY-014", 3.44, "USD", 80, "https://images.unsplash.com/photo-1592945403244-b3fbafd7f539?w=600", "Best Seller", 4.9, "Pure Moroccan argan oil taming unruly frizz, sealing split ends, and adding brilliant gloss."},
		{0, "TidurNyenyak Earbuds Silikon Pipih Relaksasi Malam", "ID-AUD-015", 429000.0, "IDR", 25, "https://images.unsplash.com/photo-1572536147248-ac59a8abfa4b?w=600", "Produk Baru", 4.8, "Earphone tidur berprofil sangat tipis tidak menekan daun telinga saat tidur miring."},
		{0, "DreamSleeper Ultra-Flat Contour Silicone Bedtime Earbuds", "EN-AUD-015", 26.81, "USD", 25, "https://images.unsplash.com/photo-1572536147248-ac59a8abfa4b?w=600", "New Arrival", 4.8, "Side-sleeper friendly soft earbuds blocking nighttime disturbances and ambient snoring."},
		{1, "KulitSapi Tali Smartwatch Jahitan Tangan Cokelat Tua", "ID-WCH-015", 169000.0, "IDR", 40, "https://images.unsplash.com/photo-1546868871-7041f2a55e12?w=600", "Populer", 4.8, "Tali jam kulit asli sentuhan vintage klasik yang memberikan nuansa kemewahan formal."},
		{1, "HeritageLeather Handcrafted Top-Grain Saddle Brown Strap", "EN-WCH-015", 10.56, "USD", 40, "https://images.unsplash.com/photo-1546868871-7041f2a55e12?w=600", "Trending", 4.8, "Genuine Italian cowhide replacement watch strap with stainless steel buckle and quick-release pins."},
		{2, "LampuLayar Bar Cahaya Monitor LED Sentuh Anti-Silau", "ID-CMP-015", 279000.0, "IDR", 40, "https://images.unsplash.com/photo-1587829741301-dc798b83add3?w=600", "Terlaris", 4.9, "Lampu gantung atas monitor hemat tempat yang menyinari meja kerja tanpa pantulan menyilaukan layar."},
		{2, "LuminaBeam Asymmetric Glare-Free Screenbar Monitor Lamp", "EN-CMP-015", 17.44, "USD", 40, "https://images.unsplash.com/photo-1587829741301-dc798b83add3?w=600", "Best Seller", 4.9, "Touch-controlled monitor light bar focusing illumination strictly onto work surface without screen reflections."},
		{3, "Cardigan Rajut Pria Kancing Depan Abu-Abu Gelap", "ID-MCL-015", 239000.0, "IDR", 25, "https://images.unsplash.com/photo-1556905055-8f358a7a47b2?w=600", "Sedang Tren", 4.8, "Outer rajut santai berkancing kayu dengan saku tempel depan untuk gaya elegan berlapis."},
		{3, "Structured Linen-Cotton Long Sleeve Resort Shirt White", "EN-MCL-015", 13.06, "USD", 40, "https://images.unsplash.com/photo-1596755094514-f87e34085b2c?w=600", "Best Seller", 4.9, "Crisp garment-washed linen-cotton blend dress shirt keeping you poised in warm climates."},
		{4, "Celana Jeans Hitam Pekat Potongan Lurus Klasik", "ID-MPN-015", 259000.0, "IDR", 30, "https://images.unsplash.com/photo-1542272604-780c96856592?w=600", "Wajib Punya", 4.9, "Jeans denim kaku hitam solid yang tidak luntur setelah dicuci berulang kali."},
		{4, "Classic Straight-Leg Jet Black Rigid Denim Jeans", "EN-MPN-015", 16.19, "USD", 30, "https://images.unsplash.com/photo-1542272604-780c96856592?w=600", "Essential", 4.9, "Pure non-stretch black denim trousers cut in an iconic heritage straight silhouette."},
		{5, "Sepatu Formal Pria Ukiran Wingtip Brogue Cokelat Tua", "ID-MSH-015", 489000.0, "IDR", 20, "https://images.unsplash.com/photo-1533867617858-e7b97e060509?w=600", "Produk Unggulan", 4.9, "Sepatu formal berlubang ukir wingtip klasik gaya bangsawan Inggris berbahan kulit sintetis mengkilap."},
		{5, "Durable High-Density Natural Rubber Beach Flip-Flops", "EN-MSH-015", 3.06, "USD", 100, "https://images.unsplash.com/photo-1549298916-b41d501d3772?w=600", "Budget Pick", 4.6, "Long-wearing non-slip natural rubber thong sandals with textured footbed pattern."},
		{6, "Gantungan Kunci Kulit Asli dengan Pengait Carabiner Kuningan", "ID-MAC-015", 55000.0, "IDR", 90, "https://images.unsplash.com/photo-1627123424574-724758594e93?w=600", "Wajib Punya", 4.8, "Gantungan kunci motor dan mobil berbahan kulit asli dengan cantolan logam kokoh."},
		{6, "Heavy Brass Keyring Lanyard with Lobster Swivel Hook", "EN-MAC-015", 3.44, "USD", 90, "https://images.unsplash.com/photo-1627123424574-724758594e93?w=600", "Essential", 4.8, "Solid brass spring carabiner clip keyholder connected to vegetable-tanned leather strap."},
		{7, "Rompi Rajut Crop Knitwear Wanita Gaya Korea Krem", "ID-WCL-015", 139000.0, "IDR", 40, "https://images.unsplash.com/photo-1515886657613-9f3515b0c78f?w=600", "Produk Baru", 4.7, "Vest rajut model kerah V bergaya santai Korea cocok dipadukan dengan kemeja putih."},
		{7, "Trench Coat Classic Double-Breasted Storm Flap Khaki", "EN-WCL-015", 21.81, "USD", 18, "https://images.unsplash.com/photo-1515886657613-9f3515b0c78f?w=600", "Premium", 4.9, "Heritage water-resistant cotton gabardine trench coat with buckled waist belt and storm shield."},
		{8, "Tas Selempang Kanvas Tahan Air Multi-Saku Santai Abu", "ID-WBG-015", 149000.0, "IDR", 45, "https://images.unsplash.com/photo-1584917865442-de89df76afd3?w=600", "Populer", 4.7, "Slingbag kasual harian berbahan kanvas tebal dengan 4 kantong ritsleting terpisah."},
		{8, "Lightweight Packable Ripstop Grocery Shopper Bag Floral", "EN-WBG-015", 2.81, "USD", 100, "https://images.unsplash.com/photo-1584917865442-de89df76afd3?w=600", "Essential", 4.8, "Reusable heavy-load shopping tote bag folding into its own compact self-contained pouch."},
		{9, "Sepatu Mary Jane Tali Tunggal Kulit Kilap Vintage Hitam", "ID-WSH-015", 259000.0, "IDR", 30, "https://images.unsplash.com/photo-1543163521-1bf539c55dd2?w=600", "Pilihan Editor", 4.9, "Sepatu Mary Jane gaya klasik dengan tali gesper tunggal di punggung kaki dan sol tebal."},
		{9, "Cloud-Foam Indoor/Outdoor Thick Sole Pillow Slides Mint", "EN-WSH-015", 4.94, "USD", 80, "https://images.unsplash.com/photo-1543163521-1bf539c55dd2?w=600", "Best Seller", 4.7, "Extra-thick 4cm compression-resistant recovery slides delivering cloud-like walking softness."},
		{10, "Biji Kopi Arabika Mandheling Sumatra Cita Rasa Tebal 250g", "ID-COF-015", 92000.0, "IDR", 50, "https://images.unsplash.com/photo-1514432324607-a09d9b4aefdd?w=600", "Terlaris", 4.9, "Kopi Mandheling proses giling basah legendaris dengan body tebal, keasaman rendah, dan aroma earthy rempah."},
		{10, "Organic Hibiscus Rosehip Berry Loose Herbal Infusion 100g", "EN-COF-015", 2.44, "USD", 90, "https://images.unsplash.com/photo-1576092768241-dec231879fc3?w=600", "Trending", 4.7, "Tart ruby-red botanical tisane packed with vitamin C and refreshing summer berry tang."},
		{11, "Keripik Jamur Tiram Krispi Bumbu Bawang Putih 100g", "ID-SNK-015", 35000.0, "IDR", 70, "https://images.unsplash.com/photo-1599599810769-bcde5a160d32?w=600", "Sedang Tren", 4.7, "Jamur tiram segar berbalut tepung bumbu rempah ditiriskan dengan mesin spinner bebas minyak berlebih."},
		{11, "Lightly Salted Crispy Organic Seaweed Snack Sheets 12pk", "EN-SNK-015", 1.56, "USD", 100, "https://images.unsplash.com/photo-1599599810769-bcde5a160d32?w=600", "Essential", 4.7, "Roasted nori seaweed crisped in organic sesame oil with light sea salt dusting."},
		{12, "Gel Lidah Buaya Murni 99% Penyejuk Kulit Terbakar Matahari", "ID-SKN-015", 49000.0, "IDR", 100, "https://images.unsplash.com/photo-1556228720-195a672e8a03?w=600", "Hemat", 4.8, "Gel serbaguna ekstrak aloe vera alami tanpa alkohol menyejukkan kulit kemerahan dan melembabkan tubuh."},
		{12, "Pure Organic Cold-Pressed Aloe Vera Soothing Gel 250ml", "EN-SKN-015", 3.06, "USD", 100, "https://images.unsplash.com/photo-1556228720-195a672e8a03?w=600", "Budget Pick", 4.8, "Multipurpose cooling botanical gel soothing razor burns, sunburns, and dry body skin."},
		{13, "Sabun Mandi Busa Melimpah Ekstrak Teh Hijau Menenangkan 300ml", "ID-BDY-015", 75000.0, "IDR", 60, "https://images.unsplash.com/photo-1592945403244-b3fbafd7f539?w=600", "Populer", 4.7, "Shower gel beraroma teh hijau segar menutrisi kulit dan memberikan perlindungan antibakteri."},
		{13, "Calming Chamomile & Green Tea Foaming Shower Gel 300ml", "EN-BDY-015", 4.69, "USD", 60, "https://images.unsplash.com/photo-1592945403244-b3fbafd7f539?w=600", "Top Rated", 4.7, "Gentle sulfate-free foaming body wash maintaining natural moisture balance during daily baths."},
		{0, "RanselLipat Headphone Travel Lipat On-Ear Praktis", "ID-AUD-016", 299000.0, "IDR", 30, "https://images.unsplash.com/photo-1546435770-a3e426bf472b?w=600", "Favorit", 4.7, "Headphone ringkas yang dapat dilipat masuk ke dalam tas jinjing dengan kabel lepas-pasang."},
		{0, "VoyagerFold Compact Swivel Travel Headphones with Hard Case", "EN-AUD-016", 18.69, "USD", 30, "https://images.unsplash.com/photo-1546435770-a3e426bf472b?w=600", "Popular", 4.7, "Lay-flat folding on-ear headphones with plush protein leather cushions and tangle-free cord."},
		{1, "PerisaiKaca Tempered Glass 3D Lengkung Layar Smartwatch", "ID-WCH-016", 49000.0, "IDR", 100, "https://images.unsplash.com/photo-1523275335684-37898b6baf30?w=600", "Wajib Punya", 4.7, "Pelindung layar anti gores kekerasan 9H dengan pinggiran melengkung halus tanpa mengurangi sensitivitas sentuhan."},
		{1, "ArmorShield 3D Curved 9H Tempered Glass Screen Guard", "EN-WCH-016", 3.06, "USD", 100, "https://images.unsplash.com/photo-1523275335684-37898b6baf30?w=600", "Essential", 4.7, "Oleophobic coated edge-to-edge protective glass shield preventing display scratches and fingerprint smudges."},
		{2, "GantungHeadset Pengait Headphone Meja Bawah Aluminium", "ID-CMP-016", 59000.0, "IDR", 80, "https://images.unsplash.com/photo-1505740420928-5e560c06d30e?w=600", "Wajib Punya", 4.8, "Gantungan headset logam penjepit bawah meja menghemat ruang kerja dan menjaga busa headphone tetap awet."},
		{2, "DeskAnchor Under-Desk Clamp-On Metal Headphone Mount", "EN-CMP-016", 3.69, "USD", 80, "https://images.unsplash.com/photo-1505740420928-5e560c06d30e?w=600", "Essential", 4.8, "Solid aluminum headset hook securing under desktops to liberate valuable workstation area."},
		{3, "Kaos Saku Dada Katun Combed 30s Cokelat Kopi", "ID-MCL-016", 95000.0, "IDR", 50, "https://images.unsplash.com/photo-1583743814966-8936f5b7be1a?w=600", "Populer", 4.7, "Kaos santai berdetail kantong saku di dada kiri dengan proses pewarnaan alami ramah lingkungan."},
		{3, "Seamless Base-Layer Stretch V-Neck Undershirt Black", "EN-MCL-016", 5.56, "USD", 70, "https://images.unsplash.com/photo-1583743814966-8936f5b7be1a?w=600", "Essential", 4.7, "Snug moisture-absorbing elastane blend base layer preventing sweat stains under suits."},
		{4, "Celana Renang Boardshorts Cepat Kering Warna Toska", "ID-MPN-016", 119000.0, "IDR", 55, "https://images.unsplash.com/photo-1624378439575-d8705ad7ae80?w=600", "Hemat", 4.6, "Celana santai pantai serat mikro tahan percikan air dengan tali serut anti-lepas."},
		{4, "Quick-Drying Water-Repellent Swim Boardshorts Cyan", "EN-MPN-016", 7.44, "USD", 55, "https://images.unsplash.com/photo-1624378439575-d8705ad7ae80?w=600", "Budget Pick", 4.6, "Hydrophobic beach trunks equipped with velcro fly, lace-up waist, and drainage back grommet."},
		{5, "Sandal Jepit Karet Alami Tahan Aus Warna Hitam Polos", "ID-MSH-016", 49000.0, "IDR", 100, "https://images.unsplash.com/photo-1549298916-b41d501d3772?w=600", "Hemat", 4.6, "Sandal jepit santai bahan getah karet alami padat empuk tahan air untuk harian."},
		{5, "Plush Shearling-Lined Suede Slipper Moccasins Warm Chestnut", "EN-MSH-016", 15.56, "USD", 30, "https://images.unsplash.com/photo-1533867617858-e7b97e060509?w=600", "Popular", 4.7, "Faux-fur lined indoor/outdoor driving moccasins finished with rawhide lace bow tie."},
		{6, "Dasi Sutra Pria Motif Garis Formal Biru Dongker Perak", "ID-MAC-016", 119000.0, "IDR", 45, "https://images.unsplash.com/photo-1627123424574-724758594e93?w=600", "Terlaris", 4.9, "Dasi formal tenun jacquard sutra sintetis motif garis elegan cocok untuk pernikahan dan rapat bisnis."},
		{6, "Silk Jacquard Woven Necktie Navy & Burgundy Stripe", "EN-MAC-016", 7.44, "USD", 45, "https://images.unsplash.com/photo-1627123424574-724758594e93?w=600", "Best Seller", 4.9, "3.15-inch standard width formal business silk tie with wool interlining for knot retention."},
		{7, "Jumpsuit Kasual Kerah V Tanpa Lengan Tali Pinggang", "ID-WCL-016", 249000.0, "IDR", 25, "https://images.unsplash.com/photo-1515886657613-9f3515b0c78f?w=600", "Pilihan Editor", 4.8, "One-piece jumpsuit bahan katun rami dengan tali pengikat pinggang yang mempermanis siluet."},
		{7, "Boho Peasant Top with Tasseled Drawstring Neck Mustard", "EN-WCL-016", 10.56, "USD", 35, "https://images.unsplash.com/photo-1515886657613-9f3515b0c78f?w=600", "Trending", 4.7, "Breezy bohemian tunic top with embroidered front bib and tassel-tipped braided tie strings."},
		{8, "Tas Laptop Jinjing Wanita Berbusa Tebal Warna Abu Lilac", "ID-WBG-016", 209000.0, "IDR", 30, "https://images.unsplash.com/photo-1584917865442-de89df76afd3?w=600", "Terlaris", 4.8, "Tas laptop jinjing wanita berukuran 13 hingga 15 inci dengan bantalan busa beludru anti-benturan."},
		{8, "Padded Velvet Laptop Sleeve with Zipper Front Pocket Teal", "EN-WBG-016", 13.06, "USD", 30, "https://images.unsplash.com/photo-1584917865442-de89df76afd3?w=600", "Best Seller", 4.8, "Shock-absorbing fleece lined protective sleeve with front accessory zipper for chargers."},
		{9, "Sandal Busa Tebal Rumah Empuk Anti-Licin Warna Mint", "ID-WSH-016", 79000.0, "IDR", 80, "https://images.unsplash.com/photo-1543163521-1bf539c55dd2?w=600", "Hemat", 4.7, "Sandal rumah sol tebal 4cm bahan EVA empuk serasa berjalan di atas awan."},
		{9, "Lightweight Air-Mesh Aerobic Walking Sneakers Berry", "EN-WSH-016", 17.44, "USD", 35, "https://images.unsplash.com/photo-1543163521-1bf539c55dd2?w=600", "Popular", 4.8, "Flexible lightweight workout trainers engineered with breathable mesh and shock-dampening heel pods."},
		{10, "Teh Rempah Wedang Uwuh Tradisional Jogja 10 Kantong", "ID-COF-016", 49000.0, "IDR", 70, "https://images.unsplash.com/photo-1576092768241-dec231879fc3?w=600", "Favorit", 4.8, "Minuman herbal hangat khas Imogiri berisi kayu secang, jahe merah, cengkeh, kayu manis, dan kapulaga."},
		{10, "Kenyan AA High-Acidity Bright Notes Specialty Coffee 12oz", "EN-COF-016", 6.88, "USD", 30, "https://images.unsplash.com/photo-1514432324607-a09d9b4aefdd?w=600", "Premium", 4.9, "Largest screen size Kenya AA beans bursting with blackcurrant, grapefuit, and complex winey acidity."},
		{11, "Minuman Sari Jahe Merah Instan Bubuk Gula Kelapa 200g", "ID-SNK-016", 39000.0, "IDR", 80, "https://images.unsplash.com/photo-1599599810769-bcde5a160d32?w=600", "Hemat", 4.8, "Minuman serbuk jahe merah hangat berpadu gula semut kelapa organik penangkal masuk angin."},
		{11, "Pure Maple Syrup Grade A Amber Rich Taste Glass 12oz", "EN-SNK-016", 6.88, "USD", 30, "https://images.unsplash.com/photo-1599599810769-bcde5a160d32?w=600", "Premium", 4.9, "100% single-farm tapped Vermont maple syrup delivering deep caramel and vanilla undertones."},
		{12, "Mist Semprotan Wajah Hidrasi Asam Hialuronat Segar 100ml", "ID-SKN-016", 69000.0, "IDR", 60, "https://images.unsplash.com/photo-1556228720-195a672e8a03?w=600", "Populer", 4.7, "Face mist semprotan halus mengunci kelembaban sebelum dan sesudah riasan wajah."},
		{12, "Botanical Rosewater Hydrating Face Spritz Mist 100ml", "EN-SKN-016", 4.31, "USD", 60, "https://images.unsplash.com/photo-1556228720-195a672e8a03?w=600", "Popular", 4.7, "Hydrosol rosewater facial mist delivering instant midday moisture refresh and setting makeup."},
		{13, "Eau de Parfum Vanila Madu Hangat Manis Tahan Lama 50ml", "ID-BDY-016", 189000.0, "IDR", 40, "https://images.unsplash.com/photo-1592945403244-b3fbafd7f539?w=600", "Sedang Tren", 4.8, "Parfum wangi manis gourmand aroma vanila madu panggang berpadu musk hangat mewah."},
		{13, "Warm Bourbon Vanilla & Spiced Amber Eau de Parfum 50ml", "EN-BDY-016", 11.81, "USD", 40, "https://images.unsplash.com/photo-1592945403244-b3fbafd7f539?w=600", "Trending", 4.8, "Intoxicating gourmand fragrance featuring aged Madagascar vanilla bean, tonka bean, and golden amber."},
		{0, "DuetKaraoke Paket 2 Mikrofon Nirkabel UHF Bebas Gangguan", "ID-AUD-017", 499000.0, "IDR", 20, "https://images.unsplash.com/photo-1590658006821-04f4008d5717?w=600", "Terlaris", 4.8, "Sepasang mikrofon panggung nirkabel dengan jangkauan 50 meter dan baterai tahan 10 jam."},
		{0, "TwinStage Professional Dual-Channel UHF Wireless Mic Set", "EN-AUD-017", 31.19, "USD", 20, "https://images.unsplash.com/photo-1590658006821-04f4008d5717?w=600", "Best Seller", 4.8, "Comprehensive twin handheld microphone setup with crystal-clear reception for events and churches."},
		{1, "TrioDaya Dudukan Charger Nirkabel 3-in-1 Meja Kerja", "ID-WCH-017", 289000.0, "IDR", 35, "https://images.unsplash.com/photo-1523275335684-37898b6baf30?w=600", "Sedang Tren", 4.8, "Stand pengisi daya nirkabel lipat untuk mengisi daya jam, earbuds, dan smartphone secara bersamaan."},
		{1, "PowerStation 3-in-1 Foldable Desktop Wireless Charger", "EN-WCH-017", 18.06, "USD", 35, "https://images.unsplash.com/photo-1523275335684-37898b6baf30?w=600", "Best Seller", 4.8, "Space-saving desk dock simultaneously fast-charging smartwatch, phone, and true wireless earbuds."},
		{2, "PelumasTombol Minyak Sintetis Sakelar Keyboard 5 Gram", "ID-CMP-017", 45000.0, "IDR", 100, "https://images.unsplash.com/photo-1587829741301-dc798b83add3?w=600", "Hemat", 4.9, "Pelumas khusus sakelar keyboard mekanikal untuk suara ketukan lebih bulat, padat, dan halus."},
		{2, "KeyLube Synthetic Mechanical Keyboard Switch Lubricant", "EN-CMP-017", 2.81, "USD", 100, "https://images.unsplash.com/photo-1587829741301-dc798b83add3?w=600", "Budget Pick", 4.9, "Specialist damping grease designed to smooth key stem friction and deepen keyboard acoustic tone."},
		{3, "Jaket Denim Washed Biru Medium Vintage 14oz", "ID-MCL-017", 249000.0, "IDR", 30, "https://images.unsplash.com/photo-1596755094514-f87e34085b2c?w=600", "Paling Dicari", 4.8, "Jaket jeans pria berbahan denim kaku tebal dengan kancing logam kuningan klasik."},
		{3, "Waffle-Knit Thermal Long Sleeve Layering Top Sand", "EN-MCL-017", 10.56, "USD", 35, "https://images.unsplash.com/photo-1556905055-8f358a7a47b2?w=600", "New Arrival", 4.8, "Honeycomb textured thermal long sleeve tee providing lightweight breathable insulation."},
		{4, "Celana Training Olahraga Garis Strip Samping Retro Navy", "ID-MPN-017", 159000.0, "IDR", 45, "https://images.unsplash.com/photo-1542272604-780c96856592?w=600", "Sedang Tren", 4.7, "Celana olahraga vintage berbahan trikot halus dengan aksen garis ganda putih di samping."},
		{4, "Retro Side-Tape Tricot Athletic Track Pants Dark Navy", "EN-MPN-017", 9.94, "USD", 45, "https://images.unsplash.com/photo-1542272604-780c96856592?w=600", "Trending", 4.7, "Sporty warm-up track trousers accented with contrast woven side piping and ankle zippers."},
		{5, "Sepatu Moccasin Slop Nyaman Jahitan Tepi Tangan Cokelat", "ID-MSH-017", 249000.0, "IDR", 30, "https://images.unsplash.com/photo-1533867617858-e7b97e060509?w=600", "Populer", 4.7, "Sepatu santai mengemudi berdetail jahitan tangan klasik di sekeliling ujung sepatu."},
		{5, "Responsive Memory Foam Everyday Walking Sneakers Navy", "EN-MSH-017", 17.44, "USD", 40, "https://images.unsplash.com/photo-1542291026-7eec264c27ff?w=600", "Favorite", 4.7, "Everyday commuter sneakers with memory foam footbeds designed to absorb walking fatigue."},
		{6, "Manset Kemeja Logam Polos Elegan Warna Perak Mewah", "ID-MAC-017", 89000.0, "IDR", 60, "https://images.unsplash.com/photo-1627123424574-724758594e93?w=600", "Produk Baru", 4.8, "Sepasang cufflinks kemeja formal bahan kuningan lapis perak berkotak eksklusif."},
		{6, "Brushed Sterling Silver Cufflinks in Presentation Case", "EN-MAC-017", 5.56, "USD", 60, "https://images.unsplash.com/photo-1627123424574-724758594e93?w=600", "New Arrival", 4.8, "Minimalist rectangular brushed metal cufflinks featuring easy swivel toggle back closures."},
		{7, "Kemeja Blouse Rayon Motif Etnik Batik Mega Mendung", "ID-WCL-017", 199000.0, "IDR", 35, "https://images.unsplash.com/photo-1515886657613-9f3515b0c78f?w=600", "Terlaris", 4.9, "Blouse batik modern warna biru awan khas Cirebon dengan kancing bungkus kain rapi."},
		{7, "Smocked Bodice Puff-Sleeve Cottagecore Dress Lavender", "EN-WCL-017", 13.69, "USD", 35, "https://images.unsplash.com/photo-1515886657613-9f3515b0c78f?w=600", "Editor's Choice", 4.8, "Charming square-neck midi dress featuring stretchy shirred elastic bust and elasticated cuff sleeves."},
		{8, "Tas Pesta Genggam Amplop Satin Mengkilap Emas Mewah", "ID-WBG-017", 189000.0, "IDR", 30, "https://images.unsplash.com/photo-1584917865442-de89df76afd3?w=600", "Pilihan Editor", 4.8, "Envelope clutch berbalut kain satin emas mengkilap dengan penutup magnetik tersembunyi."},
		{8, "Chic Woven Knot Dumpling Handbag Off-White", "EN-WBG-017", 11.81, "USD", 30, "https://images.unsplash.com/photo-1584917865442-de89df76afd3?w=600", "Trending", 4.8, "Soft vegan leather gathered dumpling pouch with playful knotted loop wrist handle."},
		{9, "Sepatu Olahraga Wanita Aerobik Lari Jalan Kaki Lavender", "ID-WSH-017", 279000.0, "IDR", 35, "https://images.unsplash.com/photo-1543163521-1bf539c55dd2?w=600", "Terlaris", 4.8, "Sepatu olahraga wanita berbobot sangat ringan dengan sol peredam getaran untuk zumba dan jogging."},
		{9, "Multi-Strap Outdoor Hiking Webbing Sandals Sage Green", "EN-WSH-017", 9.94, "USD", 50, "https://images.unsplash.com/photo-1543163521-1bf539c55dd2?w=600", "Favorite", 4.7, "Water-friendly trail sandals with quick-drying patterned webbing straps and contoured traction footbed."},
		{10, "Kopi Lanang Peaberry Tunggal Robusta Khasiat Tinggi 200g", "ID-COF-017", 79000.0, "IDR", 40, "https://images.unsplash.com/photo-1514432324607-a09d9b4aefdd?w=600", "Paling Dicari", 4.8, "Biji kopi bulat tunggal langka pilihan petani yang dipercaya meningkatkan stamina dan vitalitas."},
		{10, "Golden Turmeric Ginger Botanical Wellness Tea Blend 80g", "EN-COF-017", 3.06, "USD", 70, "https://images.unsplash.com/photo-1576092768241-dec231879fc3?w=600", "Hot Item", 4.8, "Revitalizing Ayurvedic herbal infusion combining organic root turmeric, spicy ginger, and lemongrass."},
		{11, "Keripik Emping Melinjo Manis Pedas Renyah Khas Limpung", "ID-SNK-017", 42000.0, "IDR", 60, "https://images.unsplash.com/photo-1599599810769-bcde5a160d32?w=600", "Populer", 4.7, "Emping melinjo renyah berlapis karamel gula merah cabai manis gurih tanpa rasa pahit berlebih."},
		{11, "Gluten-Free Ancient Grain Multiseed Crackers 6oz", "EN-SNK-017", 1.81, "USD", 80, "https://images.unsplash.com/photo-1599599810769-bcde5a160d32?w=600", "Healthy Choice", 4.7, "Crunchy artisan crackers baked with flaxseed, quinoa, sesame, and cracked black pepper."},
		{12, "Sabun Wajah Batang Minyak Zaitun Alami Bebas SLS 100g", "ID-SKN-017", 39000.0, "IDR", 85, "https://images.unsplash.com/photo-1556228720-195a672e8a03?w=600", "Hemat", 4.7, "Sabun batang alami proses dingin kaya gliserin nabati aman untuk kulit sensitif dan eksim."},
		{12, "Gentle Non-Stripping Squalane Jelly Cleanser 120ml", "EN-SKN-017", 5.56, "USD", 65, "https://images.unsplash.com/photo-1556228720-195a672e8a03?w=600", "New Arrival", 4.8, "Conditioning jelly-to-milk daily facial cleanser leaving skin exceptionally soft and hydrated."},
		{13, "Minyak Wangi Kasturi Putih Non-Alkohol Oles Murni 12ml", "ID-BDY-017", 69000.0, "IDR", 70, "https://images.unsplash.com/photo-1592945403244-b3fbafd7f539?w=600", "Paling Dicari", 4.9, "Attar minyak wangi oles roll-on konsentrat putih suci tanpa alkohol aroma lembut awet seharian."},
		{13, "Concentrated Pure Fragrance Rollerball Perfume Oil 10ml", "EN-BDY-017", 4.31, "USD", 70, "https://images.unsplash.com/photo-1592945403244-b3fbafd7f539?w=600", "Staff Pick", 4.9, "Pocket-friendly alcohol-free roll-on perfume oil delivering intimate long-lasting scent sillage."},
		{0, "KamarMandiTunes Speaker Bluetooth Tempel Dinding Anti-Air IPX7", "ID-AUD-018", 159000.0, "IDR", 50, "https://images.unsplash.com/photo-1608043152269-423dbba4e7e1?w=600", "Hemat", 4.6, "Speaker bulat tahan rendaman air dengan karet hisap kuat untuk mendengarkan lagu saat mandi."},
		{0, "AquaSonic Submersible IPX7 Bathroom Shower Speaker", "EN-AUD-018", 9.94, "USD", 50, "https://images.unsplash.com/photo-1608043152269-423dbba4e7e1?w=600", "Budget Pick", 4.6, "Waterproof suction-cup shower speaker with tactile volume buttons and built-in speakerphone."},
		{1, "RenangPro Smartwatch Penghitung Ayunan Lengan Gaya Dada", "ID-WCH-018", 549000.0, "IDR", 25, "https://images.unsplash.com/photo-1508685096489-7aacd43bd3b1?w=600", "Pilihan Editor", 4.7, "Smartwatch renang tahan air tinggi dengan deteksi otomatis gaya renang, skor SWOLF, dan hitungan putaran."},
		{1, "HydroLap Swim Cadence & SWOLF Efficiency Smartwatch", "EN-WCH-018", 34.31, "USD", 25, "https://images.unsplash.com/photo-1508685096489-7aacd43bd3b1?w=600", "Staff Pick", 4.7, "Dedicated pool and open-water swimming tracker monitoring stroke rate, distance, and calorie expenditure."},
		{2, "LingkarCahaya Lampu Ring Light Meja USB Dudukan HP", "ID-CMP-018", 139000.0, "IDR", 50, "https://images.unsplash.com/photo-1587829741301-dc798b83add3?w=600", "Populer", 4.7, "Lampu bundar pengatur 3 tingkat suhu warna untuk pencahayaan wajah jernih saat meeting daring."},
		{2, "GlowStudio Dimmable USB Desktop Ring Light with Ball-Head", "EN-CMP-018", 8.69, "USD", 50, "https://images.unsplash.com/photo-1587829741301-dc798b83add3?w=600", "Popular", 4.7, "Even-diffusion circular LED light with warm to cool color presets for videoconferences and portraits."},
		{3, "Baju Koko Kurta Pria Katun Toyobo Putih Polos", "ID-MCL-018", 179000.0, "IDR", 35, "https://images.unsplash.com/photo-1596755094514-f87e34085b2c?w=600", "Favorit", 4.9, "Busana muslim pria model kurta modern berkerah shanghai dengan bahan katun halus adem."},
		{3, "Casual Double-Pocket Workwear Chambray Shirt Blue", "EN-MCL-018", 11.19, "USD", 35, "https://images.unsplash.com/photo-1596755094514-f87e34085b2c?w=600", "Trending", 4.8, "Rugged lightweight indigo chambray cotton shirt detailed with contrast triple-stitching."},
		{4, "Celana Kerja Formal Lipat Depan Warna Abu Tua", "ID-MPN-018", 249000.0, "IDR", 40, "https://images.unsplash.com/photo-1624378439575-d8705ad7ae80?w=600", "Terlaris", 4.8, "Celana kantor formal dengan aksen ploi lipit depan memberikan ruang gerak pinggul leluasa."},
		{4, "Double-Pleated Formal Dress Slacks with Waist Tabs Charcoal", "EN-MPN-018", 15.56, "USD", 40, "https://images.unsplash.com/photo-1624378439575-d8705ad7ae80?w=600", "Best Seller", 4.8, "Refined double-pleat dress pants featuring side buckle adjusters eliminating the need for belts."},
		{5, "Sepatu Jalan Kaki Insole Busa Memori Biru Dongker", "ID-MSH-018", 279000.0, "IDR", 40, "https://images.unsplash.com/photo-1542291026-7eec264c27ff?w=600", "Favorit", 4.7, "Sepatu jalan santai dengan bantalan insole memory foam peredam hentakan saat berdiri lama."},
		{5, "Eco-Friendly Foam Sneaker Cleaner and Detailing Brush Kit", "EN-MSH-018", 4.31, "USD", 80, "https://images.unsplash.com/photo-1549298916-b41d501d3772?w=600", "Essential", 4.8, "Ready-to-use foaming cleaner safe for leather, suede, nubuck, and technical knit fabrics."},
		{6, "Pouch Tas Genggam Kulit Pria Handbag Serbaguna Cokelat", "ID-MAC-018", 179000.0, "IDR", 35, "https://images.unsplash.com/photo-1627123424574-724758594e93?w=600", "Populer", 4.8, "Clutch bag pria dengan tali pergelangan tangan untuk membawa dompet, kunci mobil, dan rokok."},
		{6, "Executive Leather Tech Portfolio Clutch Pouch Tan", "EN-MAC-018", 11.19, "USD", 35, "https://images.unsplash.com/photo-1627123424574-724758594e93?w=600", "Popular", 4.8, "Structured top-zip tech organizer pouch holding iPad mini, cables, chargers, and notebooks."},
		{7, "Kaos Rib Katun Rajut Kerah Tinggi Turtle Neck Abu", "ID-WCL-018", 99000.0, "IDR", 65, "https://images.unsplash.com/photo-1515886657613-9f3515b0c78f?w=600", "Wajib Punya", 4.7, "Atasan kaos bertekstur rib rajut elastis dengan kerah turtleneck hangat dan modis."},
		{7, "Slouchy Distressed Denim Jacket Oversized Vintage Wash", "EN-WCL-018", 16.81, "USD", 25, "https://images.unsplash.com/photo-1515886657613-9f3515b0c78f?w=600", "Favorite", 4.8, "Drop-shoulder relaxed boyfriend jean jacket with authentic hand-sanded fades and chest pockets."},
		{8, "Tas Selempang Mini Tempat HP dan Dompet Uang Praktis", "ID-WBG-018", 89000.0, "IDR", 75, "https://images.unsplash.com/photo-1584917865442-de89df76afd3?w=600", "Wajib Punya", 4.7, "Tas kecil khusus smartphone layar besar dengan slot kartu di bagian belakang."},
		{8, "Micro Crossbody Phone Bag with Card Slots Terracotta", "EN-WBG-018", 5.56, "USD", 75, "https://images.unsplash.com/photo-1584917865442-de89df76afd3?w=600", "Popular", 4.7, "Compact vertical phone pouch with back exterior card slots and snap-button security tab."},
		{9, "Sandal Gunung Wanita Tali Webbing Pastel Ringan", "ID-WSH-018", 159000.0, "IDR", 50, "https://images.unsplash.com/photo-1543163521-1bf539c55dd2?w=600", "Populer", 4.7, "Sandal outdoor wanita dengan tali webbing warna pastel lembut dan sol berkontur empuk."},
		{9, "Plush Shearling Fur Slippers with Rubber Sole Warm Gray", "EN-WSH-018", 10.56, "USD", 35, "https://images.unsplash.com/photo-1543163521-1bf539c55dd2?w=600", "New Arrival", 4.8, "Cross-band faux-shearling fuzzy house slippers built over a durable outdoor tread sole."},
		{10, "Teh Putih Silver Needle Pucuk Daun Langka Organik 30g", "ID-COF-018", 135000.0, "IDR", 20, "https://images.unsplash.com/photo-1576092768241-dec231879fc3?w=600", "Produk Unggulan", 4.9, "Pucuk daun teh putih termurni yang dipetik pagi hari sebelum mekar dengan rasa manis lembut alami."},
		{10, "Darjeeling First Flush Himalayan Loose Black Tea 50g", "EN-COF-018", 5.94, "USD", 35, "https://images.unsplash.com/photo-1576092768241-dec231879fc3?w=600", "Editor's Choice", 4.9, "The champagne of teas harvested during spring in misty Himalayan foothills with delicate muscatel flavor."},
		{11, "Tepung Biji Gandum Utuh Organik Serat Tinggi 500g", "ID-SNK-018", 38000.0, "IDR", 70, "https://images.unsplash.com/photo-1599599810769-bcde5a160d32?w=600", "Wajib Punya", 4.7, "Tepung whole wheat murni untuk pembuatan roti tawar gandum dan pancake sehat kaya serat."},
		{11, "Organic Medjool Large Whole Dates Fresh Pack 16oz", "EN-SNK-018", 4.94, "USD", 40, "https://images.unsplash.com/photo-1599599810769-bcde5a160d32?w=600", "Staff Pick", 4.9, "Plump, naturally sweet King of Dates packed with fiber, potassium, and magnesium."},
		{12, "Serum Alpha Arbutin 2% Pencerah Noda Hitam Wajah 30ml", "ID-SKN-018", 139000.0, "IDR", 40, "https://images.unsplash.com/photo-1556228720-195a672e8a03?w=600", "Paling Dicari", 4.8, "Serum konsentrat pencerah menghambat produksi melanin untuk menyamarkan bintik hitam penuaan."},
		{12, "Dark Spot Corrector Tranexamic Acid 3% Serum 30ml", "EN-SKN-018", 8.69, "USD", 40, "https://images.unsplash.com/photo-1556228720-195a672e8a03?w=600", "Top Rated", 4.8, "Targeted brightening solution fading stubborn post-inflammatory hyperpigmentation and sun spots."},
		{13, "Losion Tabir Surya Tubuh SPF 30 Ekstrak Lidah Buaya 150ml", "ID-BDY-018", 85000.0, "IDR", 50, "https://images.unsplash.com/photo-1592945403244-b3fbafd7f539?w=600", "Wajib Punya", 4.7, "Sunscreen badan melindungi kulit lengan dan kaki dari belang sinar matahari saat keluar rumah."},
		{13, "Daily Defense Hydrating Body Sunscreen Lotion SPF 30 150ml", "EN-BDY-018", 5.31, "USD", 50, "https://images.unsplash.com/photo-1592945403244-b3fbafd7f539?w=600", "Essential", 4.7, "Broad-spectrum daily body lotion with broad UVA/UVB protection and soothing aloe vera juice."},
		{0, "NadaEmas In-Ear Monitor 4-Driver Kabel Tembaga Berlapis Perak", "ID-AUD-019", 899000.0, "IDR", 18, "https://images.unsplash.com/photo-1572536147248-ac59a8abfa4b?w=600", "Produk Unggulan", 4.9, "IEM kabel lepas MMCX dengan 4 unit armature seimbang untuk pemisahan instrumen yang sangat akurat."},
		{0, "ApexHarmonics Quad Balanced Armature Audiophile In-Ear Monitors", "EN-AUD-019", 56.19, "USD", 18, "https://images.unsplash.com/photo-1572536147248-ac59a8abfa4b?w=600", "Premium", 4.9, "Precision-tuned four-driver acoustic monitors equipped with silver braided MMCX cable."},
		{1, "NilonRajut Tali Jam Velcro Elastis Warna Zaitun", "ID-WCH-019", 79000.0, "IDR", 60, "https://images.unsplash.com/photo-1575311373937-040b8e1fd5b6?w=600", "Populer", 4.7, "Tali jam anyaman nilon rajut mikro yang sangat empuk di kulit dengan sistem rekat velcro cepat."},
		{1, "FlexWeave Soft Woven Nylon Loop Strap Olive Green", "EN-WCH-019", 4.94, "USD", 60, "https://images.unsplash.com/photo-1575311373937-040b8e1fd5b6?w=600", "Popular", 4.7, "Double-layer breathable woven nylon wristband with hook-and-loop fastener for all-day comfort."},
		{2, "PenyanggaLayar Lengan Gas Spring Monitor Tunggal 32 Inci", "ID-CMP-019", 349000.0, "IDR", 25, "https://images.unsplash.com/photo-1587829741301-dc798b83add3?w=600", "Pilihan Editor", 4.9, "Bracket monitor hidrolik bebas diatur ketinggian, sudut kemiringan, dan orientasi vertikal horisontal."},
		{2, "GasFlex Single Gas-Spring Ergonomic Monitor Desk Arm", "EN-CMP-019", 21.81, "USD", 25, "https://images.unsplash.com/photo-1587829741301-dc798b83add3?w=600", "Staff Pick", 4.9, "Fluid motion articulating VESA monitor arm supporting displays from 17 to 32 inches with full tilt."},
		{3, "Kaos Garis Belang Katun CVC Garis Putih Biru Laut", "ID-MCL-019", 125000.0, "IDR", 45, "https://images.unsplash.com/photo-1521572267360-ee0c2909d518?w=600", "Sedang Tren", 4.7, "Kaos motif garis pelaut klasik berbahan katun lembut yang tidak susut setelah dicuci."},
		{3, "Two-Tone Raglan Baseball Sleeve Graphic T-Shirt", "EN-MCL-019", 7.44, "USD", 55, "https://images.unsplash.com/photo-1521572267360-ee0c2909d518?w=600", "Budget Pick", 4.6, "Retro contrast raglan 3/4 sleeve athletic tee designed with flexible mobility in mind."},
		{4, "Celana Pendek Denim Kasual Lipat Bawah Biru Tua", "ID-MPN-019", 159000.0, "IDR", 45, "https://images.unsplash.com/photo-1542272604-780c96856592?w=600", "Populer", 4.7, "Celana pendek jeans kasual dengan aksen lipatan tepi bawah berjahit rapi."},
		{4, "Distressed Raw-Hem Summer Denim Cut-Off Shorts Blue", "EN-MPN-019", 9.94, "USD", 45, "https://images.unsplash.com/photo-1542272604-780c96856592?w=600", "Popular", 4.7, "Washed vintage denim shorts with authentic hand-frayed raw cut leg hems."},
		{5, "Paket Pembersih Sepatu Busa Pembersih dan Sikat Kuda", "ID-MSH-019", 69000.0, "IDR", 80, "https://images.unsplash.com/photo-1549298916-b41d501d3772?w=600", "Wajib Punya", 4.8, "Cairan pembersih sepatu busa instan ramah lingkungan lengkap dengan sikat bulu kuda halus."},
		{5, "Shock-Absorbing Silicone Honeycomb Orthotic Insoles", "EN-MSH-019", 2.44, "USD", 110, "https://images.unsplash.com/photo-1542291026-7eec264c27ff?w=600", "Budget Pick", 4.7, "Honeycomb ergonomic gel inserts relieving plantar fasciitis tension during long walking shifts."},
		{6, "Payung Lipat Otomatis Buka Tutup Tombol Hitam Kokoh", "ID-MAC-019", 99000.0, "IDR", 65, "https://images.unsplash.com/photo-1627123424574-724758594e93?w=600", "Wajib Punya", 4.7, "Payung lipat mekanis satu tombol dengan 10 jari rangka baja tahan terpaan angin kencang."},
		{6, "Windproof Automatic Open-Close Compact Umbrella Black", "EN-MAC-019", 6.19, "USD", 65, "https://images.unsplash.com/photo-1627123424574-724758594e93?w=600", "Essential", 4.7, "Teflon-coated water-repellent travel umbrella with 10-rib reinforced fiberglass canopy frame."},
		{7, "Dress Pendek A-Line Katun Polos Aksen Kancing Depan", "ID-WCL-019", 179000.0, "IDR", 35, "https://images.unsplash.com/photo-1515886657613-9f3515b0c78f?w=600", "Populer", 4.7, "Mini dress kasual berpotongan A-Line bahan katun lembut untuk hangout santai."},
		{7, "Satin Pleated Midi Skirt with Elasticated Waist Champagne", "EN-WCL-019", 9.94, "USD", 40, "https://images.unsplash.com/photo-1515886657613-9f3515b0c78f?w=600", "Best Seller", 4.8, "Lustrous micro-accordion pleated midi skirt flowing gracefully from day to evening events."},
		{8, "Bag Charm Gantungan Tas Bulu Halus Aksen Pita Mutiara", "ID-WBG-019", 49000.0, "IDR", 90, "https://images.unsplash.com/photo-1584917865442-de89df76afd3?w=600", "Produk Baru", 4.6, "Aksesoris gantungan tas bulu pompom lembut dengan pengait klip emas mempercantik tas jinjing."},
		{8, "Faux Fur Pom-Pom Bag Charm with Gold Clip Lavender", "EN-WBG-019", 3.06, "USD", 90, "https://images.unsplash.com/photo-1584917865442-de89df76afd3?w=600", "New Arrival", 4.6, "Fluffy high-density faux fur keychain accessory with metallic lobster claw attachment."},
		{9, "Bantalan Sepatu Silikon Busa Pelindung Tumit Belakang", "ID-WSH-019", 29000.0, "IDR", 120, "https://images.unsplash.com/photo-1543163521-1bf539c55dd2?w=600", "Wajib Punya", 4.8, "Stiker bantalan silikon lembut ditempel di bagian belakang sepatu mencegah lecet tumit."},
		{9, "Silicone Gel Ball-of-Foot Anti-Slip Shoe Cushions", "EN-WSH-019", 1.81, "USD", 120, "https://images.unsplash.com/photo-1543163521-1bf539c55dd2?w=600", "Essential", 4.8, "Transparent self-adhesive metatarsal gel pads eliminating burning forefoot friction in high heels."},
		{10, "Bubuk Minuman Cokelat Hitam Kakao Murni Sulawesi 200g", "ID-COF-019", 58000.0, "IDR", 60, "https://images.unsplash.com/photo-1514432324607-a09d9b4aefdd?w=600", "Sedang Tren", 4.8, "Bubuk kakao murni 100% tanpa gula hasil olahan biji kakao fermentasi petani Sulawesi Selatan."},
		{10, "Artisanal Dark Cocoa 72% Drinking Chocolate Powder 250g", "EN-COF-019", 3.62, "USD", 60, "https://images.unsplash.com/photo-1514432324607-a09d9b4aefdd?w=600", "Popular", 4.8, "Single-origin organic dark cocoa flakes creating rich, velvety European style hot sipping chocolate."},
		{11, "Manisan Mangga Kering Alami Tanpa Pewarna Buatan 150g", "ID-SNK-019", 45000.0, "IDR", 60, "https://images.unsplash.com/photo-1599599810769-bcde5a160d32?w=600", "Produk Baru", 4.8, "Irisan buah mangga gedong gincu kering legit dengan rasa asam manis segar alami."},
		{11, "Roasted Wasabi Green Peas Spicy Crunchy Snack 10oz", "EN-SNK-019", 2.0, "USD", 75, "https://images.unsplash.com/photo-1599599810769-bcde5a160d32?w=600", "Trending", 4.7, "Crispy green marrowfat peas coated in fiery authentic Japanese wasabi horseradish seasoning."},
		{12, "Krim Malam Bernutrisi Minyak Biji Anggur & Vitamin E", "ID-SKN-019", 149000.0, "IDR", 35, "https://images.unsplash.com/photo-1556228720-195a672e8a03?w=600", "Pilihan Editor", 4.8, "Night cream kaya asam lemak esensial memperbaiki elastisitas kulit selagi Anda terlelap."},
		{12, "Antioxidant Daily Face Oil with Rosehip & Jojoba 30ml", "EN-SKN-019", 9.31, "USD", 35, "https://images.unsplash.com/photo-1556228720-195a672e8a03?w=600", "Editor's Choice", 4.8, "Cold-pressed non-greasy facial elixir boosting skin radiance and sealing in essential moisture."},
		{13, "Lilin Aromaterapi Kedelai Alami Wangi Bunga Cempaka", "ID-BDY-019", 69000.0, "IDR", 55, "https://images.unsplash.com/photo-1592945403244-b3fbafd7f539?w=600", "Pilihan Editor", 4.8, "Lilin aromaterapi soy wax ramah lingkungan dalam gelas kaca untuk suasana kamar tidur damai."},
		{13, "Hand-Poured Natural Soy Wax Candle Amber & Oakmoss 200g", "EN-BDY-019", 4.31, "USD", 55, "https://images.unsplash.com/photo-1592945403244-b3fbafd7f539?w=600", "Editor's Choice", 4.8, "Clean-burning non-toxic soy wax candle with cotton wick filling rooms with cozy forest warmth."},
		{0, "BusaSejuk Bantalan Telinga Headphone Pengganti Bahan Beludru", "ID-AUD-020", 79000.0, "IDR", 80, "https://images.unsplash.com/photo-1505740420928-5e560c06d30e?w=600", "Wajib Punya", 4.7, "Earpad universal lapisan kain beludru sejuk berbusa empuk nyaman tanpa membuat telinga berkeringat."},
		{0, "VelvetComfort Cooling Gel Memory Foam Headphone Ear Cushions", "EN-AUD-020", 4.94, "USD", 80, "https://images.unsplash.com/photo-1505740420928-5e560c06d30e?w=600", "Essential", 4.7, "Universal breathable velvet replacement ear cushions keeping ears fatigue-free and cool."},
		{1, "LacakBarang Gantungan Kunci Bluetooth Anti-Hilang Pintar", "ID-WCH-020", 119000.0, "IDR", 80, "https://images.unsplash.com/photo-1605100804763-247f67b3557e?w=600", "Terlaris", 4.8, "Alat pelacak barang hilang nirkabel dengan alarm bel dua arah dan penunjuk lokasi peta terakhir."},
		{1, "FinderPro Smart Bluetooth Loss Prevention Locator Tag", "EN-WCH-020", 7.44, "USD", 80, "https://images.unsplash.com/photo-1605100804763-247f67b3557e?w=600", "Hot Item", 4.8, "Ultra-compact locator disc with loud buzzer alert and smartphone map integration for keys and bags."},
		{2, "BagiSuara Kabel Cabang Audio Headset 2-ke-1 Berlapis Emas", "ID-CMP-020", 29000.0, "IDR", 120, "https://images.unsplash.com/photo-1505740420928-5e560c06d30e?w=600", "Hemat", 4.7, "Kabel konverter pemisah colokan audio mikrofon dan headphone 3.5mm untuk PC komputer desktop."},
		{2, "AudioSplit Gold-Plated 3.5mm Headphone & Mic Y-Cable", "EN-CMP-020", 1.81, "USD", 120, "https://images.unsplash.com/photo-1505740420928-5e560c06d30e?w=600", "Budget Pick", 4.7, "Oxygen-free copper braided splitter adapter connecting 4-pole headsets into dual PC audio ports."},
		{3, "Rompi Rajut Pria Tanpa Lengan Hijau Sage Gaya Santai", "ID-MCL-020", 169000.0, "IDR", 35, "https://images.unsplash.com/photo-1556905055-8f358a7a47b2?w=600", "Produk Baru", 4.8, "Vest rajutan tanpa lengan gaya preppy dipadukan sempurna di atas kemeja polos."},
		{3, "Knitted Sleeveless V-Neck Layering Sweater Vest Sage", "EN-MCL-020", 10.56, "USD", 35, "https://images.unsplash.com/photo-1556905055-8f358a7a47b2?w=600", "Editor's Choice", 4.8, "Vintage-inspired ribbed sleeveless knit vest designed to layer effortlessly over collared shirts."},
		{4, "Celana Boxer Katun Santai Motif Kotak Minimalis Isi 3", "ID-MPN-020", 89000.0, "IDR", 90, "https://images.unsplash.com/photo-1542272604-780c96856592?w=600", "Wajib Punya", 4.8, "Paket isi 3 celana boxer tidur katun murni berpori sejuk dengan karet pinggang tertutup kain."},
		{4, "Pure Woven Cotton Boxer Shorts 3-Pack Minimalist Grid", "EN-MPN-020", 5.56, "USD", 90, "https://images.unsplash.com/photo-1542272604-780c96856592?w=600", "Essential", 4.8, "Trio of breathable cotton poplin sleep boxers with elasticated gathered waistband."},
		{5, "Sol Tambahan Insole Gel Silikon Peredam Pegal Tumit", "ID-MSH-020", 39000.0, "IDR", 110, "https://images.unsplash.com/photo-1542291026-7eec264c27ff?w=600", "Hemat", 4.7, "Insole gel sarang lebah yang empuk meredakan tekanan beban tumit saat beraktivitas seharian."},
		{5, "Minimalist Monochrome Low-Profile Leather Trainers White", "EN-MSH-020", 20.56, "USD", 30, "https://images.unsplash.com/photo-1549298916-b41d501d3772?w=600", "Best Seller", 4.8, "Sleek low-top white luxury court sneakers with cushioned heel collar and tonal waxed laces."},
		{6, "Kotak Jam Tangan Isi 6 Slot Kulit Sintetis Kaca Transparan", "ID-MAC-020", 159000.0, "IDR", 35, "https://images.unsplash.com/photo-1627123424574-724758594e93?w=600", "Pilihan Editor", 4.9, "Kotak koleksi jam tangan dengan bantal busa beludru dan tutup kaca bening pelindung debu."},
		{6, "Wood & Vegan Leather 6-Slot Watch Display Collector Box", "EN-MAC-020", 9.94, "USD", 35, "https://images.unsplash.com/photo-1627123424574-724758594e93?w=600", "Staff Pick", 4.9, "Padded velvet interior timepiece showcase box with scratch-resistant real glass window."},
		{7, "Jaket Crop Denim Wanita Cuci Vintage Biru Muda", "ID-WCL-020", 269000.0, "IDR", 25, "https://images.unsplash.com/photo-1515886657613-9f3515b0c78f?w=600", "Favorit", 4.8, "Jaket jeans wanita potongan crop pendek sepinggang dengan saku berkancing kuningan."},
		{7, "Cozy Waffle-Knit Raglan Crewneck Loungewear Top Oatmeal", "EN-WCL-020", 8.69, "USD", 45, "https://images.unsplash.com/photo-1515886657613-9f3515b0c78f?w=600", "New Arrival", 4.7, "Soft thermal textured lounge sweater with raglan shoulder seams and banded cuffs."},
		{8, "Tas Jinjing Kerja Wanita Tiga Ruang Sekat Dalam Cokelat", "ID-WBG-020", 289000.0, "IDR", 20, "https://images.unsplash.com/photo-1584917865442-de89df76afd3?w=600", "Produk Unggulan", 4.9, "Executive tote bag dengan 3 sekat utama luas muat map dokumen A4 dan botol minum."},
		{8, "Triple-Compartment Executive Work Tote Bag Espresso", "EN-WBG-020", 18.06, "USD", 20, "https://images.unsplash.com/photo-1584917865442-de89df76afd3?w=600", "Staff Pick", 4.9, "Spacious business tote holding folders, tablets, and daily essentials in organized compartments."},
		{9, "Sepatu Hak Pendek Kitten Heels Elegan Kerja Kantor Hitam", "ID-WSH-020", 219000.0, "IDR", 35, "https://images.unsplash.com/photo-1543163521-1bf539c55dd2?w=600", "Terlaris", 4.8, "Kitten heels 4cm berujung lancip dengan desain profesional sopan untuk rapat kerja."},
		{9, "Pointed Kitten-Heel D'Orsay Pumps Classic Burgundy", "EN-WSH-020", 13.69, "USD", 35, "https://images.unsplash.com/photo-1543163521-1bf539c55dd2?w=600", "Editor's Choice", 4.8, "Graceful two-piece D'Orsay silhouette pumps with 1.75-inch kitten heel tailored for executive poise."},
		{10, "Biji Kopi Arabika Ijen Raung Banyuwangi Asam Lembut 250g", "ID-COF-020", 88000.0, "IDR", 45, "https://images.unsplash.com/photo-1514432324607-a09d9b4aefdd?w=600", "Populer", 4.8, "Kopi arabika perkebunan Ijen Jawa Timur dengan profil keasaman lembut rasa apel hijau dan karamel."},
		{10, "Guji Highlands Natural Process Sundried Coffee Beans 12oz", "EN-COF-020", 5.75, "USD", 45, "https://images.unsplash.com/photo-1514432324607-a09d9b4aefdd?w=600", "New Arrival", 4.8, "Raised-bed sun-dried Ethiopian specialty beans packed with wild blueberry, strawberry jam, and lavender."},
		{11, "Kerupuk Kulit Ikan Patin Renyah Bumbu Telur Asin 100g", "ID-SNK-020", 49000.0, "IDR", 65, "https://images.unsplash.com/photo-1599599810769-bcde5a160d32?w=600", "Sedang Tren", 4.8, "Kulit ikan patin krispi berbalut bumbu salted egg dan daun kari wangi tanpa bau amis."},
		{11, "Organic Roasted Edamame Beans High Plant Protein 8oz", "EN-SNK-020", 2.44, "USD", 80, "https://images.unsplash.com/photo-1599599810769-bcde5a160d32?w=600", "New Arrival", 4.8, "Dry-roasted green soybean poppers packing 14g of plant protein and 6g fiber per serving."},
		{12, "Tabir Surya Fisik Mineral Sunscreen Aman Kulit Sensitif", "ID-SKN-020", 159000.0, "IDR", 35, "https://images.unsplash.com/photo-1556228720-195a672e8a03?w=600", "Produk Unggulan", 4.9, "Sunscreen 100% zinc oxide non-nano aman untuk ibu hamil, menyusui, dan kulit rentan iritasi."},
		{12, "100% Mineral Sheer Tinted Sunscreen SPF 45 Broad Spectrum", "EN-SKN-020", 9.94, "USD", 35, "https://images.unsplash.com/photo-1556228720-195a672e8a03?w=600", "Premium", 4.9, "Reef-safe non-nano zinc sunscreen with universal sheer tint neutralizing redness and blue light."},
		{13, "Spons Mandi Serat Alami Loofah Pengangkat Daki Halus", "ID-BDY-020", 29000.0, "IDR", 110, "https://images.unsplash.com/photo-1592945403244-b3fbafd7f539?w=600", "Hemat", 4.7, "Spons mandi tanaman gambas oyong alami mengeksfoliasi daki kulit secara alami dan higienis."},
		{13, "Biodegradable Natural Egyptian Loofah Shower Sponge", "EN-BDY-020", 1.81, "USD", 110, "https://images.unsplash.com/photo-1592945403244-b3fbafd7f539?w=600", "Budget Pick", 4.7, "Organic unbleached plant loofah scrubber stimulating microcirculation and smoothing skin."},
	}

	totalCreated := 0
	for _, def := range allProductsList {
		subCat := subCategories[def.SubCatIdx]
		catID := subCat.CategoryID
		subCatID := subCat.ID

		p := product.Product{
			CategoryID:        catID,
			SubCategoryID:     &subCatID,
			Name:              def.Name,
			Slug:              security.Slugify(def.Name),
			SKU:               def.SKU,
			Description:       def.Description,
			Price:             def.Price,
			Currency:          def.Currency,
			StockQuantity:     def.Stock,
			LowStockThreshold: 5,
			ImageURL:          def.ImageURL,
			IsActive:          true,
			Badge:             def.Badge,
			Rating:            def.Rating,
		}
		if err := db.Create(&p).Error; err != nil {
			log.Printf("Failed to seed product %s: %v", def.Name, err)
		} else {
			totalCreated++
		}
	}

	log.Printf("✅ Seeding completed! Total %d retail products across 5 categories and 14 subcategories created.\n", totalCreated)

	var ollamaClient *ollama.Client
	if len(ollamaClients) > 0 {
		ollamaClient = ollamaClients[0]
	}

	if ollamaClient != nil {
		log.Println("🧠 Computing 384-dimensional pgvector embeddings for all 560 products via Ollama...")
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()

		var prods []product.Product
		db.Find(&prods)

		// Concurrent worker pool with 8 goroutines
		numWorkers := 8
		jobs := make(chan *product.Product, len(prods))
		var wg sync.WaitGroup

		for w := 0; w < numWorkers; w++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for prod := range jobs {
					if len(prod.Embedding.Slice()) > 0 {
						continue
					}
					text := fmt.Sprintf("%s. %s", prod.Name, prod.Description)
					vec, err := ollamaClient.GenerateEmbedding(ctx, text)
					if err == nil && len(vec) > 0 {
						pgVec := pgvector.NewVector(vec)
						if err := db.Exec("UPDATE products SET embedding = ? WHERE id = ?", pgVec, prod.ID).Error; err != nil {
							log.Printf("⚠️ Failed to update embedding for product %d: %v", prod.ID, err)
						}
					}
				}
			}()
		}

		for i := range prods {
			jobs <- &prods[i]
		}
		close(jobs)
		wg.Wait()
		log.Println("✅ Vector embeddings successfully generated and populated for all products in PostgreSQL pgvector!")
	}

	// 5. Seed Demo Orders for Sales Velocity and Analytics KPIs
	now := time.Now()
	var seededProducts []product.Product
	db.Limit(10).Find(&seededProducts)

	if len(seededProducts) >= 4 {
		order1 := order.Order{
			OrderNumber:     fmt.Sprintf("TRN-%d-0001", now.Unix()%10000),
			UserID:          shopper.ID,
			TotalAmount:     seededProducts[0].Price + seededProducts[1].Price*2,
			Currency:        "IDR",
			Status:          order.StatusCompleted,
			ShippingName:    shopper.Name,
			ShippingPhone:   shopper.Phone,
			ShippingAddress: shopper.Address,
			PaymentMethod:   "QRIS",
			PaymentStatus:   "PAID",
			CreatedAt:       now.AddDate(0, 0, -2),
		}
		db.Create(&order1)

		db.Create(&order.OrderItem{
			OrderID:      order1.ID,
			ProductID:    seededProducts[0].ID,
			ProductName:  seededProducts[0].Name,
			ProductSKU:   seededProducts[0].SKU,
			ProductImage: seededProducts[0].ImageURL,
			Quantity:     1,
			UnitPrice:    seededProducts[0].Price,
			Subtotal:     seededProducts[0].Price,
			Currency:     seededProducts[0].Currency,
		})
		db.Create(&order.OrderItem{
			OrderID:      order1.ID,
			ProductID:    seededProducts[1].ID,
			ProductName:  seededProducts[1].Name,
			ProductSKU:   seededProducts[1].SKU,
			ProductImage: seededProducts[1].ImageURL,
			Quantity:     2,
			UnitPrice:    seededProducts[1].Price,
			Subtotal:     seededProducts[1].Price * 2,
			Currency:     seededProducts[1].Currency,
		})

		order2 := order.Order{
			OrderNumber:     fmt.Sprintf("TRN-%d-0002", (now.Unix()+1)%10000),
			UserID:          sarah.ID,
			TotalAmount:     seededProducts[2].Price + seededProducts[3].Price,
			Currency:        "IDR",
			Status:          order.StatusProcessing,
			ShippingName:    sarah.Name,
			ShippingPhone:   sarah.Phone,
			ShippingAddress: sarah.Address,
			PaymentMethod:   "CARD",
			PaymentStatus:   "PAID",
			CreatedAt:       now.AddDate(0, 0, -1),
		}
		db.Create(&order2)

		db.Create(&order.OrderItem{
			OrderID:      order2.ID,
			ProductID:    seededProducts[2].ID,
			ProductName:  seededProducts[2].Name,
			ProductSKU:   seededProducts[2].SKU,
			ProductImage: seededProducts[2].ImageURL,
			Quantity:     1,
			UnitPrice:    seededProducts[2].Price,
			Subtotal:     seededProducts[2].Price,
			Currency:     seededProducts[2].Currency,
		})
		db.Create(&order.OrderItem{
			OrderID:      order2.ID,
			ProductID:    seededProducts[3].ID,
			ProductName:  seededProducts[3].Name,
			ProductSKU:   seededProducts[3].SKU,
			ProductImage: seededProducts[3].ImageURL,
			Quantity:     1,
			UnitPrice:    seededProducts[3].Price,
			Subtotal:     seededProducts[3].Price,
			Currency:     seededProducts[3].Currency,
		})
	}

	return nil
}
