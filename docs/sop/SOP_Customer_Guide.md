# 🛍️ Tirenn Commerce — Standard Operating Procedure (SOP)
## Panduan Lengkap Pelanggan: Cara Belanja & Penggunaan Fitur Website

* **Kode Dokumen**: `SOP-CUST-001`
* **Versi**: `2.0 (Bilingual & Multi-Currency)`
* **Target Pengguna**: Pembeli / Tamu (Customer)
* **Tanggal Efektif**: `2026-08-28`

---

## BAB 1: Pengenalan & Fitur Utama Website

Tirenn Commerce adalah platform belanja retail modern generasi baru yang dilengkapi dengan asisten belanja cerdas (*Tirenn AI Shopper*), pencarian semantik vektor, konversi mata uang otomatis (*Multi-Currency IDR & USD*), dan dukungan multibahasa (*Bahasa Indonesia & English*).

### Fitur-Fitur Kunci:
1. **Multilingual (i18n)**: Beralih bahasa secara instan antara **Bahasa Indonesia (ID)** dan **English (EN)** melalui tombol toggle bahasa di Navbar.
2. **Multi-Currency Real-Time**: Mendukung tampilan harga dalam Rupiah (**IDR / Rp**) dan Dolar Amerika (**USD / $**) dengan konversi otomatis berbasis kurs terkini.
3. **AI Shopper Assistant**: Asisten belanja AI bertenaga Qwen 2.5 yang dapat merekomendasikan produk, memeriksa sisa stok, dan memasukkan barang langsung ke keranjang belanja.
4. **Infinite Scroll Pagination**: Menampilkan seluruh 560 produk retail secara otomatis saat menggulir halaman ke bawah.

---

## BAB 2: Prosedur Pencarian & Pemilihan Produk

Pengguna dapat menemukan produk yang diinginkan melalui 4 cara praktis:
1. **Pencarian Kata Kunci (*Search Bar*)**: Masukkan nama barang atau kata kunci (contoh: *"headphone bluetooth"*, *"kemeja batik"*, *"kopi arabika"*) pada kolom pencarian di Navbar.
2. **Navigasi Kategori & Subkategori**: Klik tab kategori utama (*Elektronik, Fashion Pria, Fashion Wanita, Makanan & Minuman, Kecantikan*) lalu pilih pill subkategori yang sesuai.
3. **Filter & Pengurutan (*Sort & Filter*)**:
   - Urutkan berdasarkan: Terbaru (*Newest*), Harga Terendah (*Price: Low to High*), Harga Tertinggi (*Price: High to Low*), atau Nama A-Z.
   - Aktifkan filter **"Hanya Stok Tersedia" (*In-Stock Only*)** untuk menyaring produk yang siap kirim.
4. **Detail Produk (*Product Detail Modal / PDP*)**: Klik kartu produk untuk membaca deskripsi lengkap, memeriksa nomor SKU, melihat sisa stok, dan menentukan jumlah unit yang ingin dibeli.

---

## BAB 3: Panduan Menggunakan AI Shopper Cerdas

Asisten belanja AI dapat dibuka kapan saja melalui tombol mengambang ungu **"🤖 Tanya AI Shopper"** di pojok kanan bawah layar.

### Skenario & Contoh Perintah:
| Kebutuhan Pembeli | Contoh Perintah Chat | Aksi Cerdas AI |
| :--- | :--- | :--- |
| **Rekomendasi Spesifik** | *"Carikan celana panjang pria warna gelap"* | AI memanggil tool pencarian, menyaring kontradiksi (membuang celana pendek), dan menampilkan rekomendasi celana panjang. |
| **Pengecekan Stok & Harga** | *"Apakah AuraSound ANC-700 headphone ada stok?"* | AI memeriksa database inventaris real-time dan menginformasikan sisa unit yang tersedia. |
| **Beli Langsung via Chat** | *"Tolong masukkan 2 pcs Biji Kopi Gayo ke keranjang"* | AI otomatis mengeksekusi `AddToCartTool`, memasukkan barang ke keranjang belanja, dan memicu notifikasi toast. |

---

## BAB 4: Keranjang Belanja, Checkout & Pembayaran

Langkah-langkah menyelesaikan pesanan (*Order Checkout Workflow*):

1. **Membuka Keranjang Belanja (*Cart Drawer*)**: Klik ikon keranjang di Navbar atau klik tombol *"+ Beli Sekarang"* pada produk.
2. **Menyesuaikan Kuantitas**: Gunakan tombol **(+)** dan **(-)** untuk menambah/mengurangi jumlah barang. Total biaya akan dihitung ulang secara otomatis.
3. **Menuju Form Pembayaran**: Klik tombol **"Lanjut ke Pembayaran" (*Checkout*)**.
4. **Mengisi Informasi Pengiriman**:
   - **Nama Lengkap Penerima**
   - **Nomor Telepon / WhatsApp**
   - **Alamat Lengkap Pengiriman**
   - **Catatan Pengiriman (Opsional)**: Contoh: *"Titipkan ke satpam jika rumah kosong"*.
5. **Memilih Metode Pembayaran**:
   - **Debit / Credit Card (Instant Simulation)**: Simulasi pembayaran kartu instan.
   - **QRIS / GoPay / OVO**: Pembayaran melalui scan QRIS nasional.
   - **Bank Virtual Account**: Pembayaran transfer bank otomatis.
6. **Menyelesaikan Pesanan**: Klik tombol **"Bayar Sekarang"**. Sistem akan membuat Nomor Order unik (contoh: `#TRN-2026-0001`) dengan status pembayaran otomatis terverifikasi (**PAID**).

---

## BAB 5: Pelacakan Pesanan & Kebijakan Garansi

1. **Melihat Riwayat Belanja (*My Orders*)**: Klik menu **"Riwayat Belanja"** di Navbar untuk memantau status pesanan, rincian produk yang dibeli, nomor resi, dan total biaya.
2. **Arti Status Pesanan**:
   - `PAID / PROCESSING`: Pembayaran berhasil diterima; pesanan sedang disiapkan di gudang.
   - `SHIPPED`: Paket telah diserahkan ke jasa kurir logistik dan dalam perjalanan pengiriman.
   - `COMPLETED`: Pesanan telah berhasil diterima dengan baik oleh pembeli.
3. **Kebijakan Garansi & Retur Barang**: Pembeli berhak mengajukan retur atau penukaran barang dalam waktu maksimal 7x24 jam sejak barang diterima apabila ditemukan cacat produksi atau ketidaksesuaian barang, dengan menyertakan bukti rekaman video unboxing resmi.
