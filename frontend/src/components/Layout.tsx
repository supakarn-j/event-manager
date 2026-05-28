import type { ReactNode } from 'react'
import Nav from './Nav'

type LayoutProps = {
  children: ReactNode
  topbar?: ReactNode
  showAddStream?: boolean
  onAddStream?: () => void
}

export default function Layout({ children, topbar, showAddStream = false, onAddStream }: LayoutProps) {
  return (
    <div className="app-page">
      <div className="app-shell">
        <Nav showAddStream={showAddStream} onAddStream={onAddStream} />

        <main className="app-main">
          {topbar}
          <div className="app-content">{children}</div>
        </main>
      </div>
    </div>
  )
}
