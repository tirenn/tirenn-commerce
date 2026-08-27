/**
 * Format a numeric amount into Indonesian Rupiah (Rp)
 * e.g. 1499000 -> "Rp 1.499.000"
 */
export function formatRupiah(amount: number): string {
  const num = Number(amount) || 0;
  return new Intl.NumberFormat('id-ID', {
    style: 'currency',
    currency: 'IDR',
    minimumFractionDigits: 0,
    maximumFractionDigits: 0,
  }).format(num).replace('IDR', 'Rp').trim();
}

/**
 * Format raw exact IDR amount without currency symbol
 */
export function formatIDR(amount: number): string {
  const num = Number(amount) || 0;
  return `Rp ${new Intl.NumberFormat('id-ID').format(num)}`;
}
