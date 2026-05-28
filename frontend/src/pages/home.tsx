import { useEffect, useMemo, useState } from 'react'
import type { FormEvent } from 'react'
import Layout from '../components/Layout'
import { MatricCard } from '../components/MatricCard'

type StreamListItem = {
  name: string
  displayName?: string
  length: number
  groups: number
}

type StreamsResponse = {
  redisUrl: string
  streams: StreamListItem[]
}

type StreamTreeNode = {
  name: string
  displayName: string
  fullName: string
  length: number
  groups: number
  isStream: boolean
  children: StreamTreeNode[]
}

function buildStreamTree(streams: StreamListItem[]): StreamTreeNode[] {
  const roots: StreamTreeNode[] = []

  for (const stream of streams) {
    insertStream(roots, stream, stream.name.split(':'), stream.name)
  }

  sortNodes(roots)
  return roots
}

function insertStream(nodes: StreamTreeNode[], stream: StreamListItem, parts: string[], fullName: string) {
  if (parts.length === 0 || parts.some((part) => part === '')) {
    nodes.push(streamNode(stream, fullName, fullName))
    return
  }

  if (parts.length === 1) {
    nodes.push(streamNode(stream, parts[0], fullName))
    return
  }

  let group = nodes.find((node) => !node.isStream && node.name === parts[0])
  if (!group) {
    group = {
      name: parts[0],
      displayName: parts[0],
      fullName: '',
      length: 0,
      groups: 0,
      isStream: false,
      children: [],
    }
    nodes.push(group)
  }

  insertStream(group.children, stream, parts.slice(1), fullName)
}

function streamNode(stream: StreamListItem, displayName: string, fullName: string): StreamTreeNode {
  return {
    name: displayName,
    displayName: stream.displayName || displayName,
    fullName,
    length: stream.length,
    groups: stream.groups,
    isStream: true,
    children: [],
  }
}

function sortNodes(nodes: StreamTreeNode[]) {
  nodes.sort((a, b) => {
    if (a.isStream !== b.isStream) return a.isStream ? 1 : -1
    return a.displayName.localeCompare(b.displayName)
  })
  nodes.forEach((node) => sortNodes(node.children))
}

export default function Home() {
  const [streams, setStreams] = useState<StreamListItem[]>([])
  const [redisUrl, setRedisUrl] = useState('')
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [showModal, setShowModal] = useState(false)
  const [newStreamName, setNewStreamName] = useState('')
  const [saving, setSaving] = useState(false)
  const [formError, setFormError] = useState('')

  const loadStreams = async () => {
    setLoading(true)
    setError('')

    try {
      const response = await fetch('/api/v1/streams')
      if (!response.ok) throw new Error(await response.text())
      const data = (await response.json()) as StreamsResponse
      setStreams(data.streams || [])
      setRedisUrl(data.redisUrl || '')
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load streams')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    loadStreams()
  }, [])

  const tree = useMemo(() => buildStreamTree(streams), [streams])

  const createStream = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    setSaving(true)
    setFormError('')

    try {
      const body = new URLSearchParams({ name: newStreamName.trim() })
      const response = await fetch('/api/v1/streams', {
        method: 'POST',
        headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
        body,
      })
      if (!response.ok) throw new Error(await response.text())

      setNewStreamName('')
      setShowModal(false)
      await loadStreams()
    } catch (err) {
      setFormError(err instanceof Error ? err.message : 'Failed to create stream')
    } finally {
      setSaving(false)
    }
  }

  const deleteStream = async (stream: string) => {
    if (!window.confirm(`Delete stream '${stream}' ?`)) return

    const response = await fetch(`/api/v1/streams/${encodeURIComponent(stream)}`, { method: 'DELETE' })
    if (!response.ok) {
      setError(await response.text())
      return
    }

    await loadStreams()
  }

  return (
    <Layout showAddStream onAddStream={() => setShowModal(true)}>
      <div className="metric-grid">
        <MatricCard label='Streams' value={streams.length} />
        <MatricCard label='Source' value={redisUrl || 'Loading...'}/>
      </div>

      <section className="app-panel">
        <div className="panel-head">
          <h2 className="panel-title">Stream list</h2>
          <span className="panel-meta">Click a stream to inspect consumers and events</span>
        </div>

        {loading && <div className="empty-state">Loading streams...</div>}
        {error && <div className="empty-state text-danger">{error}</div>}
        {!loading && !error && streams.length === 0 && <div className="empty-state">No streams found</div>}
        {!loading && !error && streams.length > 0 && (
          <div className="stream-tree">
            <div className="stream-tree-head">
              <span>Stream</span>
              <span>Length</span>
              <span>Groups</span>
              <span>Status</span>
              <span></span>
            </div>
            {tree.map((node) => (
              <StreamNode key={node.fullName || node.name} node={node} onDeleteStream={deleteStream} />
            ))}
          </div>
        )}
      </section>

      {showModal && (
        <>
          <div className="modal fade show d-block" tabIndex={-1} role="dialog" aria-modal="true">
            <div className="modal-dialog">
              <div className="modal-content">
                <form onSubmit={createStream}>
                  <div className="modal-header">
                    <div>
                      <h5 className="modal-title">Add stream</h5>
                      <p className="mb-0 panel-meta">Create an empty Redis stream so it appears in the landing list.</p>
                    </div>
                    <button
                      type="button"
                      className="btn-close"
                      aria-label="Close"
                      onClick={() => setShowModal(false)}
                    ></button>
                  </div>
                  <div className="modal-body">
                    <label className="form-label" htmlFor="stream-name">
                      Stream name
                    </label>
                    <input
                      id="stream-name"
                      type="text"
                      name="name"
                      className="form-control"
                      placeholder="orders.created"
                      value={newStreamName}
                      onChange={(event) => setNewStreamName(event.target.value)}
                      required
                    />
                    {formError && <div className="form-text text-danger">{formError}</div>}
                  </div>
                  <div className="modal-footer">
                    <button type="button" className="btn btn-outline-secondary" onClick={() => setShowModal(false)}>
                      Cancel
                    </button>
                    <button type="submit" className="btn btn-primary" disabled={saving || !newStreamName.trim()}>
                      {saving ? 'Creating...' : 'Create stream'}
                    </button>
                  </div>
                </form>
              </div>
            </div>
          </div>
          <div className="modal-backdrop fade show"></div>
        </>
      )}
    </Layout>
  )
}

function StreamNode({
  node,
  onDeleteStream,
}: {
  node: StreamTreeNode
  onDeleteStream: (stream: string) => void
}) {
  if (node.isStream) {
    return (
      <div className="stream-tree-row">
        <div className="stream-tree-name">
          <a href={`/streams/${encodeURIComponent(node.fullName)}`} className="stream-link truncate">
            {node.displayName}
          </a>
        </div>
        <span>{node.length}</span>
        <span>{node.groups}</span>
        <span>
          <span className="status-pill">Available</span>
        </span>
        <span className="text-end">
          <button
            className="btn btn-sm btn-outline-danger"
            type="button"
            title="Delete stream"
            aria-label={`Delete stream ${node.fullName}`}
            onClick={() => onDeleteStream(node.fullName)}
          >
            <i className="fa-solid fa-trash"></i>
          </button>
        </span>
      </div>
    )
  }

  return (
    <details className="stream-tree-group" open>
      <summary>
        <span className="stream-tree-caret">
          <i className="fa-solid fa-chevron-right"></i>
        </span>
        <span className="stream-tree-group-name truncate">{node.displayName}</span>
        <span className="panel-meta">{node.children.length} item(s)</span>
      </summary>
      <div className="stream-tree-children">
        {node.children.map((child) => (
          <StreamNode key={child.fullName || child.name} node={child} onDeleteStream={onDeleteStream} />
        ))}
      </div>
    </details>
  )
}
