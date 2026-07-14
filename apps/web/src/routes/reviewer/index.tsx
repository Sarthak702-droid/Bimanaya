import { createFileRoute } from '@tanstack/react-router'
import {
  Briefcase,
  Clock,
  AlertTriangle,
  Users,
  RefreshCw,
  Timer,
  CheckCircle2,
  TrendingDown,
  Award,
} from 'lucide-react'

export const Route = createFileRoute('/reviewer/')({
  component: ReviewerDashboard,
})

function ReviewerDashboard() {
  return (
    <div className="animate-fade-in">
      {/* Header */}
      <div
        style={{
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'flex-start',
          marginBottom: 32,
        }}
      >
        <div>
          <h1
            style={{
              fontFamily: 'var(--font-headline)',
              fontSize: 28,
              fontWeight: 700,
              color: 'var(--on-surface)',
              marginBottom: 8,
            }}
          >
            Reviewer Workspace
          </h1>
          <p style={{ color: 'var(--on-surface-variant)', fontSize: 14 }}>
            Real-time metrics and active docket for your review.
          </p>
        </div>
        <button className="btn btn-primary">
          <RefreshCw size={16} />
          Sync Docket
        </button>
      </div>

      {/* Stat Cards */}
      <div
        style={{
          display: 'grid',
          gridTemplateColumns: 'repeat(4, 1fr)',
          gap: 16,
          marginBottom: 40,
        }}
      >
        <ReviewerStatCard label="ASSIGNED CASES" value="24" sub="Active" icon={<Briefcase size={18} />} />
        <ReviewerStatCard label="DUE TODAY" value="07" sub="Urgent" icon={<Clock size={18} />} alert />
        <ReviewerStatCard label="HIGH-RISK CASES" value="03" sub="Escalated" icon={<AlertTriangle size={18} />} danger />
        <ReviewerStatCard label="WAITING FOR USER" value="12" sub="Pending Info" icon={<Users size={18} />} />
      </div>

      {/* Charts Row */}
      <div
        style={{
          display: 'grid',
          gridTemplateColumns: '1.5fr 1fr',
          gap: 24,
          marginBottom: 40,
        }}
      >
        {/* Cases by Category */}
        <div className="card">
          <h3 className="text-label-md" style={{ marginBottom: 24, color: 'var(--on-surface-variant)' }}>
            CASES BY CATEGORY
          </h3>
          <div
            style={{
              display: 'flex',
              alignItems: 'flex-end',
              justifyContent: 'space-around',
              height: 200,
              paddingTop: 20,
            }}
          >
            {[
              { label: 'Medical Claim', height: 70 },
              { label: 'Policy Dispute', height: 45 },
              { label: 'Fraud Review', height: 25 },
              { label: 'Other', height: 15 },
            ].map((bar) => (
              <div key={bar.label} style={{ textAlign: 'center', flex: 1 }}>
                <div
                  style={{
                    width: 48,
                    height: `${bar.height}%`,
                    background: 'var(--accent)',
                    borderRadius: 'var(--radius) var(--radius) 0 0',
                    margin: '0 auto 12px',
                    opacity: 0.7 + (bar.height / 200),
                    transition: 'height 0.4s ease',
                  }}
                />
                <span style={{ fontSize: 11, color: 'var(--on-surface-variant)' }}>
                  {bar.label}
                </span>
              </div>
            ))}
          </div>
        </div>

        {/* Avg Review Time */}
        <div className="card" style={{ textAlign: 'center' }}>
          <h3 className="text-label-md" style={{ marginBottom: 24, color: 'var(--on-surface-variant)' }}>
            AVERAGE REVIEW TIME
          </h3>
          <div style={{ position: 'relative', width: 160, height: 160, margin: '0 auto 24px' }}>
            <svg width="160" height="160" viewBox="0 0 160 160">
              <circle
                cx="80"
                cy="80"
                r="70"
                fill="none"
                stroke="var(--surface-container-highest)"
                strokeWidth="12"
              />
              <circle
                cx="80"
                cy="80"
                r="70"
                fill="none"
                stroke="var(--accent)"
                strokeWidth="12"
                strokeDasharray={`${2 * Math.PI * 70 * 0.75} ${2 * Math.PI * 70 * 0.25}`}
                strokeDashoffset={2 * Math.PI * 70 * 0.25}
                strokeLinecap="round"
                transform="rotate(-90 80 80)"
              />
            </svg>
            <div
              style={{
                position: 'absolute',
                inset: 0,
                display: 'flex',
                flexDirection: 'column',
                alignItems: 'center',
                justifyContent: 'center',
              }}
            >
              <span
                style={{
                  fontFamily: 'var(--font-headline)',
                  fontSize: 36,
                  fontWeight: 700,
                  color: 'var(--on-surface)',
                }}
              >
                4.2
              </span>
              <span style={{ fontSize: 12, color: 'var(--on-surface-variant)' }}>Days / Case</span>
            </div>
          </div>
          <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: 13 }}>
            <span style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
              <span style={{ width: 8, height: 8, borderRadius: '50%', background: 'var(--accent)' }} />
              <span style={{ color: 'var(--on-surface-variant)' }}>In SLA</span>
            </span>
            <span style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
              <span style={{ width: 8, height: 8, borderRadius: '50%', background: 'var(--surface-container-highest)' }} />
              <span style={{ color: 'var(--on-surface-variant)' }}>Breached</span>
            </span>
          </div>
        </div>
      </div>

      {/* Performance Metrics Table */}
      <div className="card" style={{ padding: 0, overflow: 'hidden' }}>
        <div
          style={{
            display: 'flex',
            justifyContent: 'space-between',
            alignItems: 'center',
            padding: '16px 24px',
          }}
        >
          <h3 className="text-label-md" style={{ color: 'var(--on-surface-variant)' }}>
            REVIEWER PERFORMANCE METRICS
          </h3>
          <span
            style={{
              fontSize: 12,
              padding: '4px 12px',
              border: '1px solid var(--outline-variant)',
              borderRadius: 'var(--radius)',
              color: 'var(--on-surface-variant)',
            }}
          >
            Last 30 Days
          </span>
        </div>
        <table className="data-table">
          <thead>
            <tr>
              <th>Metric category</th>
              <th>Current Value</th>
              <th>Target Benchmark</th>
              <th>Status</th>
            </tr>
          </thead>
          <tbody>
            <tr>
              <td style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                <Timer size={16} style={{ color: 'var(--on-surface-variant)' }} />
                SLA Compliance Rate
              </td>
              <td style={{ fontWeight: 600 }}>94.5%</td>
              <td>95.0%</td>
              <td><span className="badge badge-danger">Below Target</span></td>
            </tr>
            <tr>
              <td style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                <CheckCircle2 size={16} style={{ color: 'var(--on-surface-variant)' }} />
                Citation Correction Rate
              </td>
              <td style={{ fontWeight: 600 }}>2.1%</td>
              <td>{'< 5.0%'}</td>
              <td><span className="badge badge-success">Excellent</span></td>
            </tr>
            <tr>
              <td style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                <TrendingDown size={16} style={{ color: 'var(--on-surface-variant)' }} />
                Decision Overturn Rate
              </td>
              <td style={{ fontWeight: 600 }}>0.8%</td>
              <td>{'< 2.0%'}</td>
              <td><span className="badge badge-info">Optimal</span></td>
            </tr>
          </tbody>
        </table>
      </div>

      <style>{`
        @media (max-width: 1024px) {
          div[style*="grid-template-columns: repeat(4"] {
            grid-template-columns: repeat(2, 1fr) !important;
          }
          div[style*="grid-template-columns: 1.5fr"] {
            grid-template-columns: 1fr !important;
          }
        }
      `}</style>
    </div>
  )
}

function ReviewerStatCard({
  label,
  value,
  sub,
  icon,
  alert,
  danger,
}: {
  label: string
  value: string
  sub: string
  icon: React.ReactNode
  alert?: boolean
  danger?: boolean
}) {
  return (
    <div
      className="stat-card"
      style={{
        borderColor: danger ? 'var(--status-danger)' : alert ? 'var(--status-warning)' : undefined,
        background: danger ? 'var(--status-danger-bg)' : alert ? 'var(--status-warning-bg)' : undefined,
      }}
    >
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <span className="text-label-md" style={{ color: 'var(--on-surface-variant)' }}>{label}</span>
        {danger && <AlertTriangle size={16} style={{ color: 'var(--status-danger)' }} />}
      </div>
      <div style={{ display: 'flex', alignItems: 'baseline', gap: 8 }}>
        <span
          style={{
            fontFamily: 'var(--font-headline)',
            fontSize: 36,
            fontWeight: 700,
            color: danger ? 'var(--status-danger)' : alert ? 'var(--status-warning)' : 'var(--on-surface)',
          }}
        >
          {value}
        </span>
        <span
          style={{
            fontSize: 13,
            color: danger ? 'var(--status-danger)' : alert ? 'var(--status-warning)' : 'var(--accent)',
          }}
        >
          {sub}
        </span>
      </div>
    </div>
  )
}
