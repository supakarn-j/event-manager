type NavProps = {
  showAddStream?: boolean
  onAddStream?: () => void
}

export default function Nav({ showAddStream = false, onAddStream }: NavProps) {
  return (
    <header className="app-header">
      <a className="app-brand" href="/home">
        <span className="app-brand-mark">E</span>
        <span>Event Manager</span>
      </a>
      {showAddStream && (
        <button className="btn btn-primary" type="button" onClick={onAddStream}>
          Add stream
        </button>
      )}
    </header>
  )
}
