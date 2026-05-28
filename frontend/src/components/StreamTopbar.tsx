type StreamTopbarProps = {
  streamName: string
  onSendEvent: () => void
}

export default function StreamTopbar({ streamName, onSendEvent }: StreamTopbarProps) {
  return (
    <header className="app-topbar">
      <div>
        <div className="app-crumb">
          <a href="/home">Streams</a> / {streamName}
        </div>
        <h1 className="app-title">{streamName}</h1>
      </div>
      <div className="app-actions">
        <button className="btn btn-primary" type="button" onClick={onSendEvent}>
          Send event
        </button>
      </div>
    </header>
  )
}
