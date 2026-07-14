import { createFileRoute } from '@tanstack/react-router'
import { Scale, MessageSquare, FileCheck, BookOpen } from 'lucide-react'

export const Route = createFileRoute('/dashboard/legal-aid')({
  component: LegalAidPage,
})

function LegalAidPage() {
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
        Legal Aid
      </h1>
      <p style={{ color: 'var(--on-surface-variant)', fontSize: 15, marginBottom: 40 }}>
        Access legal resources, expert consultation, and dispute resolution guidance.
      </p>

      <div
        style={{
          display: 'grid',
          gridTemplateColumns: 'repeat(auto-fill, minmax(280px, 1fr))',
          gap: 20,
        }}
      >
        {[
          {
            icon: Scale,
            title: 'IRDAI Guidelines',
            description: 'Browse the latest insurance regulatory guidelines and consumer rights.',
          },
          {
            icon: MessageSquare,
            title: 'Expert Consultation',
            description: 'Connect with vetted insurance dispute specialists for guidance.',
          },
          {
            icon: FileCheck,
            title: 'Ombudsman Process',
            description: 'Step-by-step guide to filing complaints with the Insurance Ombudsman.',
          },
          {
            icon: BookOpen,
            title: 'Knowledge Base',
            description: 'Frequently asked questions and case study resources.',
          },
        ].map((item) => {
          const Icon = item.icon
          return (
            <div key={item.title} className="card" style={{ cursor: 'pointer' }}>
              <div
                style={{
                  width: 48,
                  height: 48,
                  borderRadius: 'var(--radius-lg)',
                  background: 'var(--accent-light)',
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  marginBottom: 16,
                }}
              >
                <Icon size={24} style={{ color: 'var(--accent)' }} />
              </div>
              <h3
                style={{
                  fontFamily: 'var(--font-headline)',
                  fontSize: 18,
                  fontWeight: 600,
                  color: 'var(--on-surface)',
                  marginBottom: 8,
                }}
              >
                {item.title}
              </h3>
              <p style={{ fontSize: 14, color: 'var(--on-surface-variant)', lineHeight: 1.5 }}>
                {item.description}
              </p>
            </div>
          )
        })}
      </div>
    </div>
  )
}
