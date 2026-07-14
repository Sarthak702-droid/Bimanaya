import { createFileRoute } from '@tanstack/react-router'
import { User, Bell, Globe, Lock, Palette } from 'lucide-react'
import { useTheme } from '../../context/ThemeContext'
import { useAuth } from '../../context/AuthContext'

export const Route = createFileRoute('/dashboard/settings')({
  component: SettingsPage,
})

function SettingsPage() {
  const { theme, setTheme } = useTheme()
  const { user } = useAuth()

  return (
    <div className="animate-fade-in">
      <h1 style={{ fontFamily: 'var(--font-headline)', fontSize: 28, fontWeight: 700, color: 'var(--on-surface)', marginBottom: 8 }}>
        Settings
      </h1>
      <p style={{ color: 'var(--on-surface-variant)', fontSize: 15, marginBottom: 40 }}>
        Manage your account preferences and platform settings.
      </p>

      <div style={{ maxWidth: 640, display: 'flex', flexDirection: 'column', gap: 24 }}>
        {/* Profile */}
        <div className="card">
          <div style={{ display: 'flex', alignItems: 'center', gap: 12, marginBottom: 20 }}>
            <User size={20} style={{ color: 'var(--accent)' }} />
            <h3 style={{ fontFamily: 'var(--font-headline)', fontSize: 18, fontWeight: 600, color: 'var(--on-surface)' }}>Profile</h3>
          </div>
          <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
            <div>
              <label className="text-label-md" style={{ display: 'block', marginBottom: 6, color: 'var(--on-surface)' }}>Full Name</label>
              <input className="input" defaultValue={user?.name || 'Guest User'} />
            </div>
            <div>
              <label className="text-label-md" style={{ display: 'block', marginBottom: 6, color: 'var(--on-surface)' }}>Email</label>
              <input className="input" defaultValue={user?.email || 'guest@example.com'} disabled style={{ opacity: 0.6 }} />
            </div>
          </div>
        </div>

        {/* Theme */}
        <div className="card">
          <div style={{ display: 'flex', alignItems: 'center', gap: 12, marginBottom: 20 }}>
            <Palette size={20} style={{ color: 'var(--accent)' }} />
            <h3 style={{ fontFamily: 'var(--font-headline)', fontSize: 18, fontWeight: 600, color: 'var(--on-surface)' }}>Appearance</h3>
          </div>
          <div style={{ display: 'flex', gap: 12 }}>
            <button
              className={`btn ${theme === 'light' ? 'btn-primary' : 'btn-secondary'}`}
              onClick={() => setTheme('light')}
            >
              Light Mode
            </button>
            <button
              className={`btn ${theme === 'dark' ? 'btn-primary' : 'btn-secondary'}`}
              onClick={() => setTheme('dark')}
            >
              Dark Mode
            </button>
          </div>
        </div>

        {/* Language */}
        <div className="card">
          <div style={{ display: 'flex', alignItems: 'center', gap: 12, marginBottom: 20 }}>
            <Globe size={20} style={{ color: 'var(--accent)' }} />
            <h3 style={{ fontFamily: 'var(--font-headline)', fontSize: 18, fontWeight: 600, color: 'var(--on-surface)' }}>Language</h3>
          </div>
          <select className="input" style={{ maxWidth: 300 }} defaultValue="en">
            <option value="en">English</option>
            <option value="hi">हिन्दी (Hindi)</option>
            <option value="mr">मराठी (Marathi)</option>
            <option value="ta">தமிழ் (Tamil)</option>
          </select>
        </div>

        {/* Notifications */}
        <div className="card">
          <div style={{ display: 'flex', alignItems: 'center', gap: 12, marginBottom: 20 }}>
            <Bell size={20} style={{ color: 'var(--accent)' }} />
            <h3 style={{ fontFamily: 'var(--font-headline)', fontSize: 18, fontWeight: 600, color: 'var(--on-surface)' }}>Notifications</h3>
          </div>
          <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
            {['Email notifications for case updates', 'SMS alerts for urgent actions', 'Weekly digest summary'].map((label) => (
              <label key={label} style={{ display: 'flex', alignItems: 'center', gap: 10, fontSize: 14, color: 'var(--on-surface)', cursor: 'pointer' }}>
                <input type="checkbox" defaultChecked style={{ accentColor: 'var(--accent)' }} />
                {label}
              </label>
            ))}
          </div>
        </div>
      </div>
    </div>
  )
}
