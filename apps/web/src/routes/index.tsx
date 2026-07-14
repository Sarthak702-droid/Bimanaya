import { createFileRoute, Link } from '@tanstack/react-router'
import { TopNavBar } from '../components/layout/TopNavBar'
import { useEffect, useRef } from 'react'
import { gsap } from 'gsap'
import { useAuth } from '../context/AuthContext'
import {
  Brain,
  FileSearch,
  TrendingUp,
  Gavel,
  Shield,
  ShieldCheck,
  Lock,
  Building,
  CheckCircle,
  ArrowRight,
  Zap,
} from 'lucide-react'

export const Route = createFileRoute('/')({
  component: LandingPage,
})

function LandingPage() {
  const { setShowLoginModal, isSignedIn } = useAuth()

  useEffect(() => {
    // GSAP mounting animations
    gsap.fromTo('.gsap-badge', 
      { opacity: 0, y: -20 },
      { opacity: 1, y: 0, duration: 0.8, ease: 'power3.out' }
    )
    gsap.fromTo('.gsap-title', 
      { opacity: 0, y: 40 },
      { opacity: 1, y: 0, duration: 1.2, delay: 0.1, ease: 'power4.out' }
    )
    gsap.fromTo('.gsap-sub', 
      { opacity: 0, y: 30 },
      { opacity: 1, y: 0, duration: 1, delay: 0.3, ease: 'power2.out' }
    )
    gsap.fromTo('.gsap-cta', 
      { opacity: 0, scale: 0.95 },
      { opacity: 1, scale: 1, duration: 0.8, delay: 0.5, ease: 'back.out(1.5)' }
    )

    // GSAP Scroll reveal animation
    const revealElements = document.querySelectorAll('.gsap-reveal')
    const observer = new IntersectionObserver((entries) => {
      entries.forEach(entry => {
        if (entry.isIntersecting) {
          gsap.fromTo(entry.target, 
            { opacity: 0, y: 50 },
            { opacity: 1, y: 0, duration: 0.9, ease: 'power3.out' }
          )
          observer.unobserve(entry.target)
        }
      })
    }, { threshold: 0.12 })

    revealElements.forEach(el => observer.observe(el))

    return () => observer.disconnect()
  }, [])

  return (
    <div style={{ minHeight: '100vh', display: 'flex', flexDirection: 'column', backgroundColor: 'var(--background)' }}>
      <TopNavBar />
      <main style={{ flex: 1 }}>
        <HeroSection onStart={() => setShowLoginModal(true)} isSignedIn={isSignedIn} />
        <TrustBar />
        <ProblemSection />
        <SolutionGrid />
        <CTASection onStart={() => setShowLoginModal(true)} />
      </main>
      <Footer />
    </div>
  )
}

/* ── WebGL Voronoi Canvas Background ──────────────────────────────────── */
const VERT = `#version 300 es
void main(){ vec2 p=vec2((gl_VertexID<<1)&2, gl_VertexID&2); gl_Position=vec4(p*2.0-1.0,0.0,1.0); }`;

const FRAG = `#version 300 es
precision highp float;
out vec4 o;
uniform vec2 u_res; uniform float u_time; uniform vec2 u_mouse;
vec2 hash2(vec2 p){ p=vec2(dot(p,vec2(127.1,311.7)),dot(p,vec2(269.5,183.3))); return fract(sin(p)*43758.5453); }
vec3 pal(float t){ return 0.45+0.35*cos(6.28318*(vec3(0.1,0.55,0.45)*t+vec3(0.0,0.25,0.5))); }
void main(){
  vec2 uv=(gl_FragCoord.xy-0.5*u_res)/u_res.y;
  vec2 m=(u_mouse-0.5)*vec2(u_res.x/u_res.y,1.0);
  float scale=5.0;
  vec2 p=uv*scale;
  vec2 pull=(m*scale - p); float grab=exp(-dot(pull,pull)*0.45);
  p += pull*grab*0.6;

  vec2 g=floor(p), f=fract(p);
  float f1=9.0, f2=9.0; vec2 id;
  for(int y=-1;y<=1;y++) for(int x=-1;x<=1;x++){
    vec2 lp=vec2(float(x),float(y));
    vec2 pt=0.5+0.5*sin(u_time*0.5 + 6.2831*hash2(g+lp));
    vec2 r=lp+pt-f; float d=dot(r,r);
    if(d<f1){ f2=f1; f1=d; id=g+lp; } else if(d<f2){ f2=d; }
  }
  f1=sqrt(f1); f2=sqrt(f2);
  float edge=smoothstep(0.0,0.07,f2-f1);
  float cellKey=fract(sin(dot(id,vec2(12.9,78.2)))*43758.5);
  vec3 cell=pal(cellKey);
  vec3 col=cell*(0.22+0.78*(1.0-f1));
  col=mix(vec3(0.3,1.0,0.64), col, edge);
  col+=vec3(0.3,1.0,0.64)*(1.0-edge)*0.4;
  col*=0.55+0.45*smoothstep(1.3,0.05,length(uv));
  col=col/(col+0.85);
  o=vec4(pow(max(col,0.0),vec3(0.9)),1.0);
}`;

function VoronoiCanvas() {
  const canvasRef = useRef<HTMLCanvasElement>(null)

  useEffect(() => {
    const canvas = canvasRef.current
    if (!canvas) return
    const gl = canvas.getContext('webgl2', { antialias: true })
    if (!gl) return

    function sh(t: number, s: string) {
      const x = gl.createShader(t)
      if (!x) throw new Error('Shader failed')
      gl.shaderSource(x, s)
      gl.compileShader(x)
      if (!gl.getShaderParameter(x, gl.COMPILE_STATUS)) throw gl.getShaderInfoLog(x)
      return x
    }

    const pr = gl.createProgram()
    if (!pr) return
    gl.attachShader(pr, sh(gl.VERTEX_SHADER, VERT))
    gl.attachShader(pr, sh(gl.FRAGMENT_SHADER, FRAG))
    gl.linkProgram(pr)
    if (!gl.getProgramParameter(pr, gl.LINK_STATUS)) throw gl.getProgramInfoLog(pr)
    gl.useProgram(pr)

    const uRes = gl.getUniformLocation(pr, 'u_res')
    const uTime = gl.getUniformLocation(pr, 'u_time')
    const uMouse = gl.getUniformLocation(pr, 'u_mouse')

    let mouse = [0.5, 0.5]
    let target = [0.5, 0.5]

    const onMove = (e: PointerEvent) => {
      const rect = canvas.getBoundingClientRect()
      target = [(e.clientX - rect.left) / rect.width, 1.0 - (e.clientY - rect.top) / rect.height]
    }
    window.addEventListener('pointermove', onMove)

    function resize() {
      if (!canvas) return
      const d = Math.min(window.devicePixelRatio || 1, 2)
      const w = canvas.clientWidth * d
      const h = canvas.clientHeight * d
      if (canvas.width !== w || canvas.height !== h) {
        canvas.width = w
        canvas.height = h
        gl.viewport(0, 0, w, h)
      }
    }
    window.addEventListener('resize', resize)
    resize()

    const t0 = performance.now()
    let frameId: number

    function frame(now: number) {
      resize()
      mouse[0] += (target[0] - mouse[0]) * 0.06
      mouse[1] += (target[1] - mouse[1]) * 0.06
      gl.uniform2f(uRes, canvas.width, canvas.height)
      gl.uniform1f(uTime, (now - t0) / 1000)
      gl.uniform2f(uMouse, mouse[0], mouse[1])
      gl.drawArrays(gl.TRIANGLES, 0, 3)
      frameId = requestAnimationFrame(frame)
    }
    frameId = requestAnimationFrame(frame)

    return () => {
      window.removeEventListener('pointermove', onMove)
      window.removeEventListener('resize', resize)
      cancelAnimationFrame(frameId)
    }
  }, [])

  return (
    <canvas
      ref={canvasRef}
      style={{
        position: 'absolute',
        inset: 0,
        width: '100%',
        height: '100%',
        display: 'block',
        pointerEvents: 'none',
        opacity: 0.18,
        mixBlendMode: 'screen',
      }}
    />
  )
}

/* ── Hero Section ─────────────────────────────────────────────────────── */
function HeroSection({ onStart, isSignedIn }: { onStart: () => void; isSignedIn: boolean }) {
  return (
    <section
      style={{
        position: 'relative',
        paddingTop: 120,
        paddingBottom: 160,
        paddingLeft: 24,
        paddingRight: 24,
        overflow: 'hidden',
        display: 'flex',
        flexDirection: 'column',
        alignItems: 'center',
        justifyContent: 'center',
        minHeight: '85vh',
        borderBottom: '1px solid var(--outline-variant)',
      }}
    >
      <VoronoiCanvas />

      {/* Hero Content */}
      <div style={{ maxWidth: 840, margin: '0 auto', textAlign: 'center', position: 'relative', zIndex: 1 }}>
        {/* Badge */}
        <div
          className="gsap-badge"
          style={{
            display: 'inline-flex',
            alignItems: 'center',
            gap: 10,
            padding: '8px 20px',
            borderRadius: 'var(--radius-full)',
            background: 'var(--surface-container-low)',
            border: '1px solid var(--outline-variant)',
            fontSize: 11,
            fontWeight: 700,
            letterSpacing: '0.2em',
            textTransform: 'uppercase',
            color: 'var(--accent)',
            marginBottom: 32,
            boxShadow: '0 0 20px rgba(77, 255, 163, 0.05)',
          }}
        >
          <span
            style={{
              width: 8,
              height: 8,
              borderRadius: '50%',
              background: 'var(--accent)',
              boxShadow: '0 0 10px var(--accent)',
            }}
          />
          Next-Gen AI Dispute Resolution
        </div>

        {/* Headline */}
        <h1
          className="gsap-title"
          style={{
            fontFamily: 'var(--font-headline)',
            fontSize: 'clamp(40px, 7vw, 84px)',
            fontWeight: 800,
            lineHeight: 0.95,
            letterSpacing: '-0.04em',
            color: 'var(--on-background)',
            marginBottom: 24,
          }}
        >
          JUSTICE FOR YOUR
          <br />
          <span
            style={{
              color: 'transparent',
              WebkitTextStroke: '1px var(--on-background)',
              marginRight: 12,
            }}
          >
            INSURANCE
          </span>
          CLAIMS
        </h1>

        {/* Subtitle */}
        <p
          className="gsap-sub"
          style={{
            fontSize: 'clamp(15px, 2vw, 18px)',
            color: 'var(--on-surface-variant)',
            marginBottom: 48,
            maxWidth: 620,
            margin: '0 auto 48px',
            lineHeight: 1.6,
          }}
        >
          BimaNyaya leverages advanced reasoning AI and IRDAI regulatory guidelines to auto-generate
          bulletproof legal representations against unfair claim rejections.
        </p>

        {/* CTA Buttons */}
        <div
          className="gsap-cta"
          style={{
            display: 'flex',
            flexWrap: 'wrap',
            alignItems: 'center',
            justifyContent: 'center',
            gap: 16,
          }}
        >
          {isSignedIn ? (
            <Link
              to="/dashboard"
              className="btn btn-primary btn-lg"
              style={{
                boxShadow: '0 0 24px var(--accent-light)',
                padding: '16px 36px',
                borderRadius: 100,
                fontSize: 14,
                letterSpacing: '0.05em',
                textTransform: 'uppercase',
                fontWeight: 700,
              }}
            >
              <Zap size={18} />
              Open Workspace
            </Link>
          ) : (
            <button
              onClick={onStart}
              className="btn btn-primary btn-lg"
              style={{
                boxShadow: '0 0 24px var(--accent-light)',
                padding: '16px 36px',
                borderRadius: 100,
                fontSize: 14,
                letterSpacing: '0.05em',
                textTransform: 'uppercase',
                fontWeight: 700,
                cursor: 'pointer',
              }}
            >
              <Zap size={18} />
              Start Dispute Review
            </button>
          )}
          <Link
            to="/dashboard"
            className="btn btn-secondary btn-lg"
            style={{
              padding: '16px 36px',
              borderRadius: 100,
              fontSize: 14,
              letterSpacing: '0.05em',
              textTransform: 'uppercase',
              fontWeight: 700,
            }}
          >
            Methodology
          </Link>
        </div>
      </div>
    </section>
  )
}

/* ── Trust Bar ────────────────────────────────────────────────────────── */
function TrustBar() {
  const partners = [
    { name: 'Ombudsman Registry', icon: Building },
    { name: 'IRDAI Compliant', icon: Shield },
    { name: 'ISO Secured', icon: ShieldCheck },
    { name: '256-bit Encrypted', icon: Lock },
  ]

  return (
    <section
      style={{
        padding: '32px 24px',
        borderBottom: '1px solid var(--outline-variant)',
        background: 'rgba(5, 8, 7, 0.4)',
        overflow: 'hidden',
      }}
    >
      <div
        style={{
          maxWidth: 1080,
          margin: '0 auto',
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'center',
          flexWrap: 'wrap',
          gap: 24,
        }}
      >
        <span
          style={{
            fontSize: 11,
            fontWeight: 700,
            letterSpacing: '0.15em',
            textTransform: 'uppercase',
            color: 'rgba(234, 255, 244, 0.4)',
          }}
        >
          COMPLIANCE STANDARDS:
        </span>
        <div style={{ display: 'flex', flexWrap: 'wrap', gap: 32 }}>
          {partners.map((p, i) => {
            const Icon = p.icon
            return (
              <div key={i} style={{ display: 'flex', alignItems: 'center', gap: 8, opacity: 0.65 }}>
                <Icon size={16} style={{ color: 'var(--accent)' }} />
                <span style={{ fontSize: 12, fontWeight: 500, letterSpacing: '0.05em', color: 'var(--on-surface)' }}>
                  {p.name}
                </span>
              </div>
            )
          })}
        </div>
      </div>
    </section>
  )
}

/* ── Problem Section ──────────────────────────────────────────────────── */
function ProblemSection() {
  const painPoints = [
    'Ambiguous policy denial codes designed to confuse.',
    'Endless document requests aiming for claimant fatigue.',
    'Inherent power asymmetry against massive carrier legal teams.',
  ]

  return (
    <section className="gsap-reveal" style={{ padding: '120px 24px', position: 'relative', borderBottom: '1px solid var(--outline-variant)' }}>
      <div style={{ maxWidth: 1080, margin: '0 auto' }}>
        <div
          style={{
            display: 'grid',
            gridTemplateColumns: 'repeat(auto-fit, minmax(360px, 1fr))',
            gap: 80,
            alignItems: 'center',
          }}
        >
          {/* Left info */}
          <div>
            <span
              style={{
                fontSize: 11,
                fontWeight: 700,
                color: 'var(--accent)',
                letterSpacing: '0.2em',
                textTransform: 'uppercase',
                display: 'block',
                marginBottom: 16,
              }}
            >
              The Systemic Challenge
            </span>
            <h2
              style={{
                fontFamily: 'var(--font-headline)',
                fontSize: 'clamp(28px, 4vw, 44px)',
                fontWeight: 800,
                lineHeight: 1.05,
                marginBottom: 24,
                letterSpacing: '-0.02em',
                color: 'var(--on-surface)',
              }}
            >
              THE BLACK BOX OF
              <br />
              CLAIM REJECTIONS.
            </h2>
            <p
              style={{
                fontSize: 16,
                color: 'var(--on-surface-variant)',
                lineHeight: 1.7,
                marginBottom: 20,
              }}
            >
              Carriers routinely leverage dense legalese, complex policy structures, and procedural
              exhaustion to reduce payout obligations. Over 68% of policyholders abandon denials simply
              because the appeal process feels impenetrable.
            </p>
            <p
              style={{
                fontSize: 16,
                color: 'var(--on-surface-variant)',
                lineHeight: 1.7,
                marginBottom: 32,
              }}
            >
              BimaNyaya levels the playing field. We translate policy clauses into structural arguments,
              providing legal representation that commands regulatory attention.
            </p>

            <ul style={{ listStyle: 'none', padding: 0, display: 'flex', flexDirection: 'column', gap: 16 }}>
              {painPoints.map((point, index) => (
                <li
                  key={index}
                  style={{
                    display: 'flex',
                    alignItems: 'flex-start',
                    gap: 12,
                    color: 'var(--on-surface)',
                    fontSize: 14,
                  }}
                >
                  <span
                    style={{
                      width: 6,
                      height: 6,
                      borderRadius: '50%',
                      background: 'var(--error)',
                      boxShadow: '0 0 8px var(--error)',
                      marginTop: 8,
                      flexShrink: 0,
                    }}
                  />
                  {point}
                </li>
              ))}
            </ul>
          </div>

          {/* Right visual */}
          <div style={{ position: 'relative' }}>
            <div
              style={{
                position: 'absolute',
                inset: 0,
                background: 'var(--accent-light)',
                borderRadius: 24,
                filter: 'blur(60px)',
                transform: 'rotate(-4deg)',
                opacity: 0.3,
              }}
            />
            <div
              className="glass-card"
              style={{
                position: 'relative',
                padding: 32,
                borderRadius: 24,
                border: '1px solid var(--glass-border)',
                background: 'var(--glass-bg)',
                backdropFilter: 'blur(16px)',
                height: 380,
                display: 'flex',
                flexDirection: 'column',
                justifyContent: 'space-between',
              }}
            >
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                <div style={{ display: 'flex', gap: 6 }}>
                  <span style={{ width: 10, height: 10, borderRadius: '50%', background: '#ff5f56' }} />
                  <span style={{ width: 10, height: 10, borderRadius: '50%', background: '#ffbd2e' }} />
                  <span style={{ width: 10, height: 10, borderRadius: '50%', background: '#27c93f' }} />
                </div>
                <div style={{ fontSize: 11, fontWeight: 700, letterSpacing: '0.1em', color: 'rgba(234,255,244,0.4)' }}>
                  REJECTION_AUDIT.LOG
                </div>
              </div>

              <div style={{ flex: 1, display: 'flex', flexDirection: 'column', justifyContent: 'center', gap: 16 }}>
                <div style={{ fontFamily: 'var(--font-mono)', fontSize: 13, color: 'var(--on-surface-variant)' }}>
                  &gt; Analysing denial code: <span style={{ color: 'var(--error)' }}>PRE_EXISTING_EXCLUSION_4.2</span>
                </div>
                <div
                  style={{
                    padding: 16,
                    background: 'rgba(255, 107, 107, 0.05)',
                    borderRadius: 12,
                    border: '1px solid rgba(255, 107, 107, 0.15)',
                  }}
                >
                  <div style={{ fontWeight: 600, fontSize: 14, color: 'var(--on-surface)', marginBottom: 6 }}>
                    Clause 4.2 Rebuttal Target:
                  </div>
                  <div style={{ fontSize: 12, color: 'var(--on-surface-variant)', lineHeight: 1.5 }}>
                    Carriers must prove diagnosis within 48 months prior to policy inception. No historical records established. IRDAI Section 45 rules apply.
                  </div>
                </div>
              </div>

              <div
                style={{
                  display: 'flex',
                  alignItems: 'center',
                  gap: 8,
                  padding: '10px 16px',
                  background: 'rgba(77, 255, 163, 0.06)',
                  border: '1px solid rgba(77, 255, 163, 0.15)',
                  borderRadius: 8,
                  fontSize: 13,
                  color: 'var(--accent)',
                  fontWeight: 600,
                }}
              >
                <CheckCircle size={16} /> Rebuttal Strategy Formulated
              </div>
            </div>
          </div>
        </div>
      </div>
    </section>
  )
}

/* ── Solution Bento Grid ──────────────────────────────────────────────── */
function SolutionGrid() {
  const cards = [
    {
      icon: Brain,
      title: 'Evidence-Based Representation',
      description:
        'Cross-references policy documentation and medical discharge records with IRDAI rules to auto-generate structured, irrefutable legal rebuttals.',
      span: 2,
    },
    {
      icon: FileSearch,
      title: 'Policy Dissection',
      description:
        'Parses dense 100+ page booklets in seconds to isolate exclusions, definitions, and clauses favorable to your claim dispute.',
      span: 1,
    },
    {
      icon: TrendingUp,
      title: 'Predictive Success Rate',
      description:
        'Calculates resolution probability by comparing case parameters against thousands of historical Ombudsman awards and court judgments.',
      span: 1,
    },
    {
      icon: Gavel,
      title: 'Ombudsman Escalation Files',
      description:
        'Formats documentation automatically into compliant files ready for formal regulatory submission if carrier appeals are stonewalled.',
      span: 2,
    },
  ]

  return (
    <section
      className="gsap-reveal"
      style={{
        padding: '120px 24px',
        background: 'rgba(5, 8, 7, 0.25)',
        borderBottom: '1px solid var(--outline-variant)',
      }}
    >
      <div style={{ maxWidth: 1080, margin: '0 auto' }}>
        {/* Header */}
        <div style={{ textAlign: 'center', marginBottom: 72 }}>
          <span
            style={{
              fontSize: 11,
              fontWeight: 700,
              color: 'var(--accent)',
              letterSpacing: '0.2em',
              textTransform: 'uppercase',
              display: 'block',
              marginBottom: 16,
            }}
          >
            Engineering Resolution
          </span>
          <h2
            style={{
              fontFamily: 'var(--font-headline)',
              fontSize: 'clamp(28px, 4vw, 44px)',
              fontWeight: 800,
              letterSpacing: '-0.02em',
              color: 'var(--on-surface)',
              marginBottom: 16,
            }}
          >
            SYSTEMATIC REASONING.
          </h2>
          <p
            style={{
              fontSize: 16,
              color: 'var(--on-surface-variant)',
              maxWidth: 580,
              margin: '0 auto',
            }}
          >
            Deconstructing carrier rejections through code, evidence parsing, and regulatory compliance rules.
          </p>
        </div>

        {/* Bento Grid */}
        <div
          style={{
            display: 'grid',
            gridTemplateColumns: 'repeat(3, 1fr)',
            gap: 24,
          }}
        >
          {cards.map((card, idx) => {
            const Icon = card.icon
            return (
              <div
                key={idx}
                className="glass-card"
                style={{
                  padding: 32,
                  gridColumn: card.span === 2 ? 'span 2' : 'span 1',
                  borderRadius: 20,
                  border: '1px solid var(--glass-border)',
                  background: 'var(--glass-bg)',
                  display: 'flex',
                  flexDirection: 'column',
                  justifyContent: 'space-between',
                  transition: 'all 0.3s ease',
                }}
                onMouseEnter={(e) => {
                  e.currentTarget.style.borderColor = 'var(--accent)'
                  e.currentTarget.style.boxShadow = '0 0 20px rgba(77, 255, 163, 0.06)'
                  e.currentTarget.style.transform = 'translateY(-2px)'
                }}
                onMouseLeave={(e) => {
                  e.currentTarget.style.borderColor = 'var(--glass-border)'
                  e.currentTarget.style.boxShadow = 'none'
                  e.currentTarget.style.transform = 'none'
                }}
              >
                <div>
                  <div
                    style={{
                      width: 44,
                      height: 44,
                      borderRadius: 12,
                      background: 'var(--accent-light)',
                      display: 'flex',
                      alignItems: 'center',
                      justifyContent: 'center',
                      marginBottom: 24,
                    }}
                  >
                    <Icon size={20} style={{ color: 'var(--accent)' }} />
                  </div>
                  <h3
                    style={{
                      fontFamily: 'var(--font-headline)',
                      fontSize: 18,
                      fontWeight: 700,
                      marginBottom: 12,
                      color: 'var(--on-surface)',
                    }}
                  >
                    {card.title}
                  </h3>
                  <p
                    style={{
                      color: 'var(--on-surface-variant)',
                      fontSize: 14,
                      lineHeight: 1.6,
                    }}
                  >
                    {card.description}
                  </p>
                </div>
              </div>
            )
          })}
        </div>
      </div>
    </section>
  )
}

/* ── CTA Section ──────────────────────────────────────────────────────── */
function CTASection({ onStart }: { onStart: () => void }) {
  return (
    <section
      className="gsap-reveal"
      style={{
        padding: '140px 24px',
        position: 'relative',
        overflow: 'hidden',
        textAlign: 'center',
        borderBottom: '1px solid var(--outline-variant)',
      }}
    >
      <div
        style={{
          position: 'absolute',
          inset: 0,
          background: 'radial-gradient(circle, rgba(77, 255, 163, 0.05) 0%, transparent 60%)',
          pointerEvents: 'none',
        }}
      />
      <div style={{ maxWidth: 640, margin: '0 auto', position: 'relative', zIndex: 1 }}>
        <h2
          style={{
            fontFamily: 'var(--font-headline)',
            fontSize: 'clamp(28px, 5vw, 56px)',
            fontWeight: 800,
            lineHeight: 1,
            color: 'var(--on-surface)',
            marginBottom: 24,
            letterSpacing: '-0.02em',
          }}
        >
          RECOVER WHAT IS
          <br />
          <span style={{ color: 'var(--accent)' }}>LEGALLY YOURS.</span>
        </h2>
        <p
          style={{
            fontSize: 16,
            color: 'var(--on-surface-variant)',
            marginBottom: 40,
            lineHeight: 1.6,
          }}
        >
          No payment info required. Submit your rejection letter to verify claim eligibility under IRDAI.
        </p>
        <button
          onClick={onStart}
          className="btn btn-primary btn-lg"
          style={{
            boxShadow: '0 0 24px var(--accent-light)',
            padding: '16px 36px',
            borderRadius: 100,
            fontSize: 14,
            letterSpacing: '0.05em',
            textTransform: 'uppercase',
            fontWeight: 700,
            cursor: 'pointer',
          }}
        >
          Upload Rejection PDF
          <ArrowRight size={16} />
        </button>
      </div>
    </section>
  )
}

/* ── Footer ───────────────────────────────────────────────────────────── */
function Footer() {
  return (
    <footer
      style={{
        padding: '48px 40px',
        background: '#050807',
        borderTop: '1px solid var(--outline-variant)',
        color: 'var(--on-surface-variant)',
      }}
    >
      <div
        style={{
          maxWidth: 1080,
          margin: '0 auto',
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'center',
          flexWrap: 'wrap',
          gap: 24,
        }}
      >
        <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
          <span style={{ width: 6, height: 6, borderRadius: '50%', background: 'var(--accent)' }} />
          <span style={{ fontSize: 13, fontWeight: 700, color: 'var(--on-surface)', letterSpacing: '0.05em' }}>
            BIMANYAYA
          </span>
        </div>
        <div style={{ display: 'flex', gap: 24, fontSize: 13 }}>
          <a href="#" style={{ color: 'inherit', textDecoration: 'none' }}>Privacy Policy</a>
          <a href="#" style={{ color: 'inherit', textDecoration: 'none' }}>Terms of Service</a>
          <a href="#" style={{ color: 'inherit', textDecoration: 'none' }}>Legal Disclaimer</a>
        </div>
        <span style={{ fontSize: 12 }}>
          © {new Date().getFullYear()} BimaNyaya. All rights reserved.
        </span>
      </div>
    </footer>
  )
}
