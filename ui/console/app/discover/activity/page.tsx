'use client'

import { RoutedShellHero } from '@/components/shell/RoutedShellHero'

export default function Page() {
  return (
    <RoutedShellHero
      activeNavId="activity"
      title="AI Activity"
      subtitle="Fleet-wide summaries of modeled AI tooling will consolidate findings, DNS hints, and process evidence."
    />
  )
}
