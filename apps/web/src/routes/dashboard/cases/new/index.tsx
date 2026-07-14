import { createFileRoute, Link } from '@tanstack/react-router'
import { useState } from 'react'
import {
  Heart,
  Shield,
  Car,
  Home,
  ArrowRight,
  X,
  FileText,
  Upload,
  CheckCircle2,
} from 'lucide-react'
import { Logo } from '../../../../components/Logo'

export const Route = createFileRoute('/dashboard/cases/new/')({
  component: EligibilityWizard,
})

const steps = [
  'Claim Type',
  'Claim Details',
  'Documents',
  'Consent',
  'Review',
  'Submit',
]

const claimTypes = [
  {
    icon: Heart,
    label: 'Health & Medical',
    description: 'Hospitalization, treatments, and medical expenses.',
    value: 'health',
  },
  {
    icon: Shield,
    label: 'Life Insurance',
    description: 'Death benefits, critical illness, term policies.',
    value: 'life',
  },
  {
    icon: Car,
    label: 'Motor & Vehicle',
    description: 'Accidents, theft, and third-party liabilities.',
    value: 'motor',
  },
  {
    icon: Home,
    label: 'Property & Fire',
    description: 'Damage to home, business premises, or contents.',
    value: 'property',
  },
]

function EligibilityWizard() {
  const [currentStep, setCurrentStep] = useState(0)
  const [selectedType, setSelectedType] = useState<string | null>(null)

  return (
    <div
      style={{
        position: 'fixed',
        inset: 0,
        zIndex: 100,
        background: 'var(--background)',
        display: 'flex',
        flexDirection: 'column',
      }}
    >
      {/* Top Bar */}
      <div
        style={{
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'center',
          padding: '12px 24px',
          borderBottom: '1px solid var(--outline-variant)',
        }}
      >
        <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
          <Logo size="sm" />
          <span
            style={{
              fontFamily: 'var(--font-headline)',
              fontWeight: 700,
              fontSize: 16,
              color: 'var(--on-surface)',
            }}
          >
            BimaNyaya
          </span>
        </div>
        <Link
          to="/dashboard"
          style={{
            display: 'flex',
            alignItems: 'center',
            gap: 4,
            color: 'var(--on-surface-variant)',
            textDecoration: 'none',
            fontSize: 14,
          }}
        >
          <X size={16} />
          Exit
        </Link>
      </div>

      {/* Progress */}
      <div style={{ padding: '24px 40px 0' }}>
        <div
          style={{
            display: 'flex',
            justifyContent: 'space-between',
            alignItems: 'center',
            marginBottom: 8,
          }}
        >
          <span
            className="text-label-md"
            style={{ color: 'var(--accent)', letterSpacing: '0.05em' }}
          >
            ELIGIBILITY CHECK
          </span>
          <span
            style={{
              fontSize: 13,
              color: 'var(--on-surface-variant)',
            }}
          >
            {currentStep + 1} of {steps.length}
          </span>
        </div>
        <div className="progress-bar">
          <div
            className="progress-bar-fill"
            style={{
              width: `${((currentStep + 1) / steps.length) * 100}%`,
            }}
          />
        </div>
      </div>

      {/* Step Content */}
      <div
        style={{
          flex: 1,
          display: 'flex',
          justifyContent: 'center',
          alignItems: 'flex-start',
          padding: '48px 24px',
          overflowY: 'auto',
        }}
      >
        <div
          className="card animate-fade-in"
          style={{
            maxWidth: 680,
            width: '100%',
            padding: 40,
          }}
        >
          {currentStep === 0 && (
            <>
              <h2
                style={{
                  fontFamily: 'var(--font-headline)',
                  fontSize: 24,
                  fontWeight: 600,
                  marginBottom: 8,
                  color: 'var(--on-surface)',
                }}
              >
                What type of insurance claim is this?
              </h2>
              <p
                style={{
                  color: 'var(--on-surface-variant)',
                  fontSize: 14,
                  marginBottom: 32,
                }}
              >
                Select the primary category of the policy involved.
              </p>

              <div
                style={{
                  display: 'grid',
                  gridTemplateColumns: 'repeat(2, 1fr)',
                  gap: 16,
                }}
              >
                {claimTypes.map((ct) => {
                  const Icon = ct.icon
                  const isSelected = selectedType === ct.value
                  return (
                    <div
                      key={ct.value}
                      className={`radio-card ${isSelected ? 'selected' : ''}`}
                      onClick={() => setSelectedType(ct.value)}
                      style={{ position: 'relative' }}
                    >
                      <div
                        style={{
                          width: 40,
                          height: 40,
                          borderRadius: 'var(--radius-lg)',
                          background: isSelected
                            ? 'var(--accent-light)'
                            : 'var(--surface-container)',
                          display: 'flex',
                          alignItems: 'center',
                          justifyContent: 'center',
                          flexShrink: 0,
                        }}
                      >
                        <Icon
                          size={20}
                          style={{
                            color: isSelected
                              ? 'var(--accent)'
                              : 'var(--on-surface-variant)',
                          }}
                        />
                      </div>
                      <div style={{ flex: 1 }}>
                        <div
                          style={{
                            fontWeight: 600,
                            fontSize: 15,
                            color: 'var(--on-surface)',
                            marginBottom: 4,
                          }}
                        >
                          {ct.label}
                        </div>
                        <div
                          style={{
                            fontSize: 13,
                            color: 'var(--on-surface-variant)',
                          }}
                        >
                          {ct.description}
                        </div>
                      </div>
                      {isSelected && (
                        <div
                          style={{
                            width: 20,
                            height: 20,
                            borderRadius: '50%',
                            border: '2px solid var(--accent)',
                            display: 'flex',
                            alignItems: 'center',
                            justifyContent: 'center',
                            position: 'absolute',
                            top: 16,
                            right: 16,
                          }}
                        >
                          <div
                            style={{
                              width: 10,
                              height: 10,
                              borderRadius: '50%',
                              background: 'var(--accent)',
                            }}
                          />
                        </div>
                      )}
                    </div>
                  )
                })}
              </div>
            </>
          )}

          {currentStep === 1 && (
            <>
              <h2
                style={{
                  fontFamily: 'var(--font-headline)',
                  fontSize: 24,
                  fontWeight: 600,
                  marginBottom: 8,
                  color: 'var(--on-surface)',
                }}
              >
                Claim Details
              </h2>
              <p style={{ color: 'var(--on-surface-variant)', fontSize: 14, marginBottom: 32 }}>
                Provide the basic details of your insurance claim.
              </p>
              <div style={{ display: 'flex', flexDirection: 'column', gap: 20 }}>
                <div>
                  <label className="text-label-md" style={{ display: 'block', marginBottom: 8, color: 'var(--on-surface)' }}>
                    Insurance Company
                  </label>
                  <input className="input" placeholder="e.g. Horizon Health Insurance" />
                </div>
                <div>
                  <label className="text-label-md" style={{ display: 'block', marginBottom: 8, color: 'var(--on-surface)' }}>
                    Policy Number
                  </label>
                  <input className="input" placeholder="e.g. POL-2024-XXXX" />
                </div>
                <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 16 }}>
                  <div>
                    <label className="text-label-md" style={{ display: 'block', marginBottom: 8, color: 'var(--on-surface)' }}>
                      Claimed Amount (₹)
                    </label>
                    <input className="input" type="number" placeholder="e.g. 450000" />
                  </div>
                  <div>
                    <label className="text-label-md" style={{ display: 'block', marginBottom: 8, color: 'var(--on-surface)' }}>
                      Disputed Amount (₹)
                    </label>
                    <input className="input" type="number" placeholder="e.g. 140000" />
                  </div>
                </div>
                <div>
                  <label className="text-label-md" style={{ display: 'block', marginBottom: 8, color: 'var(--on-surface)' }}>
                    Reason for Dispute
                  </label>
                  <textarea
                    className="input"
                    rows={3}
                    placeholder="Briefly describe why you believe the claim was unfairly denied or underpaid..."
                    style={{ resize: 'vertical' }}
                  />
                </div>
              </div>
            </>
          )}

          {currentStep === 2 && (
            <>
              <h2
                style={{
                  fontFamily: 'var(--font-headline)',
                  fontSize: 24,
                  fontWeight: 600,
                  marginBottom: 8,
                  color: 'var(--on-surface)',
                }}
              >
                Upload Documents
              </h2>
              <p style={{ color: 'var(--on-surface-variant)', fontSize: 14, marginBottom: 32 }}>
                Upload your rejection letter, policy schedule, and discharge summary.
              </p>
              <div
                style={{
                  border: '2px dashed var(--outline-variant)',
                  borderRadius: 'var(--radius-xl)',
                  padding: 48,
                  textAlign: 'center',
                  cursor: 'pointer',
                  transition: 'border-color 0.2s',
                }}
                onMouseEnter={(e) => (e.currentTarget.style.borderColor = 'var(--accent)')}
                onMouseLeave={(e) => (e.currentTarget.style.borderColor = 'var(--outline-variant)')}
              >
                <Upload size={40} style={{ color: 'var(--on-surface-variant)', marginBottom: 16 }} />
                <div style={{ fontWeight: 600, fontSize: 16, color: 'var(--on-surface)', marginBottom: 8 }}>
                  Drag & Drop files here
                </div>
                <div style={{ fontSize: 13, color: 'var(--on-surface-variant)', marginBottom: 16 }}>
                  or click to browse from your device
                </div>
                <button className="btn btn-primary">Select Files</button>
                <div style={{ fontSize: 12, color: 'var(--on-surface-variant)', marginTop: 16 }}>
                  Supported formats: PDF, JPG, PNG (Max 10MB per file)
                </div>
              </div>
            </>
          )}

          {currentStep >= 3 && (
            <>
              <div style={{ textAlign: 'center', padding: 40 }}>
                <CheckCircle2 size={64} style={{ color: 'var(--accent)', marginBottom: 24 }} />
                <h2
                  style={{
                    fontFamily: 'var(--font-headline)',
                    fontSize: 24,
                    fontWeight: 600,
                    marginBottom: 8,
                    color: 'var(--on-surface)',
                  }}
                >
                  {currentStep === 3 && 'Review & Consent'}
                  {currentStep === 4 && 'Final Review'}
                  {currentStep === 5 && 'Case Submitted!'}
                </h2>
                <p style={{ color: 'var(--on-surface-variant)', fontSize: 14 }}>
                  {currentStep === 5
                    ? 'Your case has been submitted for analysis. You will receive updates in your dashboard.'
                    : 'Please review the information and confirm to proceed.'}
                </p>
              </div>
            </>
          )}
        </div>
      </div>

      {/* Bottom Action Bar */}
      <div
        style={{
          display: 'flex',
          justifyContent: 'flex-end',
          padding: '16px 40px',
          borderTop: '1px solid var(--outline-variant)',
          background: 'var(--surface-container-lowest)',
        }}
      >
        {currentStep > 0 && currentStep < 5 && (
          <button
            className="btn btn-secondary"
            onClick={() => setCurrentStep(currentStep - 1)}
            style={{ marginRight: 12 }}
          >
            Back
          </button>
        )}
        {currentStep < 5 ? (
          <button
            className="btn btn-primary"
            onClick={() => setCurrentStep(currentStep + 1)}
            disabled={currentStep === 0 && !selectedType}
            style={{
              opacity: currentStep === 0 && !selectedType ? 0.5 : 1,
            }}
          >
            Continue
            <ArrowRight size={16} />
          </button>
        ) : (
          <Link to="/dashboard" className="btn btn-primary">
            Go to Dashboard
            <ArrowRight size={16} />
          </Link>
        )}
      </div>
    </div>
  )
}
