import { createFileRoute, Link } from '@tanstack/react-router'
import {
  Eye,
  FileText,
  Clock,
  CheckCircle2,
  AlertCircle,
  Download,
  Filter,
  Plus,
} from 'lucide-react'

export const Route = createFileRoute('/dashboard/claims')({
  component: MyClaimsPage,
})

const claims = [
  {
    id: 'BN-2026-000142',
    insurer: 'Horizon Health',
    type: 'Room-Rent Deduction',
    amount: '₹77,000',
    status: 'Review Required',
    statusType: 'danger' as const,
    date: 'Jul 10, 2026',
  },
  {
    id: 'BN-2026-000089',
    insurer: 'Apex General',
    type: 'Pre-existing Clause',
    amount: '₹1,45,000',
    status: 'Processing',
    statusType: 'processing' as const,
    date: 'Jul 8, 2026',
  },
  {
    id: 'BN-2025-000456',
    insurer: 'Star Health',
    type: 'Medical Necessity',
    amount: '₹2,10,000',
    status: 'Resolved',
    statusType: 'success' as const,
    date: 'Mar 15, 2025',
  },
  {
    id: 'BN-2025-000312',
    insurer: 'ICICI Lombard',
    type: 'Waiting Period',
    amount: '₹55,000',
    status: 'Draft',
    statusType: 'neutral' as const,
    date: 'Feb 2, 2025',
  },
]

function MyClaimsPage() {
  return (
    <div className="animate-fade-in">
      {/* Header */}
      <div
        style={{
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'center',
          marginBottom: 24,
        }}
      >
        <h1
          style={{
            fontFamily: 'var(--font-headline)',
            fontSize: 28,
            fontWeight: 700,
            color: 'var(--on-surface)',
          }}
        >
          My Claims
        </h1>
        <div style={{ display: 'flex', gap: 12 }}>
          <button className="btn btn-secondary">
            <Filter size={16} />
            Filter
          </button>
          <Link to="/dashboard/cases/new" className="btn btn-primary">
            <Plus size={16} />
            New Dispute
          </Link>
        </div>
      </div>

      {/* Claims Cards Grid */}
      <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
        {claims.map((claim) => (
          <div
            key={claim.id}
            className="card"
            style={{
              display: 'flex',
              justifyContent: 'space-between',
              alignItems: 'center',
              padding: '20px 24px',
              cursor: 'pointer',
              transition: 'all 0.2s ease',
            }}
            onMouseEnter={(e) => {
              e.currentTarget.style.borderColor = 'var(--accent)'
              e.currentTarget.style.transform = 'translateX(4px)'
            }}
            onMouseLeave={(e) => {
              e.currentTarget.style.borderColor = 'var(--outline-variant)'
              e.currentTarget.style.transform = 'translateX(0)'
            }}
          >
            <div style={{ display: 'flex', gap: 24, alignItems: 'center' }}>
              <div
                style={{
                  width: 44,
                  height: 44,
                  borderRadius: 'var(--radius-lg)',
                  background: 'var(--surface-container)',
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                }}
              >
                <FileText size={20} style={{ color: 'var(--accent)' }} />
              </div>
              <div>
                <div style={{ fontWeight: 600, fontSize: 15, color: 'var(--on-surface)', marginBottom: 4 }}>
                  {claim.id}
                </div>
                <div style={{ fontSize: 13, color: 'var(--on-surface-variant)' }}>
                  {claim.insurer} · {claim.type}
                </div>
              </div>
            </div>
            <div style={{ display: 'flex', alignItems: 'center', gap: 32 }}>
              <div style={{ textAlign: 'right' }}>
                <div style={{ fontWeight: 700, fontSize: 16, color: 'var(--on-surface)' }}>
                  {claim.amount}
                </div>
                <div style={{ fontSize: 12, color: 'var(--on-surface-variant)' }}>
                  {claim.date}
                </div>
              </div>
              <span className={`badge badge-${claim.statusType}`}>
                {claim.status}
              </span>
              <Link
                to="/dashboard"
                style={{ color: 'var(--accent)', display: 'flex', padding: 4 }}
              >
                <Eye size={18} />
              </Link>
            </div>
          </div>
        ))}
      </div>
    </div>
  )
}
