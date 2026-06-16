export type AckConsumer = {
    consumer?: string;
    Consumer?: string;
    group?: string;
    Group?: string;
    timestamp?: string;
    Timestamp?: string;
};

export type StreamEvent = {
    id: string;
    values: Record<string, unknown>;
    consumers?: AckConsumer[];
};

type EventTableProps = {
    streamName: string;
    headers: string[];
    events: StreamEvent[];
    loading: boolean;
    onRefresh: () => void;
    onDeleteEvent: (eventID: string) => void;
};

export default function EventTable({
    streamName,
    headers,
    events,
    loading,
    onRefresh,
    onDeleteEvent,
}: EventTableProps) {
    const colSpan = Math.max(2, headers.length + 2);

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
                            <th style={{ width: 250 }}>ID</th>
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
                                    <td className="id">
                                        <div className="event-id-cell">
                                            <code>{event.id}</code>
                                            {event.consumers &&
                                                event.consumers.length > 0 && (
                                                    <span
                                                        className="ack-badge"
                                                        tabIndex={0}
                                                        title={ackBadgeTitle(
                                                            event.consumers,
                                                        )}
                                                    >
                                                        {ackBadgeLabel(
                                                            event.consumers,
                                                        )}
                                                    </span>
                                                )}
                                        </div>
                                    </td>
                                    {headers.map((header) => (
                                        <td className="values" key={header}>
                                            <pre className="json">
                                                {String(
                                                    event.values[header] ?? "",
                                                )}
                                            </pre>
                                        </td>
                                    ))}
                                    <td>
                                        <div className="event-row-actions">
                                            <button
                                                className="btn btn-sm btn-outline-danger"
                                                type="button"
                                                title="Delete event"
                                                aria-label={`Delete event ${event.id}`}
                                                onClick={() =>
                                                    onDeleteEvent(event.id)
                                                }
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
    );
}

export function ackConsumerLabel(consumer: AckConsumer) {
    const group = consumer.group || consumer.Group;
    const name = consumer.consumer || consumer.Consumer;

    if (group && name) return `${group}: ${name}`;
    return name || group || "Unknown";
}

export function ackBadgeLabel(consumers: AckConsumer[]) {
    return consumers.length > 1 ? `Acked ${consumers.length}` : "Acked";
}

export function ackBadgeTitle(consumers: AckConsumer[]) {
    return consumers
        .map((consumer) => {
            const timestamp = consumer.timestamp || consumer.Timestamp;
            const label = ackConsumerLabel(consumer);

            return timestamp ? `${label} at ${timestamp}` : label;
        })
        .join("\n");
}
