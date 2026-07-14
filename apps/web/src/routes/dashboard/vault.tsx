import { createFileRoute } from '@tanstack/react-router'
import { FileText, Upload, Search, Shield } from 'lucide-react'

export const Route = createFileRoute('/dashboard/vault')({
  component: PolicyVaultPage,
})

function PolicyVaultPage() {
  return (
    <div className="animate-fade-in">
      <h1
        style={{
          fontFamily: 'var(--font-headline)',
          fontSize: 28,
          fontWeight: 700,
          color: 'var(--on-surface)',
          marginBottom: 8,
        }}
      >
        Policy Vault
      </h1>
      <p style={{ color: 'var(--on-surface-variant)', fontSize: 15, marginBottom: 32 }}>
        Securely store and organize all your insurance documentation with end-to-end encryption.
      </p>

      {/* Search */}
      <div
        style={{
          display: 'flex',
          alignItems: 'center',
          gap: 8,
          padding: '10px 16px',
          background: 'var(--surface-container)',
          borderRadius: 'var(--radius-lg)',
          border: '1px solid var(--outline-variant)',
          marginBottom: 32,
          maxWidth: 400,
        }}
      >
        <Search size={18} style={{ color: 'var(--on-surface-variant)' }} />
        <input
          type="text"
          placeholder="Search documents..."
          style={{
            border: 'none',
            background: 'transparent',
            outline: 'none',
            color: 'var(--on-surface)',
            fontSize: 14,
            flex: 1,
            fontFamily: 'var(--font-body)',
          }}
        />
      </div>

      {/* Documents Grid */}
      <div
        style={{
          display: 'grid',
          gridTemplateColumns: 'repeat(auto-fill, minmax(280px, 1fr))',
          gap: 16,
        }}
      >
        {[
          { name: 'Health Policy - Horizon', pages: 45, type: 'PDF', date: 'Jun 2026' },
          { name: 'Motor Policy - AutoGuard', pages: 22, type: 'PDF', date: 'Mar 2026' },
          { name: 'Rejection Letter #001', pages: 3, type: 'PDF', date: 'Jul 2026' },
          { name: 'Discharge Summary', pages: 8, type: 'PDF', date: 'Jul 2026' },
        ].map((doc) => (
          <div
            key={doc.name}
            className="card"
            style={{
              display: 'flex',
              gap: 16,
              alignItems: 'flex-start',
              cursor: 'pointer',
            }}
          >
            <div
              style={{
                width: 44,
                height: 44,
                borderRadius: 'var(--radius-lg)',
                background: 'var(--accent-light)',
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                flexShrink: 0,
              }}
            >
              <FileText size={20} style={{ color: 'var(--accent)' }} />
            </div>
            <div>
              <div style={{ fontWeight: 600, fontSize: 14, color: 'var(--on-surface)', marginBottom: 4 }}>
                {doc.name}
              </div>
              <div style={{ fontSize: 12, color: 'var(--on-surface-variant)' }}>
                {doc.pages} pages · {doc.type} · {doc.date}
              </div>
            </div>
          </div>
        ))}

        {/* Upload new */}
        <div
          className="card"
          style={{
            display: 'flex',
            flexDirection: 'column',
            alignItems: 'center',
            justifyContent: 'center',
            gap: 12,
            cursor: 'pointer',
            minHeight: 100,
            borderStyle: 'dashed',
          }}
        >
          <Upload size={24} style={{ color: 'var(--on-surface-variant)' }} />
          <span style={{ fontSize: 14, color: 'var(--on-surface-variant)', fontWeight: 500 }}>
            Upload New Document
          </span>
        </div>
      </div>
    </div>
  )
}
