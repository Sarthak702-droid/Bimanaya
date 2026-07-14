import { createFileRoute, Link } from '@tanstack/react-router'
import {
  Search,
  ChevronDown,
  ExternalLink,
  Heart,
  Shield,
  Car,
  ChevronLeft,
  ChevronRight,
} from 'lucide-react'

export const Route = createFileRoute('/reviewer/queue')({
  component: GrievanceQueue,
})

const cases = [
  {
    id: '#BN-2024-8829',
    category: 'Health',
    categoryIcon: Heart,
    insurer: 'Reliable Care Inc.',
    amount: '₹4,50,000',
    risk: 'HIGH',
    language: 'English',
    deadline: 'Today, 14:00',
    deadlineUrgent: true,
    status: 'Pending Review',
    statusColor: 'warning',
  },
  {
    id: '#BN-2024-8830',
    category: 'Motor',
    categoryIcon: Car,
    insurer: 'AutoGuard General',
    amount: '₹1,25,000',
    risk: 'LOW',
    language: 'Hindi',
    deadline: 'Oct 26, 2024',
    deadlineUrgent: false,
    status: 'In Progress',
    statusColor: 'info',
  },
  {
    id: '#BN-2024-8831',
    category: 'Life',
    categoryIcon: Shield,
    insurer: 'SecureLife Ltd.',
    amount: '₹15,00,000',
    risk: 'MEDIUM',
    language: 'Marathi',
    deadline: 'Oct 27, 2024',
    deadlineUrgent: false,
    status: 'Draft',
    statusColor: 'neutral',
  },
]

const riskColors: Record<string, string> = {
  HIGH: 'var(--status-danger)',
  MEDIUM: 'var(--status-warning)',
  LOW: 'var(--status-success)',
}

const riskBgColors: Record<string, string> = {
  HIGH: 'var(--status-danger-bg)',
  MEDIUM: 'var(--status-warning-bg)',
  LOW: 'var(--status-success-bg)',
}

function GrievanceQueue() {
  return (
    <div className="animate-fade-in">
      <div className="card" style={{ padding: 0, overflow: 'hidden' }}>
        {/* Header */}
        <div
          style={{
            padding: '24px 28px',
            display: 'flex',
            justifyContent: 'space-between',
            alignItems: 'flex-start',
            flexWrap: 'wrap',
            gap: 16,
          }}
        >
          <div>
            <h1
              style={{
                fontFamily: 'var(--font-headline)',
                fontSize: 24,
                fontWeight: 700,
                color: 'var(--on-surface)',
                marginBottom: 4,
              }}
            >
              Grievance Queue
            </h1>
            <p style={{ color: 'var(--on-surface-variant)', fontSize: 14 }}>
              Manage and assign pending legal and clinical reviews.
            </p>
          </div>

          <div style={{ display: 'flex', gap: 12, alignItems: 'center' }}>
            {/* Search */}
            <div
              style={{
                display: 'flex',
                alignItems: 'center',
                gap: 8,
                padding: '8px 14px',
                background: 'var(--surface-container)',
                borderRadius: 'var(--radius-lg)',
                border: '1px solid var(--outline-variant)',
              }}
            >
              <Search size={16} style={{ color: 'var(--on-surface-variant)' }} />
              <input
                type="text"
                placeholder="Search cases..."
                style={{
                  border: 'none',
                  background: 'transparent',
                  outline: 'none',
                  color: 'var(--on-surface)',
                  fontSize: 14,
                  width: 180,
                  fontFamily: 'var(--font-body)',
                }}
              />
            </div>

            {/* Filters */}
            <button className="btn btn-secondary" style={{ fontSize: 13 }}>
              All Risks <ChevronDown size={14} />
            </button>
            <button className="btn btn-secondary" style={{ fontSize: 13 }}>
              All Categories <ChevronDown size={14} />
            </button>
          </div>
        </div>

        {/* Table */}
        <table className="data-table">
          <thead>
            <tr>
              <th>CASE NUMBER</th>
              <th>CATEGORY</th>
              <th>INSURER</th>
              <th>DISPUTED AMT</th>
              <th>RISK LEVEL</th>
              <th>LANGUAGE</th>
              <th>DEADLINE</th>
              <th>STATUS</th>
              <th>ACTIONS</th>
            </tr>
          </thead>
          <tbody>
            {cases.map((c) => {
              const CategoryIcon = c.categoryIcon
              return (
                <tr key={c.id}>
                  <td>
                    <Link
                      to="/reviewer"
                      style={{ color: 'var(--accent)', textDecoration: 'none', fontWeight: 500 }}
                    >
                      {c.id}
                    </Link>
                  </td>
                  <td>
                    <span style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
                      <CategoryIcon size={14} style={{ color: 'var(--on-surface-variant)' }} />
                      {c.category}
                    </span>
                  </td>
                  <td>{c.insurer}</td>
                  <td style={{ fontWeight: 600 }}>{c.amount}</td>
                  <td>
                    <span
                      style={{
                        padding: '2px 10px',
                        borderRadius: 'var(--radius)',
                        fontSize: 11,
                        fontWeight: 700,
                        letterSpacing: '0.05em',
                        background: riskBgColors[c.risk],
                        color: riskColors[c.risk],
                      }}
                    >
                      {c.risk}
                    </span>
                  </td>
                  <td>{c.language}</td>
                  <td
                    style={{
                      color: c.deadlineUrgent ? 'var(--status-danger)' : 'var(--on-surface)',
                      fontWeight: c.deadlineUrgent ? 600 : 400,
                    }}
                  >
                    {c.deadline}
                  </td>
                  <td>
                    <span className={`badge badge-${c.statusColor}`}>
                      <span
                        style={{
                          width: 6,
                          height: 6,
                          borderRadius: '50%',
                          background: 'currentColor',
                        }}
                      />
                      {c.status}
                    </span>
                  </td>
                  <td>
                    <Link
                      to="/reviewer"
                      className="btn-ghost"
                      style={{ padding: 6 }}
                    >
                      <ExternalLink size={16} />
                    </Link>
                  </td>
                </tr>
              )
            })}
          </tbody>
        </table>

        {/* Pagination */}
        <div
          style={{
            display: 'flex',
            justifyContent: 'space-between',
            alignItems: 'center',
            padding: '12px 24px',
            fontSize: 13,
            color: 'var(--on-surface-variant)',
            borderTop: '1px solid var(--outline-variant)',
          }}
        >
          <span>Showing 1 to 3 of 42 entries</span>
          <div style={{ display: 'flex', gap: 4 }}>
            <button className="btn-ghost" style={{ padding: 4 }}>
              <ChevronLeft size={16} />
            </button>
            <button className="btn-ghost" style={{ padding: 4 }}>
              <ChevronRight size={16} />
            </button>
          </div>
        </div>
      </div>
    </div>
  )
}
