'use client'

import { useEffect } from 'react'
import { useRouter } from 'next/navigation'

export default function AgentsPage() {
  const router = useRouter()
  useEffect(() => {
    router.replace('/?panel=agents')
  }, [router])
  return (
    <div className="flex min-h-[50vh] items-center justify-center bg-gray-50 text-sm text-gray-600">
      Opening agents…
    </div>
  )
}
