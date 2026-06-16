import { useCallback, useEffect, useState } from 'react'
import { useParams } from 'react-router-dom'
import ConsumerGroups from '../components/ConsumerGroups'
import type { StreamConsumerGroup } from '../components/ConsumerGroups'
import EventTable from '../components/EventTable'
import type { StreamEvent } from '../components/EventTable'
import Layout from '../components/Layout'
import SendEventModal from '../components/SendEventModal'
import StreamTopbar from '../components/StreamTopbar'

type StreamDetailResponse = {
  name: string
  groups: StreamConsumerGroup[]
}

type HealthCheckEvent = {
  action: 'health_check'
  group?: string
  consumer?: string
  ip?: string
  healthy?: boolean
  last_seen?: string
}

type EventAddedEvent = {
  action: 'event_added'
}

type StreamPublication = HealthCheckEvent | EventAddedEvent

type CentrifugeSubscription = {
  on: (event: 'publication', handler: (ctx: { data?: StreamPublication }) => void) => void
  subscribe: () => void
  unsubscribe?: () => void
}

type CentrifugeClient = {
  newSubscription: (channel: string) => CentrifugeSubscription
  connect: () => void
  disconnect?: () => void
}

declare global {
  interface Window {
    Centrifuge?: new (url: string) => CentrifugeClient
  }
}

export default function StreamDetail() {
  const params = useParams()
  const streamName = params.streamName || ''
  const [groups, setGroups] = useState<StreamConsumerGroup[]>([])
  const [headers, setHeaders] = useState<string[]>([])
  const [events, setEvents] = useState<StreamEvent[]>([])
  const [loadingDetail, setLoadingDetail] = useState(true)
  const [loadingEvents, setLoadingEvents] = useState(true)
  const [error, setError] = useState('')
  const [showEventModal, setShowEventModal] = useState(false)
  const [savingEvent, setSavingEvent] = useState(false)
  const [eventFormError, setEventFormError] = useState('')

  const streamPath = encodeURIComponent(streamName)

  const loadDetail = useCallback(async () => {
    if (!streamName) return

    setLoadingDetail(true)
    setError('')

    try {
      const response = await fetch(`/api/v1/streams/${streamPath}`)
      if (!response.ok) throw new Error(await response.text())
      const data = (await response.json()) as StreamDetailResponse
      setGroups(data.groups || [])
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load stream detail')
    } finally {
      setLoadingDetail(false)
    }
  }, [streamName, streamPath])

  const loadEvents = useCallback(async () => {
    if (!streamName) return

    setLoadingEvents(true)
    setError('')

    try {
      const response = await fetch(`/api/v1/streams/${streamPath}/events`)
      if (!response.ok) throw new Error(await response.text())
      const data = ((await response.json()) as StreamEvent[]) || []
      setHeaders(deriveEventHeaders(data))
      setEvents(data)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load stream events')
    } finally {
      setLoadingEvents(false)
    }
  }, [streamName, streamPath])

  useEffect(() => {
    loadDetail()
    loadEvents()
  }, [loadDetail, loadEvents])

  useEffect(() => {
    if (!streamName || !window.Centrifuge) return

    const centrifuge = new window.Centrifuge(`ws://${location.host}/ws`)
    const sub = centrifuge.newSubscription(streamName)

    sub.on('publication', (ctx) => {
      const event = ctx.data
      if (!event) return

      if (event.action === 'health_check') {
        updateConsumerHealth(event)
      }
      if (event.action === 'event_added') {
        loadEvents()
      }
    })

    sub.subscribe()
    centrifuge.connect()

    return () => {
      sub.unsubscribe?.()
      centrifuge.disconnect?.()
    }
  }, [loadEvents, streamName])

  const updateConsumerHealth = (event: HealthCheckEvent) => {
    if (!event.group || !event.consumer) return
    const groupName = event.group
    const consumerName = event.consumer

    setGroups((current) => {
      const groupIndex = current.findIndex((group) => group.name === groupName)
      if (groupIndex === -1 && !event.healthy) return current

      const next = [...current]
      if (groupIndex === -1) {
        next.unshift({
          name: groupName,
          pending: 0,
          consumers: [
            {
              name: consumerName,
              ip: event.ip,
              lastSeen: event.last_seen || '-',
              healthy: Boolean(event.healthy),
              pending: 0,
            },
          ],
        })
        return next
      }

      const group = next[groupIndex]
      const consumerIndex = group.consumers.findIndex((consumer) => consumer.name === consumerName)
      const consumers = [...group.consumers]
      if (consumerIndex === -1) {
        if (!event.healthy) return current
        consumers.push({
          name: consumerName,
          ip: event.ip,
          lastSeen: event.last_seen || '-',
          healthy: true,
          pending: 0,
        })
      } else {
        consumers[consumerIndex] = {
          ...consumers[consumerIndex],
          ip: event.ip || consumers[consumerIndex].ip,
          lastSeen: event.last_seen || consumers[consumerIndex].lastSeen,
          healthy: Boolean(event.healthy),
        }
      }

      next[groupIndex] = { ...group, consumers }
      return next
    })
  }

  const deleteConsumer = async (group: string, consumer: string) => {
    if (!window.confirm(`Delete consumer '${consumer}' ?`)) return

    const response = await fetch(
      `/api/v1/streams/${streamPath}/consumers/${encodeURIComponent(group)}/${encodeURIComponent(consumer)}`,
      { method: 'DELETE' },
    )
    if (!response.ok) {
      setError(await response.text())
      return
    }

    setGroups((current) =>
      current.map((item) =>
        item.name === group ? { ...item, consumers: item.consumers.filter((entry) => entry.name !== consumer) } : item,
      ),
    )
  }

  const deleteEvent = async (eventID: string) => {
    if (!window.confirm(`Delete event '${eventID}' ?`)) return

    const response = await fetch(`/api/v1/streams/${streamPath}/events/${encodeURIComponent(eventID)}`, {
      method: 'DELETE',
    })
    if (!response.ok) {
      setError(await response.text())
      return
    }

    setEvents((current) => current.filter((event) => event.id !== eventID))
  }

  const sendEvent = async (payload: string) => {
    setSavingEvent(true)
    setEventFormError('')

    try {
      const response = await fetch(`/api/v1/streams/${streamPath}/events`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: payload,
      })
      if (!response.ok) throw new Error(await response.text())

      setShowEventModal(false)
      await loadEvents()
    } catch (err) {
      setEventFormError(err instanceof Error ? err.message : 'Failed to send event')
    } finally {
      setSavingEvent(false)
    }
  }

  return (
    <Layout topbar={<StreamTopbar streamName={streamName} onSendEvent={() => setShowEventModal(true)} />}>
      {error && <div className="alert alert-danger">{error}</div>}
      {loadingDetail && (
        <section className="app-panel">
          <div className="empty-state">Loading stream detail...</div>
        </section>
      )}

      {!loadingDetail && (
        <div className="detail-grid">
          <ConsumerGroups streamName={streamName} groups={groups} onDeleteConsumer={deleteConsumer} />
          <div>
            <EventTable
              streamName={streamName}
              headers={headers}
              events={events}
              loading={loadingEvents}
              onRefresh={loadEvents}
              onDeleteEvent={deleteEvent}
            />
          </div>
        </div>
      )}

      {showEventModal && (
        <SendEventModal
          streamName={streamName}
          saving={savingEvent}
          error={eventFormError}
          onClose={() => setShowEventModal(false)}
          onSubmit={sendEvent}
        />
      )}
    </Layout>
  )
}

export function deriveEventHeaders(events: StreamEvent[]) {
  const headers = new Set<string>()

  for (const event of events) {
    for (const header of Object.keys(event.values || {})) {
      headers.add(header)
    }
  }

  return Array.from(headers).sort()
}
