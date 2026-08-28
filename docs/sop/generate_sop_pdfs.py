import os
import sys
from reportlab.lib.pagesizes import letter
from reportlab.lib import colors
from reportlab.lib.styles import getSampleStyleSheet, ParagraphStyle
from reportlab.lib.units import inch
from reportlab.platypus import (
    SimpleDocTemplate, Paragraph, Spacer, Table, TableStyle, PageBreak, KeepTogether, HRFlowable
)
from reportlab.pdfgen import canvas

# Destination directory
DOCS_DIR = r"c:\Users\Ryzen\Documents\Projects\ai-commerce\docs\sop"
os.makedirs(DOCS_DIR, exist_ok=True)

class NumberedCanvas(canvas.Canvas):
    def __init__(self, *args, **kwargs):
        super().__init__(*args, **kwargs)
        self._saved_page_states = []

    def showPage(self):
        self._saved_page_states.append(dict(self.__dict__))
        self._startPage()

    def save(self):
        num_pages = len(self._saved_page_states)
        for state in self._saved_page_states:
            self.__dict__.update(state)
            self.draw_page_decorations(num_pages)
            super().showPage()
        super().save()

    def draw_page_decorations(self, page_count):
        self.saveState()
        self.setFont("Helvetica", 9)
        self.setFillColor(colors.HexColor("#64748B"))
        
        # Header (pages > 1)
        if self._pageNumber > 1:
            self.drawString(54, 750, "Tirenn Commerce — Standard Operating Procedure (SOP)")
            self.setStrokeColor(colors.HexColor("#CBD5E1"))
            self.setLineWidth(0.5)
            self.line(54, 742, 558, 742)
            
        # Footer
        self.setStrokeColor(colors.HexColor("#CBD5E1"))
        self.setLineWidth(0.5)
        self.line(54, 50, 558, 50)
        
        page_str = f"Halaman {self._pageNumber} dari {page_count}"
        self.drawRightString(558, 38, page_str)
        self.drawString(54, 38, "Dokumen Resmi Operasional Tirenn Commerce • Rahasia & Terkendali")
        self.restoreState()


def create_customer_sop_pdf():
    pdf_path = os.path.join(DOCS_DIR, "SOP_Customer_Guide.pdf")
    doc = SimpleDocTemplate(
        pdf_path,
        pagesize=letter,
        leftMargin=54,
        rightMargin=54,
        topMargin=54,
        bottomMargin=54
    )

    styles = getSampleStyleSheet()
    
    title_style = ParagraphStyle(
        'DocTitle',
        parent=styles['Normal'],
        fontName='Helvetica-Bold',
        fontSize=24,
        leading=28,
        textColor=colors.HexColor("#1E293B"),
        alignment=1, # Center
        spaceAfter=15
    )

    subtitle_style = ParagraphStyle(
        'DocSubtitle',
        parent=styles['Normal'],
        fontName='Helvetica-Bold',
        fontSize=12,
        leading=16,
        textColor=colors.HexColor("#2563EB"),
        alignment=1,
        spaceAfter=25
    )

    h1_style = ParagraphStyle(
        'Heading1_Custom',
        parent=styles['Normal'],
        fontName='Helvetica-Bold',
        fontSize=14,
        leading=18,
        textColor=colors.HexColor("#0F172A"),
        spaceBefore=14,
        spaceAfter=8,
        keepWithNext=True
    )

    h2_style = ParagraphStyle(
        'Heading2_Custom',
        parent=styles['Normal'],
        fontName='Helvetica-Bold',
        fontSize=11,
        leading=15,
        textColor=colors.HexColor("#1E40AF"),
        spaceBefore=10,
        spaceAfter=4,
        keepWithNext=True
    )

    body_style = ParagraphStyle(
        'Body_Custom',
        parent=styles['Normal'],
        fontName='Helvetica',
        fontSize=9.5,
        leading=14,
        textColor=colors.HexColor("#334155"),
        spaceAfter=6
    )

    callout_style = ParagraphStyle(
        'Callout_Text',
        parent=styles['Normal'],
        fontName='Helvetica-Oblique',
        fontSize=9,
        leading=13,
        textColor=colors.HexColor("#1E3A8A")
    )

    elements = []

    # Title Box / Banner
    elements.append(Spacer(1, 10))
    elements.append(Paragraph("TIRENN COMMERCE", subtitle_style))
    elements.append(Paragraph("STANDAR OPERASIONAL PROSEDUR (SOP)<br/>PANDUAN BELANJA & PENGGUNAAN FITUR", title_style))
    elements.append(Paragraph("Dokumen Panduan Lengkap untuk Pelanggan (Customer User Manual)", ParagraphStyle('center_desc', parent=body_style, alignment=1, textColor=colors.HexColor("#64748B"))))
    elements.append(Spacer(1, 15))
    elements.append(HRFlowable(width="100%", thickness=1.5, color=colors.HexColor("#2563EB"), spaceAfter=20))

    # Meta Table
    meta_data = [
        [Paragraph("<b>Kode Dokumen:</b> SOP-CUST-001", body_style), Paragraph("<b>Versi:</b> 2.0 (Bilingual & Multi-Currency)", body_style)],
        [Paragraph("<b>Target Pengguna:</b> Pembeli / Tamu (Customer)", body_style), Paragraph("<b>Tanggal Efektif:</b> 2026-08-28", body_style)],
    ]
    t_meta = Table(meta_data, colWidths=[250, 250])
    t_meta.setStyle(TableStyle([
        ('BACKGROUND', (0,0), (-1,-1), colors.HexColor("#F8FAFC")),
        ('BOX', (0,0), (-1,-1), 0.5, colors.HexColor("#E2E8F0")),
        ('VALIGN', (0,0), (-1,-1), 'MIDDLE'),
        ('PADDING', (0,0), (-1,-1), 8),
    ]))
    elements.append(t_meta)
    elements.append(Spacer(1, 15))

    # BAB 1
    elements.append(Paragraph("BAB 1: Pengenalan & Fitur Utama Website", h1_style))
    elements.append(Paragraph(
        "Tirenn Commerce adalah platform belanja online cerdas generasi baru yang dilengkapi dengan pencarian semantik AI, asisten belanja percakapan cerdas (AI Shopper Qwen 2.5), konversi mata uang otomatis (IDR & USD), dan dukungan multibahasa (Bahasa Indonesia & English).",
        body_style
    ))
    elements.append(Paragraph("<b>Fitur-Fitur Kunci:</b>", body_style))
    elements.append(Paragraph("• <b>Multilingual (i18n):</b> Beralih secara instan antara Bahasa Indonesia (ID) dan English (EN) melalui tombol bahasa di Navbar.", body_style))
    elements.append(Paragraph("• <b>Multi-Currency Real-Time:</b> Mendukung mata uang Rupiah (Rp / IDR) dan Dolar AS ($ / USD) dengan auto-conversion kurs langsung.", body_style))
    elements.append(Paragraph("• <b>AI Shopper Assistant:</b> Asisten belanja AI interaktif yang dapat mencarikan produk, memeriksa ketersediaan stok, dan memasukkan barang ke keranjang.", body_style))
    elements.append(Paragraph("• <b>Infinite Scroll Pagination:</b> Produk dimuat secara otomatis saat menggulir halaman ke bawah (560 produk unik).", body_style))
    elements.append(Spacer(1, 10))

    # BAB 2
    elements.append(Paragraph("BAB 2: Prosedur Pencarian & Pemilihan Produk", h1_style))
    elements.append(Paragraph("Pengguna dapat menemukan produk yang diinginkan melalui beberapa metode:", body_style))
    elements.append(Paragraph("<b>1. Pencarian Real-Time (Search Bar):</b> Masukkan nama produk, kata kunci, atau tipe barang pada kolom pencarian di bagian atas Navbar.", body_style))
    elements.append(Paragraph("<b>2. Navigasi Kategori & Subkategori:</b> Klik tab kategori utama (Elektronik, Fashion Pria, Fashion Wanita, Makanan & Minuman, Kecantikan) lalu pilih pill subkategori yang sesuai.", body_style))
    elements.append(Paragraph("<b>3. Pengurutan & Filter:</b> Gunakan menu filter untuk mengurutkan produk (Terbaru, Harga Terendah, Harga Tertinggi, Nama A-Z) serta membatasi hanya barang yang memiliki stok (In-Stock Only).", body_style))
    elements.append(Paragraph("<b>4. Detail Produk (Modal PDP):</b> Klik kartu produk untuk membuka rincian spesifikasi lengkap, stok tersisa, SKU produk, dan memilih jumlah pembelian.", body_style))
    elements.append(Spacer(1, 10))

    # BAB 3
    elements.append(Paragraph("BAB 3: Panduan Menggunakan AI Shopper Cerdas", h1_style))
    elements.append(Paragraph(
        "Tirenn Commerce menyediakan asisten belanja cerdas bertenaga AI (Qwen 2.5 1.5B) yang dapat diakses melalui tombol mengambang <b>'🤖 Tanya AI Shopper'</b> di pojok kanan bawah.",
        body_style
    ))
    
    ai_guide_data = [
        [Paragraph("<b>Kebutuhan Pengguna</b>", body_style), Paragraph("<b>Contoh Perintah Chat AI</b>", body_style), Paragraph("<b>Aksi Otomatis AI</b>", body_style)],
        [
            Paragraph("Mencari rekomendasi spesifik", body_style),
            Paragraph("<i>'Carikan celana panjang pria untuk santai'</i>", body_style),
            Paragraph("AI menjalankan pencarian semantik, memfilter kontradiksi (membuang celana pendek), dan menampilkan kartu produk.", body_style)
        ],
        [
            Paragraph("Pengecekan stok & harga", body_style),
            Paragraph("<i>'Apakah AuraSound headphone masih ada stok?'</i>", body_style),
            Paragraph("AI mengecek database inventaris real-time dan menginformasikan sisa unit.", body_style)
        ],
        [
            Paragraph("Membeli via chat", body_style),
            Paragraph("<i>'Masukkan 2 pcs Biji Kopi Gayo ke keranjang'</i>", body_style),
            Paragraph("AI otomatis memasukkan barang ke keranjang belanja (berlaku untuk tamu & user login).", body_style)
        ]
    ]
    t_ai = Table(ai_guide_data, colWidths=[130, 170, 200])
    t_ai.setStyle(TableStyle([
        ('BACKGROUND', (0,0), (-1,0), colors.HexColor("#EFF6FF")),
        ('GRID', (0,0), (-1,-1), 0.5, colors.HexColor("#CBD5E1")),
        ('VALIGN', (0,0), (-1,-1), 'TOP'),
        ('PADDING', (0,0), (-1,-1), 6),
    ]))
    elements.append(t_ai)
    elements.append(Spacer(1, 15))

    # BAB 4
    elements.append(Paragraph("BAB 4: Keranjang Belanja, Checkout & Pembayaran", h1_style))
    elements.append(Paragraph("Langkah-langkah penyelesaian pembelian (*Order Checkout*):", body_style))
    elements.append(Paragraph("<b>Langkah 1:</b> Buka Cart Drawer melalui ikon keranjang di Navbar atau klik 'Beli Sekarang' pada produk.", body_style))
    elements.append(Paragraph("<b>Langkah 2:</b> Sesuaikan jumlah unit menggunakan tombol <b>(+)</b> dan <b>(-)</b>. Total biaya akan terkalkulasi otomatis.", body_style))
    elements.append(Paragraph("<b>Langkah 3:</b> Klik tombol <b>'Lanjut ke Pembayaran'</b> untuk membuka form Checkout.", body_style))
    elements.append(Paragraph("<b>Langkah 4:</b> Lengkapi data pengiriman: Nama Penerima, Nomor WhatsApp/Telepon, Alamat Lengkap, dan Catatan Kurir (opsional).", body_style))
    elements.append(Paragraph("<b>Langkah 5:</b> Pilih Metode Pembayaran:", body_style))
    elements.append(Paragraph("  • <i>Debit / Credit Card (Instant Simulation)</i>: Verifikasi kartu kredit/debit instan.", body_style))
    elements.append(Paragraph("  • <i>QRIS / GoPay / OVO</i>: Pembayaran instan via scan QRIS nasional.", body_style))
    elements.append(Paragraph("  • <i>Bank Virtual Account</i>: Pembayaran transfer bank otomatis.", body_style))
    elements.append(Paragraph("<b>Langkah 6:</b> Klik <b>'Bayar Sekarang'</b>. Sistem akan menerbitkan Nomor Order unik (contoh: <code>#TRN-2026-0001</code>) dan status pembayaran otomatis terverifikasi (PAID).", body_style))
    elements.append(Spacer(1, 10))

    # BAB 5
    elements.append(Paragraph("BAB 5: Pelacakan Pesanan & Kebijakan Garansi", h1_style))
    elements.append(Paragraph("<b>1. Melihat Riwayat Belanja:</b> Klik menu <b>'Riwayat Belanja' (My Orders)</b> di Navbar untuk melihat status pesanan terbaru, rincian barang yang dibeli, dan total pembayaran.", body_style))
    elements.append(Paragraph("<b>2. Status Pesanan:</b>", body_style))
    elements.append(Paragraph("  • <code>PAID / PROCESSING</code>: Pembayaran berhasil, pesanan sedang disiapkan tim logistik.", body_style))
    elements.append(Paragraph("  • <code>SHIPPED</code>: Paket telah diserahkan ke jasa kurir ekspedisi pengiriman.", body_style))
    elements.append(Paragraph("  • <code>COMPLETED</code>: Paket telah sukses diterima oleh pelanggan.", body_style))
    elements.append(Paragraph("<b>3. Kebijakan Garansi & Pengembalian:</b> Pelanggan berhak mengajukan pengembalian barang dalam waktu 7x24 jam sejak barang diterima jika terdapat cacat produksi atau ketidaksesuaian barang, dengan menyertakan video unboxing resmi.", body_style))

    doc.build(elements, canvasmaker=NumberedCanvas)
    print(f"[SUCCESS] Generated Customer SOP: {pdf_path}")


def create_admin_sop_pdf():
    pdf_path = os.path.join(DOCS_DIR, "SOP_Admin_Operations.pdf")
    doc = SimpleDocTemplate(
        pdf_path,
        pagesize=letter,
        leftMargin=54,
        rightMargin=54,
        topMargin=54,
        bottomMargin=54
    )

    styles = getSampleStyleSheet()
    
    title_style = ParagraphStyle(
        'DocTitle_Admin',
        parent=styles['Normal'],
        fontName='Helvetica-Bold',
        fontSize=22,
        leading=26,
        textColor=colors.HexColor("#0F172A"),
        alignment=1,
        spaceAfter=15
    )

    subtitle_style = ParagraphStyle(
        'DocSubtitle_Admin',
        parent=styles['Normal'],
        fontName='Helvetica-Bold',
        fontSize=12,
        leading=16,
        textColor=colors.HexColor("#7C3AED"), # Purple for admin
        alignment=1,
        spaceAfter=25
    )

    h1_style = ParagraphStyle(
        'Heading1_Admin',
        parent=styles['Normal'],
        fontName='Helvetica-Bold',
        fontSize=13,
        leading=17,
        textColor=colors.HexColor("#0F172A"),
        spaceBefore=14,
        spaceAfter=8,
        keepWithNext=True
    )

    body_style = ParagraphStyle(
        'Body_Admin',
        parent=styles['Normal'],
        fontName='Helvetica',
        fontSize=9.5,
        leading=14,
        textColor=colors.HexColor("#334155"),
        spaceAfter=6
    )

    elements = []

    # Title Box / Banner
    elements.append(Spacer(1, 10))
    elements.append(Paragraph("TIRENN COMMERCE — MERCHANT CONTROL PANEL", subtitle_style))
    elements.append(Paragraph("STANDAR OPERASIONAL PROSEDUR (SOP)<br/>OPERASIONAL ADMIN & PEMROSESAN PESANAN", title_style))
    elements.append(Paragraph("Petunjuk Resmi Pengelolaan Toko, Inventaris, dan Pemrosesan Order (*Merchant Operations SOP*)", ParagraphStyle('center_desc_admin', parent=body_style, alignment=1, textColor=colors.HexColor("#64748B"))))
    elements.append(Spacer(1, 15))
    elements.append(HRFlowable(width="100%", thickness=1.5, color=colors.HexColor("#7C3AED"), spaceAfter=20))

    # Meta Table
    meta_data = [
        [Paragraph("<b>Kode Dokumen:</b> SOP-ADM-002", body_style), Paragraph("<b>Versi:</b> 2.0 (Enterprise Operations)", body_style)],
        [Paragraph("<b>Otoritas:</b> Admin Toko & Operasional Merchant", body_style), Paragraph("<b>Tanggal Efektif:</b> 2026-08-28", body_style)],
    ]
    t_meta = Table(meta_data, colWidths=[250, 250])
    t_meta.setStyle(TableStyle([
        ('BACKGROUND', (0,0), (-1,-1), colors.HexColor("#FAF5FF")),
        ('BOX', (0,0), (-1,-1), 0.5, colors.HexColor("#E9D5FF")),
        ('VALIGN', (0,0), (-1,-1), 'MIDDLE'),
        ('PADDING', (0,0), (-1,-1), 8),
    ]))
    elements.append(t_meta)
    elements.append(Spacer(1, 15))

    # BAB 1
    elements.append(Paragraph("BAB 1: Akses Admin Panel & Dashboard KPI", h1_style))
    elements.append(Paragraph(
        "Admin Panel Tirenn Commerce digunakan oleh pengelola toko untuk memantau performa penjualan, mengatur inventaris gudang, mengelola katalog produk, dan mengeksekusi pesanan pelanggan.",
        body_style
    ))
    elements.append(Paragraph("<b>1. Autentikasi Admin:</b> Masuk menggunakan akun berwenang (*Role: ADMIN*). Sistem otomatis membuka navigasi Dashboard Khusus Merchant.", body_style))
    elements.append(Paragraph("<b>2. KPI Eksekutif Dashboard:</b>", body_style))
    elements.append(Paragraph("  • <b>Total Revenue:</b> Total omzet akumulasi penjualan toko dalam Rupiah (Rp).", body_style))
    elements.append(Paragraph("  • <b>Total Orders:</b> Jumlah transaksi yang berhasil tercatat di sistem.", body_style))
    elements.append(Paragraph("  • <b>Total Customers:</b> Basis data pelanggan aktif yang terdaftar.", body_style))
    elements.append(Paragraph("  • <b>Active Products:</b> Jumlah katalog SKU produk yang aktif diperjualbelikan.", body_style))
    elements.append(Spacer(1, 10))

    # BAB 2
    elements.append(Paragraph("BAB 2: Standar Operasional Pemrosesan Pesanan (*Order Fulfillment SOP*)", h1_style))
    elements.append(Paragraph("Setiap pesanan baru wajib diproses mengikuti 4 tahapan standar berikut:", body_style))
    
    order_sop_table = [
        [Paragraph("<b>Tahap</b>", body_style), Paragraph("<b>Aktivitas Operasional</b>", body_style), Paragraph("<b>Status Sistem</b>", body_style), Paragraph("<b>SLA Waktu</b>", body_style)],
        [
            Paragraph("<b>Tahap 1:<br/>Verifikasi</b>", body_style),
            Paragraph("Buka menu <b>Customer Orders</b>. Periksa kelengkapan nama, alamat, nomor telepon pembeli, serta verifikasi status pembayaran.", body_style),
            Paragraph("<code>PAID / PROCESSING</code>", body_style),
            Paragraph("Maks. 30 Menit", body_style)
        ],
        [
            Paragraph("<b>Tahap 2:<br/>Picking & Packing</b>", body_style),
            Paragraph("Ambil barang dari gudang sesuai SKU & kuantitas. Lakukan pengecekan fisik (*quality control*) dan kemas aman dengan bubble wrap.", body_style),
            Paragraph("<code>PROCESSING</code>", body_style),
            Paragraph("Maks. 2 Jam", body_style)
        ],
        [
            Paragraph("<b>Tahap 3:<br/>Pengiriman Kurir</b>", body_style),
            Paragraph("Tempel label resi pengiriman. Serahkan paket ke kurir logistik dan perbarui status pesanan menjadi Dikirim.", body_style),
            Paragraph("<code>SHIPPED</code>", body_style),
            Paragraph("Maks. 1x24 Jam", body_style)
        ],
        [
            Paragraph("<b>Tahap 4:<br/>Penyelesaian</b>", body_style),
            Paragraph("Pantau konfirmasi kurir saat paket tiba di alamat tujuan. Sistem akan menandai pesanan selesai.", body_style),
            Paragraph("<code>COMPLETED</code>", body_style),
            Paragraph("Otomatis / 2-3 Hari", body_style)
        ]
    ]
    t_order = Table(order_sop_table, colWidths=[80, 230, 110, 80])
    t_order.setStyle(TableStyle([
        ('BACKGROUND', (0,0), (-1,0), colors.HexColor("#F3E8FF")),
        ('GRID', (0,0), (-1,-1), 0.5, colors.HexColor("#D8B4FE")),
        ('VALIGN', (0,0), (-1,-1), 'TOP'),
        ('PADDING', (0,0), (-1,-1), 6),
    ]))
    elements.append(t_order)
    elements.append(Spacer(1, 15))

    # BAB 3
    elements.append(Paragraph("BAB 3: Manajemen Inventaris & Penyesuaian Stok Gudang", h1_style))
    elements.append(Paragraph("Admin bertanggung jawab menjaga akurasi stok fisik dan sistem:", body_style))
    elements.append(Paragraph("<b>1. Peringatan Low Stock (Stok Menipis):</b> Produk dengan sisa stok &le; 5 unit otomatis ditandai dengan badge kuning <code>LOW</code>.", body_style))
    elements.append(Paragraph("<b>2. Prosedur Stock Adjustment Modal:</b>", body_style))
    elements.append(Paragraph("  • Buka menu <b>Product Management</b>, lalu klik tombol <b>'Adjust Stock'</b> pada produk target.", body_style))
    elements.append(Paragraph("  • Pilih mode: <b>Tambah (+)</b> untuk barang masuk baru, <b>Kurang (-)</b> untuk barang rusak/rusak fisik, atau <b>Set Nilai</b> untuk stock opname fisik.", body_style))
    elements.append(Paragraph("  • Masukkan <b>Alasan Penyesuaian (*Reason*)</b> wajib (contoh: <i>'Penerimaan restock batch supplier #45'</i> atau <i>'Barang rusak saat unboxing'</i>).", body_style))
    elements.append(Paragraph("  • Klik <b>'Save Stock'</b>. Sistem secara otomatis mencatat riwayat ke tabel audit log <code>stock_adjustment_logs</code>.", body_style))
    elements.append(Spacer(1, 10))

    # BAB 4
    elements.append(Paragraph("BAB 4: Manajemen Produk & Pembaruan Katalog", h1_style))
    elements.append(Paragraph("<b>1. Menambah Produk Baru:</b> Klik tombol <b>'+ Add Product'</b>. Masukkan Nama Produk, Kategori, Subkategori, SKU unik, Harga (IDR/USD), Stok Awal, URL Gambar, dan Deskripsi lengkap.", body_style))
    elements.append(Paragraph("<b>2. Mengedit Produk:</b> Klik tombol <b>'Edit'</b> pada tabel untuk memperbarui harga jual, deskripsi, atau mengganti foto barang.", body_style))
    elements.append(Paragraph("<b>3. Menonaktifkan / Menghapus Produk:</b> Produk yang dihentikan penjualannya dapat diubah statusnya menjadi <code>Inactive</code> atau dihapus permanen melalui tombol <b>'Delete'</b>.", body_style))
    elements.append(Spacer(1, 10))

    # BAB 5
    elements.append(Paragraph("BAB 5: Sinkronisasi AI Embeddings & Vector Database", h1_style))
    elements.append(Paragraph(
        "Setelah Admin melakukan penambahan produk baru atau perubahan nama/deskripsi secara massal, Admin disarankan memicu sinkronisasi AI Vector Embeddings agar AI Shopper dan Semantic Search langsung mengenali produk terbaru.",
        body_style
    ))
    elements.append(Paragraph("<b>Endpoint Sinkronisasi AI:</b> <code>POST /api/v1/sync-from-backend</code> (dengan header <code>X-Internal-API-Key</code>).", body_style))
    elements.append(Paragraph("Sistem secara otomatis membuatkan dense vector embeddings (384-dimensi) dan menyimpannya ke indeks PostgreSQL pgvector.", body_style))

    doc.build(elements, canvasmaker=NumberedCanvas)
    print(f"[SUCCESS] Generated Admin SOP: {pdf_path}")

if __name__ == "__main__":
    create_customer_sop_pdf()
    create_admin_sop_pdf()
    print("[SUCCESS] Both SOP PDFs generated successfully!")
