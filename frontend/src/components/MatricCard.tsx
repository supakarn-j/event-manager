type MatricCardProps = {
    label: string
    value: string | number
}
export function MatricCard({label, value}: MatricCardProps) {
    return (
        <div className="metric-card">
          <div className="metric-label">{label}</div>
          <strong className="metric-value">{value}</strong>
        </div>

    )
}