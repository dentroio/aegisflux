import { redirect } from 'next/navigation'

export default function InventoryPage({
  searchParams,
}: {
  searchParams?: { device?: string }
}) {
  const device = (searchParams?.device || '').trim()
  const target = device
    ? `/?panel=inventory&device=${encodeURIComponent(device)}`
    : '/?panel=inventory'

  redirect(target)
}
