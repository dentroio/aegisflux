'use client'

import { Suspense, useEffect } from 'react'
import { useRouter, useSearchParams } from 'next/navigation'

function InventoryRedirectInner() {
  const router = useRouter()
  const searchParams = useSearchParams()
  const device = (searchParams.get('device') || '').trim()

  useEffect(() => {
    const target = device
      ? `/?panel=inventory&device=${encodeURIComponent(device)}`
      : '/?panel=inventory'
    router.replace(target)
  }, [router, device])

  return (
    <div className="flex min-h-[50vh] items-center justify-center bg-gray-50 text-sm text-gray-600">
      Opening inventory…
    </div>
  )
}

export default function InventoryPage() {
  return (
    <Suspense
      fallback={
        <div className="flex min-h-screen items-center justify-center bg-gray-50 text-sm text-gray-600">
          Loading…
        </div>
      }
    >
      <InventoryRedirectInner />
    </Suspense>
  )
}
