import { useState } from 'react'
import type { FormEvent } from 'react'

const defaultPayload = `{
  "source": "manual",
  "status": "created"
}`

type SendEventModalProps = {
  streamName: string
  saving: boolean
  error: string
  onClose: () => void
  onSubmit: (payload: string) => Promise<void>
}

export default function SendEventModal({ streamName, saving, error, onClose, onSubmit }: SendEventModalProps) {
  const [payload, setPayload] = useState(defaultPayload)

  const submit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    await onSubmit(payload)
  }

  return (
    <>
      <div className="modal fade show d-block" tabIndex={-1} role="dialog" aria-modal="true">
        <div className="modal-dialog modal-lg">
          <div className="modal-content">
            <form onSubmit={submit}>
              <div className="modal-header">
                <div>
                  <h5 className="modal-title">Send event</h5>
                  <p className="mb-0 panel-meta">Append a new Redis stream entry to {streamName}.</p>
                </div>
                <button type="button" className="btn-close" aria-label="Close" onClick={onClose}></button>
              </div>
              <div className="modal-body">
                <div className="mb-3">
                  <label className="form-label" htmlFor="event-stream">
                    Stream
                  </label>
                  <input id="event-stream" className="form-control" value={streamName} disabled />
                </div>
                <div>
                  <label className="form-label" htmlFor="event-payload">
                    Event values as JSON
                  </label>
                  <textarea
                    id="event-payload"
                    name="payload"
                    className="form-control event-payload-input"
                    spellCheck="false"
                    required
                    style={{ height: 300 }}
                    value={payload}
                    onChange={(event) => setPayload(event.target.value)}
                  />
                  <div className="form-text">Send a JSON object. Top-level keys become Redis stream fields.</div>
                  {error && <div className="form-text text-danger">{error}</div>}
                </div>
              </div>
              <div className="modal-footer">
                <button type="button" className="btn btn-outline-secondary" onClick={onClose}>
                  Cancel
                </button>
                <button type="submit" className="btn btn-primary" disabled={saving}>
                  {saving ? 'Sending...' : 'Send event'}
                </button>
              </div>
            </form>
          </div>
        </div>
      </div>
      <div className="modal-backdrop fade show"></div>
    </>
  )
}
