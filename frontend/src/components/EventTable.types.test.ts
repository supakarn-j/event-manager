import { ackBadgeTitle } from './EventTable'
import type { StreamEvent } from './EventTable'

const eventWithAckConsumers: StreamEvent = {
  id: '1747041200000-0',
  values: {
    source: 'manual',
  },
  consumers: [
    {
      group: 'billing',
      consumer: 'worker-a',
      timestamp: '2026-06-09 12:34:56 +08:00 CST',
    },
  ],
}

void eventWithAckConsumers

const ackTitle: string = ackBadgeTitle(eventWithAckConsumers.consumers || [])

void ackTitle

const eventWithoutTimestampColumn: StreamEvent = {
  id: '1747041200001-0',
  values: {},
}

// @ts-expect-error event timestamps are not part of the event table model
eventWithoutTimestampColumn.timestamp = '2026-06-09 12:34:56 +08:00 CST'
