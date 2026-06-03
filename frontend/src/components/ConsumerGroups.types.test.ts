import type { StreamConsumer } from './ConsumerGroups'

const consumerWithIP: StreamConsumer = {
  name: 'worker-a',
  ip: '192.0.2.10',
  lastSeen: '2026-06-02 12:00:00 +08:00 CST',
  healthy: true,
  pending: 0,
}

void consumerWithIP
