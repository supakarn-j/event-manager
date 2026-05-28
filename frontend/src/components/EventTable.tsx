export type StreamAckStatus = {
  label: string
  state: string
  tooltip: string
}

export type StreamEvent = {
  timestamp: string
  id: string
  values: Record<string, string>
  ack: StreamAckStatus
}

type EventTableProps = {
  streamName: string
  headers: string[]
  events: StreamEvent[]
  loading: boolean
  onRefresh: () => void
  onDeleteEvent: (eventID: string) => void
}

export default function EventTable({
  streamName,
  headers,
  events,
  loading,
  onRefresh,
  onDeleteEvent,
}: EventTableProps) {
  const colSpan = Math.max(4, headers.length + 4)

  return (
    <section className="app-panel">
      <div className="panel-head">
        <div className="panel-title-group">
          <h2 className="panel-title">Events</h2>
          <button
            className="btn btn-sm btn-outline-secondary"
            type="button"
            title="Refresh events"
            aria-label={`Refresh events for ${streamName}`}
            onClick={onRefresh}
          >
            <i className="fa-solid fa-rotate-right"></i>
          </button>
        </div>
        <div className="panel-meta">
          <span id="event-num">{events.length}</span>
          <span> event(s)</span>
        </div>
      </div>

      <div className="event-table-wrap">
        <table className="app-table">
          <thead>
            <tr>
              <th style={{ width: 210 }}>Timestamp</th>
              <th style={{ width: 180 }}>ID</th>
              <th style={{ width: 130 }}>Ack</th>
              {headers.map((header) => (
                <th key={header}>{header}</th>
              ))}
              <th style={{ width: 90 }}></th>
            </tr>
          </thead>
          <tbody>
            {loading && (
              <tr>
                <td colSpan={colSpan} className="empty-state">
                  Loading events...
                </td>
              </tr>
            )}

            {!loading && events.length === 0 && (
              <tr>
                <td colSpan={colSpan} className="empty-state">
                  No events found
                </td>
              </tr>
            )}

            {!loading &&
              events.map((event) => (
                <tr key={event.id}>
                  <td className="ts muted">{event.timestamp}</td>
                  <td className="id">
                    <code>{event.id}</code>
                  </td>
                  <td className="ack-cell">
                    <span className={`ack-pill ${event.ack.state}`} tabIndex={0} title={event.ack.tooltip}>
                      {event.ack.label}
                    </span>
                  </td>
                  {headers.map((header) => (
                    <td className="values" key={header}>
                      <pre className="json">{event.values[header] || ''}</pre>
                    </td>
                  ))}
                  <td>
                    <div className="event-row-actions">
                      <button
                        className="btn btn-sm btn-outline-danger"
                        type="button"
                        title="Delete event"
                        aria-label={`Delete event ${event.id}`}
                        onClick={() => onDeleteEvent(event.id)}
                      >
                        <i className="fa-solid fa-trash"></i>
                      </button>
                    </div>
                  </td>
                </tr>
              ))}
          </tbody>
        </table>
      </div>
    </section>
  )
}
