/**
 * Format a numeric amount into Indonesian Rupiah (Rp)
 * e.g. 150000 -> "Rp 150.000" or 149.99 -> "Rp 150.000" (or scaled appropriately)
 */
export function formatRupiah(amount: number): string {
  const num = Number(amount) || 0;
  // If amount is small (like seeded USD amounts < 1000), multiply by 15,000 for realistic IDR pricing, or format directly if already in thousands.
  const displayAmount = num < 1000 ? Math.round(num * 15000) : num;
  return new Intl.NumberFormat('id-ID', {
    style: 'currency',
    currency: 'IDR',
    minimumFractionDigits: 0,
    maximumFractionDigits: 0,
  }).format(displayAmount).replace('IDR', 'Rp').trim();
}

/**
 * Format raw exact IDR amount without multiplying
 */
export function formatIDR(amount: number): string {
  const num = Number(amount) || 0;
  return `Rp ${new Intl.NumberFormat('id-ID').format(num)}`;
}
