import type { Metadata } from 'next'
import './globals.css'

export const metadata: Metadata = {
  title: 'AegisFlux Console',
  description: 'Agent Management and Network Security Policy Builder',
}

export default function RootLayout({
  children,
}: {
  children: React.ReactNode
}) {
  return (
    <html lang="en">
      <body>
        <div className="min-h-screen bg-gray-50">
          {children}
        </div>
      </body>
    </html>
  )
}
