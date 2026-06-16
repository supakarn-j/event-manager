import { deriveEventHeaders } from './stream-detail'
import type { StreamEvent } from '../components/EventTable'

const eventListResponse: StreamEvent[] = [
  {
    id: '1747041200000-0',
    values: {
      source: 'manual',
      status: 'created',
    },
    consumers: [{ Group: 'billing', Consumer: 'worker-a', Timestamp: '2026-06-09 12:34:56 +08:00 CST' }],
  },
]

const headers: string[] = deriveEventHeaders(eventListResponse)

void headers
