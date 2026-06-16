export type StreamConsumer = {
  name: string
  ip?: string
  lastSeen: string
  healthy: boolean
  pending: number
}

export type StreamConsumerGroup = {
  name: string
  pending: number
  consumers: StreamConsumer[]
}

type ConsumerGroupsProps = {
  streamName: string
  groups: StreamConsumerGroup[]
  onDeleteConsumer: (group: string, consumer: string) => void
}

export default function ConsumerGroups({ streamName, groups, onDeleteConsumer }: ConsumerGroupsProps) {
  return (
    <section className="group-stack" id="consumer-groups">
      {groups.length === 0 && (
        <section className="app-panel">
          <div className="panel-head">
            <h2 className="panel-title">Consumers</h2>
            <span className="panel-meta">No groups</span>
          </div>
          <div className="empty-state">No consumer groups registered</div>
        </section>
      )}

      {groups.map((group) => (
        <div className="app-panel" id={`card-group-${group.name}`} key={group.name}>
          <div className="panel-head">
            <h2 className="panel-title">Consumers</h2>
            <span className="panel-meta">
              {group.name} · PEL: {group.pending}
            </span>
          </div>
          <div id={`group-${group.name}`}>
            {group.consumers.length === 0 && (
              <div className="empty-state" id="no-consumer">
                No consumers registered
              </div>
            )}

            {group.consumers.map((consumer) => (
              <div className="consumer-row" id={`consumer-${group.name}-${consumer.name}`} key={consumer.name}>
                <div className="consumer-summary">
                  <div className="consumer-title-row">
                    <span className={`status-pill consumer-badge ${consumer.healthy ? '' : 'unknown'}`}>
                      {consumer.healthy ? 'Healthy' : 'Unknown'}
                    </span>
                    <span className="consumer-name truncate">{consumer.name}</span>
                    <span className="panel-meta consumer-pel">PEL: {consumer.pending}</span>
                  </div>
                  <div className="panel-meta consumer-meta-row">
                    <span className="consumer-meta-item consumer-last-seen">Last seen: {consumer.lastSeen || '-'}</span>
                    <span className="consumer-meta-item">IP: {consumer.ip || '-'}</span>
                  </div>
                </div>
                <div className="d-flex align-items-center gap-2">
                  <button
                    className="btn btn-sm btn-outline-danger"
                    type="button"
                    title="Delete consumer"
                    aria-label={`Delete consumer ${consumer.name} from ${streamName}`}
                    onClick={() => onDeleteConsumer(group.name, consumer.name)}
                  >
                    <i className="fa-solid fa-trash"></i>
                  </button>
                </div>
              </div>
            ))}
          </div>
        </div>
      ))}
    </section>
  )
}
