import { createFileRoute, Link } from '@tanstack/react-router'
import { useAuth } from '../../context/AuthContext'
import {
  FolderOpen,
  FileUp,
  AlertCircle,
  Download,
  Plus,
  Eye,
  Clock,
  CheckCircle2,
  Zap,
} from 'lucide-react'

export const Route = createFileRoute('/dashboard/')({
  component: DashboardOverview,
})

function DashboardOverview() {
  const { user } = useAuth()

  return (
    <div className="animate-fade-in">
      {/* Welcome Header */}
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
              fontSize: 32,
              fontWeight: 700,
              color: 'var(--on-surface)',
              marginBottom: 8,
            }}
          >
            Welcome back, {user?.name || 'Guest'}
          </h1>
          <p style={{ color: 'var(--on-surface-variant)', fontSize: 15 }}>
            Here is the status of your current health insurance disputes.
          </p>
        </div>
        <Link
          to="/dashboard/cases/new"
          className="btn btn-primary"
          style={{ gap: 8 }}
        >
          <Plus size={16} />
          Create New Case
        </Link>
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
        <StatCard
          label="Active Cases"
          value="2"
          icon={<FolderOpen size={18} />}
        />
        <StatCard
          label="Waiting for Docs"
          value="1"
          icon={<FileUp size={18} />}
        />
        <StatCard
          label="Review Required"
          value="1"
          icon={<AlertCircle size={18} />}
          alert
        />
        <StatCard
          label="Ready for Export"
          value="0"
          icon={<Download size={18} />}
        />
      </div>

      {/* Content Grid */}
      <div
        style={{
          display: 'grid',
          gridTemplateColumns: '2fr 1fr',
          gap: 24,
        }}
      >
        {/* Cases Table */}
        <div>
          <h2
            style={{
              fontFamily: 'var(--font-headline)',
              fontSize: 20,
              fontWeight: 600,
              marginBottom: 16,
              color: 'var(--on-surface)',
            }}
          >
            Current Cases
          </h2>
          <div className="card" style={{ padding: 0, overflow: 'hidden' }}>
            <table className="data-table">
              <thead>
                <tr>
                  <th>Case ID & Insurer</th>
                  <th>Issue</th>
                  <th>Disputed Amount</th>
                  <th>Status</th>
                  <th>Action</th>
                </tr>
              </thead>
              <tbody>
                <tr>
                  <td>
                    <div
                      style={{
                        fontWeight: 600,
                        fontSize: 14,
                        color: 'var(--on-surface)',
                      }}
                    >
                      BN-2026-000142
                    </div>
                    <div style={{ fontSize: 12, color: 'var(--on-surface-variant)' }}>
                      Horizon Health
                    </div>
                  </td>
                  <td style={{ fontSize: 14 }}>Room-Rent Deduction</td>
                  <td style={{ fontWeight: 600, fontSize: 14 }}>₹77,000</td>
                  <td>
                    <span className="badge badge-danger">
                      <span style={{ width: 6, height: 6, borderRadius: '50%', background: 'currentColor' }} />
                      Review Required
                    </span>
                  </td>
                  <td>
                    <Link
                      to="/dashboard"
                      style={{
                        color: 'var(--accent)',
                        fontSize: 14,
                        fontWeight: 500,
                        textDecoration: 'none',
                        display: 'flex',
                        alignItems: 'center',
                        gap: 4,
                      }}
                    >
                      <Eye size={14} />
                      View Details
                    </Link>
                  </td>
                </tr>
                <tr>
                  <td>
                    <div
                      style={{
                        fontWeight: 600,
                        fontSize: 14,
                        color: 'var(--on-surface)',
                      }}
                    >
                      BN-2026-000089
                    </div>
                    <div style={{ fontSize: 12, color: 'var(--on-surface-variant)' }}>
                      Apex General
                    </div>
                  </td>
                  <td style={{ fontSize: 14 }}>Pre-existing Clause</td>
                  <td style={{ fontWeight: 600, fontSize: 14 }}>₹1,45,000</td>
                  <td>
                    <span className="badge badge-processing">
                      <span style={{ width: 6, height: 6, borderRadius: '50%', background: 'currentColor' }} />
                      Processing
                    </span>
                  </td>
                  <td>
                    <Link
                      to="/dashboard"
                      style={{
                        color: 'var(--accent)',
                        fontSize: 14,
                        fontWeight: 500,
                        textDecoration: 'none',
                        display: 'flex',
                        alignItems: 'center',
                        gap: 4,
                      }}
                    >
                      <Eye size={14} />
                      View Details
                    </Link>
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>

        {/* Recent Activity */}
        <div>
          <h2
            style={{
              fontFamily: 'var(--font-headline)',
              fontSize: 20,
              fontWeight: 600,
              marginBottom: 16,
              color: 'var(--on-surface)',
            }}
          >
            Recent Activity
          </h2>
          <div className="card">
            <ActivityItem
              icon={<AlertCircle size={16} />}
              iconColor="var(--accent)"
              title="Action Required: BN-2026-000142"
              description="Review the AI-generated dispute letter draft for Horizon Health."
              time="2 hours ago"
            />
            <ActivityItem
              icon={<CheckCircle2 size={16} />}
              iconColor="var(--on-surface-variant)"
              title="AI Analysis Completed"
              description="Case BN-2026-000142 policy documents analyzed against IRDAI guidelines."
              time="Yesterday, 14:30"
            />
            <ActivityItem
              icon={<Zap size={16} />}
              iconColor="var(--on-surface-variant)"
              title="Document OCR Successful"
              description="3 discharge summaries digitized and verified."
              time="Yesterday, 11:15"
            />
          </div>
        </div>
      </div>

      <style>{`
        @media (max-width: 1024px) {
          div[style*="grid-template-columns: repeat(4"] {
            grid-template-columns: repeat(2, 1fr) !important;
          }
          div[style*="grid-template-columns: 2fr 1fr"] {
            grid-template-columns: 1fr !important;
          }
        }
        @media (max-width: 640px) {
          div[style*="grid-template-columns: repeat(4"], div[style*="grid-template-columns: repeat(2"] {
            grid-template-columns: 1fr !important;
          }
        }
      `}</style>
    </div>
  )
}

/* ── StatCard ─────────────────────────────────────────────────────────── */
function StatCard({
  label,
  value,
  icon,
  alert,
}: {
  label: string
  value: string
  icon: React.ReactNode
  alert?: boolean
}) {
  return (
    <div className={`stat-card ${alert ? 'stat-card-alert' : ''}`}>
      <div
        style={{
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'center',
        }}
      >
        <span
          className="text-label-md"
          style={{ color: alert ? 'var(--status-danger)' : 'var(--on-surface-variant)' }}
        >
          {label}
        </span>
        <span style={{ color: alert ? 'var(--status-danger)' : 'var(--on-surface-variant)' }}>
          {icon}
        </span>
      </div>
      <span
        style={{
          fontFamily: 'var(--font-headline)',
          fontSize: 36,
          fontWeight: 700,
          color: alert ? 'var(--status-danger)' : 'var(--on-surface)',
          lineHeight: 1,
        }}
      >
        {value}
      </span>
    </div>
  )
}

/* ── ActivityItem ─────────────────────────────────────────────────────── */
function ActivityItem({
  icon,
  iconColor,
  title,
  description,
  time,
}: {
  icon: React.ReactNode
  iconColor: string
  title: string
  description: string
  time: string
}) {
  return (
    <div className="activity-item">
      <div
        style={{
          width: 28,
          height: 28,
          borderRadius: '50%',
          border: '2px solid var(--outline-variant)',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          flexShrink: 0,
          color: iconColor,
        }}
      >
        {icon}
      </div>
      <div>
        <div
          style={{
            fontWeight: 600,
            fontSize: 14,
            color: 'var(--on-surface)',
            marginBottom: 4,
          }}
        >
          {title}
        </div>
        <div
          style={{
            fontSize: 13,
            color: 'var(--on-surface-variant)',
            lineHeight: 1.5,
            marginBottom: 6,
          }}
        >
          {description}
        </div>
        <div style={{ fontSize: 12, color: 'var(--accent)', fontWeight: 500 }}>
          {time}
        </div>
      </div>
    </div>
  )
}
