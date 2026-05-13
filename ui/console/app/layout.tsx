import type { Metadata } from 'next'
import './globals.css'

export const metadata: Metadata = {
  title: 'AegisFlux Console',
  description: 'Agent Management and Network Security Policy Builder',
  icons: {
    icon: '/aegisflux-shield.png',
    shortcut: '/aegisflux-shield.png',
    apple: '/aegisflux-shield.png',
  },
}

export default function RootLayout({
  children,
}: {
  children: React.ReactNode
}) {
  return (
    <html lang="en">
      <body>
        <div className="min-h-screen">
          {children}
        </div>
      </body>
    </html>
  )
}
