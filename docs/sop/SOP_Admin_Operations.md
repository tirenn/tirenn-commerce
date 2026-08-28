# 🛡️ Tirenn Commerce — Standard Operating Procedure (SOP)
## Panduan Operasional Admin: Pengelolaan Toko, Inventaris & Pemrosesan Pesanan

* **Kode Dokumen**: `SOP-ADM-002`
* **Versi**: `2.0 (Enterprise Operations)`
* **Otoritas**: Admin Toko & Operasional Merchant (Store Admin)
* **Tanggal Efektif**: `2026-08-28`

---

## BAB 1: Akses Admin Panel & Ringkasan Dashboard

Admin Panel Tirenn Commerce digunakan oleh pengelola toko untuk mengawasi operasional bisnis, mengelola katalog produk, mengatur inventaris gudang, dan memproses pesanan pembeli.

### 1. Autentikasi Admin:
- Masuk menggunakan akun terotorisasi (*Role: ADMIN*).
- Sistem akan otomatis menampilkan navigasi Dashboard Khusus Merchant di sisi atas.

### 2. Metrik Utama Dashboard KPI:
- **Total Revenue**: Akumulasi total nilai transaksi penjualan toko dalam Rupiah (Rp).
- **Total Orders**: Jumlah transaksi pesanan yang telah tercatat di sistem.
- **Total Customers**: Jumlah pembeli dan pelanggan terdaftar.
- **Active Products**: Jumlah total SKU katalog produk yang aktif diperjualbelikan.

---

## BAB 2: Standar Operasional Pemrosesan Pesanan (*Order Fulfillment SOP*)

Setiap pesanan yang masuk wajib dieksekusi secara terstruktur melalui 4 tahapan operasional:

| Tahap | Aktivitas Operasional | Status Sistem | Batas SLA Waktu |
| :--- | :--- | :--- | :--- |
| **Tahap 1: Verifikasi Pesanan** | Buka tab **Customer Orders**. Periksa rincian data pembeli (nama, nomor kontak, alamat pengiriman) dan pastikan pembayaran telah lunas. | `PAID / PROCESSING` | Maksimal 30 Menit |
| **Tahap 2: Pengambilan & Pengepakan (*Picking & Packing*)** | Ambil produk dari rak gudang sesuai SKU dan kuantitas order. Lakukan pengecekan fisik (*Quality Control*) dan kemas aman menggunakan bubble wrap & kardus tebal. | `PROCESSING` | Maksimal 2 Jam |
| **Tahap 3: Pengiriman ke Kurir** | Tempel label alamat dan nomor resi pengiriman. Serahkan barang ke pihak kurir ekspedisi logistik dan perbarui status pesanan menjadi Dikirim. | `SHIPPED` | Maksimal 1x24 Jam |
| **Tahap 4: Konfirmasi Penerimaan** | Pantau status pelacakan ekspedisi. Saat paket telah sukses diterima oleh pelanggan, tandai pesanan sebagai Selesai. | `COMPLETED` | 2–3 Hari Kerja |

---

## BAB 3: Manajemen Inventaris & Penyesuaian Stok Gudang

Admin bertanggung jawab penuh menjaga sinkronisasi antara stok fisik di gudang dan stok sistem.

### 1. Peringatan Stok Menipis (*Low Stock Alert*):
Produk dengan sisa stok &le; 5 unit akan otomatis ditandai dengan badge peringatan kuning **`LOW`** pada tabel produk.

### 2. Prosedur Penyesuaian Stok (*Stock Adjustment Modal*):
1. Masuk ke menu **Product Management**.
2. Klik tombol **"Adjust Stock"** pada baris produk yang ingin diubah.
3. Pilih salah satu mode penyesuaian:
   - **Tambah (+)**: Digunakan saat penerimaan barang baru / restock dari supplier.
   - **Kurang (-)**: Digunakan jika ditemukan barang rusak fisik, cacat produksi, atau kedaluwarsa.
   - **Set Nilai Baru**: Digunakan saat pelaksanaan *Stock Opname* fisik berkala.
4. Masukkan **Alasan Penyesuaian (*Adjustment Reason*)** wajib (contoh: *"Penerimaan pasokan restock batch #88"* atau *"Barang cacat unboxing"*).
5. Klik **"Save Stock"**. Sistem akan otomatis menyimpan log perubahan ke tabel audit `stock_adjustment_logs`.

---

## BAB 4: Manajemen Produk & Pembaruan Katalog

1. **Menambah Produk Baru**:
   - Klik tombol **"+ Add Product"**.
   - Isi field: Nama Produk, Kategori Utama, Subkategori, Nomor SKU unik, Harga Satuan (IDR/USD), Stok Awal, URL Gambar Produk, dan Deskripsi lengkap.
2. **Mengedit Produk**:
   - Klik tombol **"Edit"** pada tabel produk untuk mengubah harga, deskripsi, atau mengganti gambar barang.
3. **Menonaktifkan / Menghapus Produk**:
   - Produk yang sudah tidak diproduksi dapat diubah statusnya menjadi `Inactive` atau dihapus permanen melalui tombol **"Delete"**.

---

## BAB 5: Sinkronisasi AI Vector Embeddings

Setelah Admin melakukan penambahan atau pembaruan nama/deskripsi produk secara massal, jalankan sinkronisasi vektor AI agar AI Shopper dan Semantic Search langsung mengenali informasi produk terbaru:

- **Endpoint API**: `POST http://localhost:8000/api/v1/sync-from-backend`
- **Header Autentikasi**: `X-Internal-API-Key: tirenn-ai-internal-key-2026`
- Sistem akan secara otomatis menghitung vektor dense embedding (384 dimensi) dan memperbarui indeks `pgvector` di PostgreSQL.
