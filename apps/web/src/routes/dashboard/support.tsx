import { createFileRoute } from '@tanstack/react-router'
import { HelpCircle, MessageCircle, Phone, Mail } from 'lucide-react'

export const Route = createFileRoute('/dashboard/support')({
  component: SupportPage,
})

function SupportPage() {
  return (
    <div className="animate-fade-in">
      <h1 style={{ fontFamily: 'var(--font-headline)', fontSize: 28, fontWeight: 700, color: 'var(--on-surface)', marginBottom: 8 }}>
        Support
      </h1>
      <p style={{ color: 'var(--on-surface-variant)', fontSize: 15, marginBottom: 40 }}>
        Get help with your insurance disputes and platform usage.
      </p>
      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(280px, 1fr))', gap: 20 }}>
        {[
          { icon: MessageCircle, title: 'Live Chat', desc: 'Chat with our support team in real-time.' },
          { icon: Mail, title: 'Email Support', desc: 'Send us a detailed query at support@bimanyaya.in' },
          { icon: Phone, title: 'Call Us', desc: 'Speak directly with a support specialist.' },
          { icon: HelpCircle, title: 'FAQ', desc: 'Browse frequently asked questions.' },
        ].map((item) => {
          const Icon = item.icon
          return (
            <div key={item.title} className="card" style={{ cursor: 'pointer' }}>
              <div style={{ width: 48, height: 48, borderRadius: 'var(--radius-lg)', background: 'var(--accent-light)', display: 'flex', alignItems: 'center', justifyContent: 'center', marginBottom: 16 }}>
                <Icon size={24} style={{ color: 'var(--accent)' }} />
              </div>
              <h3 style={{ fontFamily: 'var(--font-headline)', fontSize: 18, fontWeight: 600, color: 'var(--on-surface)', marginBottom: 8 }}>{item.title}</h3>
              <p style={{ fontSize: 14, color: 'var(--on-surface-variant)', lineHeight: 1.5 }}>{item.desc}</p>
            </div>
          )
        })}
      </div>
    </div>
  )
}
