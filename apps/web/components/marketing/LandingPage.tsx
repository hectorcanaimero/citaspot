'use client';

import { useState, useEffect, useCallback } from 'react';
import Link from 'next/link';
import {
  Check, Star, Menu, X, ArrowRight,
  Calendar, Bell, Brain, MessageCircle,
  BarChart3, Sparkles, Clock, Shield, Zap,
} from 'lucide-react';
import { useTranslations } from '@/lib/i18n';

/* ── Iconos de features (orden = t.landing.features) ──── */

const FEATURE_ICONS = [
  <Brain key="brain" size={20} />,
  <Calendar key="cal" size={20} />,
  <Bell key="bell" size={20} />,
  <BarChart3 key="bar" size={20} />,
  <Shield key="shield" size={20} />,
  <Clock key="clock" size={20} />,
];

const STEP_ICONS = [
  <MessageCircle key="msg" size={20} />,
  <Calendar key="cal" size={20} />,
  <Sparkles key="spark" size={20} />,
];

const TESTIMONIAL_META = [
  { initials: 'MG', color: 'linear-gradient(135deg,#C9A96A,#7A5A10)' },
  { initials: 'LR', color: 'linear-gradient(135deg,#6A9AC9,#144B7A)' },
  { initials: 'VM', color: 'linear-gradient(135deg,#C96A9A,#7A1450)' },
];

/* ── Phone mockup ──────────────────────────────────────── */

function PhoneMockup() {
  const t = useTranslations();
  const l = t.landing;
  const [step, setStep] = useState(0);

  const runAnimation = useCallback(() => {
    setStep(0);
    const t1 = setTimeout(() => setStep(1), 700);
    const t2 = setTimeout(() => setStep(2), 1800);
    const t3 = setTimeout(() => setStep(3), 3000);
    const t4 = setTimeout(() => setStep(4), 4000);
    const t5 = setTimeout(() => setStep(5), 5500);
    const t6 = setTimeout(runAnimation, 9500);
    return [t1, t2, t3, t4, t5, t6];
  }, []);

  useEffect(() => {
    const timers = runAnimation();
    return () => timers.forEach(clearTimeout);
  }, [runAnimation]);

  return (
    <div style={{
      width: 272,
      height: 560,
      background: '#0E0E0E',
      borderRadius: 44,
      border: '9px solid #1C1C1E',
      boxShadow: '0 48px 96px rgba(0,0,0,0.65), 0 0 0 1px rgba(255,255,255,0.05), inset 0 0 0 1px rgba(255,255,255,0.04)',
      overflow: 'hidden',
      display: 'flex',
      flexDirection: 'column',
      flexShrink: 0,
      transform: 'rotate(2deg)',
    }}>
      {/* Status bar */}
      <div style={{ background: '#075E54', padding: '8px 16px 4px', display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <span style={{ fontSize: 11, color: 'white', fontWeight: 700 }}>9:41</span>
        <span style={{ fontSize: 10, color: 'rgba(255,255,255,0.8)' }}>●●●</span>
      </div>

      {/* WA header */}
      <div style={{ background: '#075E54', padding: '6px 14px 10px', display: 'flex', alignItems: 'center', gap: 10 }}>
        <div style={{
          width: 36, height: 36, borderRadius: '50%',
          background: 'linear-gradient(135deg,#25D366,#128C7E)',
          display: 'flex', alignItems: 'center', justifyContent: 'center',
          fontSize: 17, flexShrink: 0,
        }}>🤖</div>
        <div>
          <div style={{ fontSize: 13, fontWeight: 600, color: 'white' }}>{l.phoneAgentName}</div>
          <div style={{ fontSize: 10, color: 'rgba(255,255,255,0.7)' }}>{l.phoneOnline}</div>
        </div>
      </div>

      {/* Chat body */}
      <div style={{
        background: '#ECE5DD',
        flex: 1,
        padding: '12px 8px',
        display: 'flex',
        flexDirection: 'column',
        gap: 6,
        overflow: 'hidden',
      }}>
        {/* Msg 1 — out */}
        <div style={{
          maxWidth: '82%', alignSelf: 'flex-end',
          background: '#DCF8C6', borderRadius: '8px 8px 2px 8px',
          padding: '7px 10px', fontSize: 12, color: '#111', lineHeight: 1.4,
          opacity: step >= 1 ? 1 : 0,
          transform: step >= 1 ? 'translateY(0)' : 'translateY(8px)',
          transition: 'opacity 0.3s ease, transform 0.3s ease',
        }}>
          {l.phoneMsg1}
          <div style={{ fontSize: 10, color: '#999', textAlign: 'right', marginTop: 2 }}>9:41 ✓✓</div>
        </div>

        {/* Typing indicator */}
        <div style={{
          alignSelf: 'flex-start',
          background: 'white', borderRadius: '8px 8px 8px 2px',
          padding: '10px 14px', display: 'flex', gap: 4, width: 'fit-content',
          opacity: step === 2 ? 1 : 0,
          transition: 'opacity 0.25s ease',
        }}>
          {[0, 1, 2].map(i => (
            <span key={i} style={{
              width: 6, height: 6, background: '#888', borderRadius: '50%',
              display: 'block',
              animation: step === 2 ? `typing 1.1s ${i * 0.18}s infinite` : 'none',
            }} />
          ))}
        </div>

        {/* Msg 2 — in */}
        <div style={{
          maxWidth: '86%', alignSelf: 'flex-start',
          background: 'white', borderRadius: '8px 8px 8px 2px',
          padding: '7px 10px', fontSize: 12, color: '#111', lineHeight: 1.4,
          opacity: step >= 3 ? 1 : 0,
          transform: step >= 3 ? 'translateY(0)' : 'translateY(8px)',
          transition: 'opacity 0.3s ease, transform 0.3s ease',
        }}>
          {l.phoneMsg2}
          <div style={{ fontSize: 10, color: '#999', textAlign: 'right', marginTop: 2 }}>9:41</div>
        </div>

        {/* Msg 3 — out */}
        <div style={{
          maxWidth: '78%', alignSelf: 'flex-end',
          background: '#DCF8C6', borderRadius: '8px 8px 2px 8px',
          padding: '7px 10px', fontSize: 12, color: '#111', lineHeight: 1.4,
          opacity: step >= 4 ? 1 : 0,
          transform: step >= 4 ? 'translateY(0)' : 'translateY(8px)',
          transition: 'opacity 0.3s ease, transform 0.3s ease',
        }}>
          {l.phoneMsg3}
          <div style={{ fontSize: 10, color: '#999', textAlign: 'right', marginTop: 2 }}>9:42 ✓✓</div>
        </div>

        {/* Msg 4 — in */}
        <div style={{
          maxWidth: '90%', alignSelf: 'flex-start',
          background: 'white', borderRadius: '8px 8px 8px 2px',
          padding: '7px 10px', fontSize: 12, color: '#111', lineHeight: 1.4,
          opacity: step >= 5 ? 1 : 0,
          transform: step >= 5 ? 'translateY(0)' : 'translateY(8px)',
          transition: 'opacity 0.3s ease, transform 0.3s ease',
        }}>
          {l.phoneMsg4}
          <div style={{ fontSize: 10, color: '#999', textAlign: 'right', marginTop: 2 }}>9:42</div>
        </div>
      </div>

      {/* Input bar */}
      <div style={{ background: '#F0F0F0', height: 48, display: 'flex', alignItems: 'center', padding: '0 10px', gap: 8 }}>
        <div style={{ flex: 1, background: 'white', borderRadius: 20, height: 34, padding: '0 12px', display: 'flex', alignItems: 'center', fontSize: 11, color: '#aaa' }}>
          {l.phonePlaceholder}
        </div>
        <div style={{ width: 34, height: 34, borderRadius: '50%', background: '#075E54', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
          <span style={{ fontSize: 14, color: 'white' }}>→</span>
        </div>
      </div>
    </div>
  );
}

/* ── Landing Page ──────────────────────────────────────── */

export default function LandingPage() {
  const t = useTranslations();
  const l = t.landing;
  const [menuOpen, setMenuOpen] = useState(false);
  const [scrolled, setScrolled] = useState(false);

  useEffect(() => {
    const onScroll = () => setScrolled(window.scrollY > 24);
    window.addEventListener('scroll', onScroll);
    return () => window.removeEventListener('scroll', onScroll);
  }, []);

  return (
    <>
      <style dangerouslySetInnerHTML={{ __html: `
        @import url('https://fonts.googleapis.com/css2?family=Cormorant+Garamond:ital,wght@0,500;0,600;0,700;1,500;1,600&family=Plus+Jakarta+Sans:wght@400;500;600;700&display=swap');

        :root {
          --bg: #0C0A08;
          --bg-card: #131009;
          --gold: #C9A96A;
          --gold-lt: #E4C98B;
          --cream: #F2EDE4;
          --muted: #8A7E74;
          --green: #25D366;
          --border: rgba(201,169,106,0.18);
          --border-s: rgba(242,237,228,0.07);
        }

        .lp { background: var(--bg); color: var(--cream); font-family: 'Plus Jakarta Sans', system-ui, sans-serif; min-height: 100vh; overflow-x: hidden; }
        .lp *, .lp *::before, .lp *::after { box-sizing: border-box; }

        /* Grain */
        .lp::before {
          content: '';
          position: fixed; inset: 0; pointer-events: none; z-index: 9999;
          background-image: url("data:image/svg+xml,%3Csvg viewBox='0 0 256 256' xmlns='http://www.w3.org/2000/svg'%3E%3Cfilter id='n'%3E%3CfeTurbulence type='fractalNoise' baseFrequency='0.9' numOctaves='4' stitchTiles='stitch'/%3E%3C/filter%3E%3Crect width='100%25' height='100%25' filter='url(%23n)'/%3E%3C/svg%3E");
          opacity: 0.032;
        }

        /* Type */
        .serif { font-family: 'Cormorant Garamond', Georgia, serif; }
        .hero-h { font-family: 'Cormorant Garamond', Georgia, serif; font-weight: 600; font-size: clamp(58px, 8.5vw, 104px); line-height: 0.93; letter-spacing: -0.02em; }
        .sec-h  { font-family: 'Cormorant Garamond', Georgia, serif; font-weight: 600; font-size: clamp(36px, 4.5vw, 58px); line-height: 1.04; letter-spacing: -0.01em; }
        .gold   { color: var(--gold); }
        .gold-i { color: var(--gold); font-style: italic; }

        /* Nav */
        .nav { position: fixed; top: 0; left: 0; right: 0; z-index: 100; padding: 22px 0; transition: padding .3s, background .3s, border-color .3s; border-bottom: 1px solid transparent; }
        .nav.stuck { padding: 14px 0; background: rgba(12,10,8,.93); backdrop-filter: blur(18px); border-color: var(--border-s); }

        /* Container */
        .c { max-width: 1160px; margin: 0 auto; padding: 0 24px; }

        /* Logo */
        .logo { font-family: 'Cormorant Garamond', Georgia, serif; font-weight: 700; font-size: 26px; color: var(--cream); letter-spacing: -0.01em; text-decoration: none; }
        .logo span { color: var(--gold); }

        /* Buttons */
        .btn-g  { display: inline-flex; align-items: center; gap: 7px; background: transparent; color: var(--muted); font-family: 'Plus Jakarta Sans', sans-serif; font-size: 14px; font-weight: 500; padding: 12px 18px; border-radius: 100px; border: 1px solid var(--border-s); cursor: pointer; text-decoration: none; transition: color .2s, border-color .2s; white-space: nowrap; }
        .btn-g:hover { color: var(--cream); border-color: var(--border); }
        .btn-p  { display: inline-flex; align-items: center; gap: 7px; background: var(--gold); color: #0C0A08; font-family: 'Plus Jakarta Sans', sans-serif; font-size: 14px; font-weight: 700; padding: 13px 26px; border-radius: 100px; border: none; cursor: pointer; text-decoration: none; transition: background .2s, transform .2s, box-shadow .2s; white-space: nowrap; letter-spacing: 0.01em; }
        .btn-p:hover { background: var(--gold-lt); transform: translateY(-1px); box-shadow: 0 8px 28px rgba(201,169,106,.28); }

        /* Badge */
        .pill { display: inline-flex; align-items: center; gap: 7px; background: rgba(201,169,106,.07); border: 1px solid var(--border); color: var(--gold); font-size: 11px; font-weight: 700; letter-spacing: .08em; text-transform: uppercase; padding: 6px 14px; border-radius: 100px; }

        /* Card */
        .card { background: var(--bg-card); border: 1px solid var(--border-s); border-radius: 20px; transition: border-color .25s, transform .25s, box-shadow .25s; }
        .card:hover { border-color: var(--border); transform: translateY(-2px); box-shadow: 0 20px 48px rgba(0,0,0,.38); }

        /* Feature icon */
        .f-icon { width: 44px; height: 44px; border-radius: 12px; background: rgba(201,169,106,.09); border: 1px solid var(--border); display: flex; align-items: center; justify-content: center; color: var(--gold); flex-shrink: 0; }

        /* Check item */
        .chk { display: flex; align-items: flex-start; gap: 10px; font-size: 15px; color: var(--muted); line-height: 1.55; }
        .chk-i { width: 18px; height: 18px; border-radius: 50%; background: rgba(37,211,102,.13); border: 1px solid rgba(37,211,102,.28); display: flex; align-items: center; justify-content: center; flex-shrink: 0; margin-top: 2px; }

        /* Avatar cluster */
        .avs { display: flex; }
        .av  { width: 32px; height: 32px; border-radius: 50%; border: 2px solid var(--bg); margin-left: -10px; font-size: 12px; font-weight: 700; display: flex; align-items: center; justify-content: center; color: #0C0A08; flex-shrink: 0; }
        .avs .av:first-child { margin-left: 0; }

        /* Divider */
        .hr  { height: 1px; background: linear-gradient(90deg, transparent, var(--border-s), transparent); }
        .hr-l { width: 40px; height: 1px; background: linear-gradient(90deg, var(--gold), transparent); }

        /* Stats */
        .stat-n { font-family: 'Cormorant Garamond', Georgia, serif; font-weight: 700; font-size: clamp(48px, 6vw, 72px); line-height: 1; color: var(--gold); }

        /* Step number */
        .step-bg { font-family: 'Cormorant Garamond', Georgia, serif; font-weight: 700; font-size: 88px; line-height: 1; color: var(--gold); opacity: .1; position: absolute; top: -12px; left: -6px; user-select: none; }

        /* Quote */
        .quote { font-family: 'Cormorant Garamond', Georgia, serif; font-size: clamp(19px, 2vw, 24px); font-style: italic; line-height: 1.55; color: var(--cream); }

        /* WA badge */
        .wa-pill { display: inline-flex; align-items: center; gap: 7px; background: rgba(37,211,102,.09); border: 1px solid rgba(37,211,102,.22); color: #4ADE80; font-size: 11px; font-weight: 700; letter-spacing: .07em; text-transform: uppercase; padding: 6px 14px; border-radius: 100px; }
        .wa-dot  { width: 7px; height: 7px; border-radius: 50%; background: var(--green); animation: pulse 2s infinite; flex-shrink: 0; }
        @keyframes pulse { 0%,100% { opacity:1; transform:scale(1); } 50% { opacity:.6; transform:scale(.8); } }

        /* Pricing featured */
        .plan-pro { background: linear-gradient(145deg, rgba(201,169,106,.1), rgba(201,169,106,.04)); border: 1px solid var(--gold); position: relative; border-radius: 20px; }
        .plan-pro::before { content:''; position:absolute; top:0; left:10%; right:10%; height:1px; background: linear-gradient(90deg, transparent, var(--gold-lt), transparent); }

        /* Marquee */
        .marquee-wrap { overflow: hidden; border-top: 1px solid var(--border-s); border-bottom: 1px solid var(--border-s); padding: 14px 0; }
        .marquee-track { display: flex; width: 200%; animation: marquee 22s linear infinite; }
        .marquee-inner { display: flex; gap: 0; white-space: nowrap; width: 50%; }
        @keyframes marquee { from { transform: translateX(0); } to { transform: translateX(-50%); } }

        /* Mobile menu */
        .mob-menu { position: fixed; inset: 0; z-index: 200; background: rgba(12,10,8,.97); display: flex; flex-direction: column; align-items: center; justify-content: center; gap: 36px; }
        .mob-menu a { font-family: 'Cormorant Garamond', Georgia, serif; font-size: 38px; font-weight: 600; color: var(--cream); text-decoration: none; transition: color .2s; }
        .mob-menu a:hover { color: var(--gold); }

        /* CTA radial */
        .cta-glow { position: absolute; inset: 0; background: radial-gradient(ellipse 55% 65% at 50% 50%, rgba(201,169,106,.08) 0%, transparent 70%); pointer-events: none; }

        /* Typing animation */
        @keyframes typing { 0%,100% { transform:translateY(0); opacity:.45; } 50% { transform:translateY(-4px); opacity:1; } }

        /* Hero bg glow */
        .hero-glow { position: absolute; inset: 0; background: radial-gradient(ellipse 70% 55% at 68% 38%, rgba(201,169,106,.07) 0%, transparent 60%), radial-gradient(ellipse 45% 50% at 8% 65%, rgba(37,211,102,.04) 0%, transparent 50%); pointer-events: none; }

        /* Responsive grids */
        .g-3 { display: grid; grid-template-columns: repeat(3,1fr); gap: 20px; }
        .g-2 { display: grid; grid-template-columns: repeat(2,1fr); gap: 24px; }
        .g-stats { display: grid; grid-template-columns: repeat(3,1fr); gap: 24px; text-align: center; }
        .g-hero { display: grid; grid-template-columns: 1fr auto; gap: 48px; align-items: center; }
        .phone-col { display: flex; justify-content: center; }

        @media (max-width: 900px) {
          .g-3 { grid-template-columns: repeat(2,1fr); }
        }
        @media (max-width: 768px) {
          .g-hero { grid-template-columns: 1fr; }
          .phone-col { display: none; }
          .g-2 { grid-template-columns: 1fr; }
          .g-3 { grid-template-columns: 1fr; }
          .nav-links { display: none !important; }
          .mob-btn { display: flex !important; }
        }
        @media (max-width: 540px) {
          .g-stats { grid-template-columns: 1fr; }
        }

        /* Section spacing */
        .sec { padding: 96px 0; }
        @media (max-width: 768px) { .sec { padding: 64px 0; } }
      ` }} />

      <div className="lp">

        {/* ── Mobile menu ──────────────────────────────── */}
        {menuOpen && (
          <div className="mob-menu">
            <button
              onClick={() => setMenuOpen(false)}
              style={{ position: 'absolute', top: 24, right: 24, background: 'none', border: 'none', color: 'var(--muted)', cursor: 'pointer' }}
            >
              <X size={24} />
            </button>
            <a href="#como-funciona" onClick={() => setMenuOpen(false)}>{l.navHowItWorks}</a>
            <a href="#precios" onClick={() => setMenuOpen(false)}>{l.navPricing}</a>
            <Link href="/login" onClick={() => setMenuOpen(false)} style={{ fontSize: 22, fontFamily: 'Plus Jakarta Sans', color: 'var(--muted)' }}>{l.navLogin}</Link>
            <Link href="/register" className="btn-p" onClick={() => setMenuOpen(false)} style={{ fontSize: 15 }}>
              {l.navStartFree} <ArrowRight size={15} />
            </Link>
          </div>
        )}

        {/* ── Nav ──────────────────────────────────────── */}
        <nav className={`nav ${scrolled ? 'stuck' : ''}`}>
          <div className="c" style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
            <Link href="/" className="logo">CitaSpot<span>.</span></Link>

            <div className="nav-links" style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
              <a href="#como-funciona" className="btn-g">{l.navHowItWorks}</a>
              <a href="#precios" className="btn-g">{l.navPricing}</a>
              <Link href="/login" className="btn-g">{l.navLogin}</Link>
              <Link href="/register" className="btn-p">
                {l.navStartFree} <ArrowRight size={14} />
              </Link>
            </div>

            <button
              className="mob-btn"
              onClick={() => setMenuOpen(true)}
              style={{ display: 'none', background: 'none', border: 'none', color: 'var(--cream)', cursor: 'pointer', padding: 4 }}
            >
              <Menu size={22} />
            </button>
          </div>
        </nav>

        {/* ── Hero ─────────────────────────────────────── */}
        <section style={{ paddingTop: 130, paddingBottom: 80, position: 'relative', minHeight: '92vh', display: 'flex', alignItems: 'center' }}>
          <div className="hero-glow" />
          <div className="c" style={{ width: '100%' }}>
            <div className="g-hero">
              <div>
                <div className="wa-pill" style={{ marginBottom: 36 }}>
                  <span className="wa-dot" />
                  {l.heroBadge}
                </div>

                <h1 className="hero-h" style={{ marginBottom: 28 }}>
                  {l.heroTitle1}<br />
                  {l.heroTitle2}<br />
                  <span className="gold-i">{l.heroTitle3}</span>
                </h1>

                <p style={{ fontSize: 18, color: 'var(--muted)', maxWidth: 440, lineHeight: 1.72, marginBottom: 40 }}>
                  {l.heroSub}
                </p>

                <div style={{ display: 'flex', alignItems: 'center', flexWrap: 'wrap', gap: 14, marginBottom: 48 }}>
                  <Link href="/register" className="btn-p" style={{ fontSize: 15, padding: '15px 32px' }}>
                    {l.heroCta} <ArrowRight size={16} />
                  </Link>
                  <a href="#como-funciona" className="btn-g" style={{ fontSize: 15 }}>
                    {l.heroCtaAlt}
                  </a>
                </div>

                {/* Mini social proof */}
                <div style={{ display: 'flex', alignItems: 'center', gap: 16 }}>
                  <div className="avs">
                    {[
                      { l: 'S', c: 'linear-gradient(135deg,#C9A96A,#7A5A10)' },
                      { l: 'M', c: 'linear-gradient(135deg,#6A9AC9,#144B7A)' },
                      { l: 'L', c: 'linear-gradient(135deg,#9AC96A,#4A7A14)' },
                      { l: 'K', c: 'linear-gradient(135deg,#C96A9A,#7A1450)' },
                    ].map((a, i) => (
                      <div key={i} className="av" style={{ background: a.c }}>{a.l}</div>
                    ))}
                  </div>
                  <div>
                    <div style={{ color: 'var(--gold)', fontSize: 13, letterSpacing: 2 }}>★★★★★</div>
                    <div style={{ fontSize: 13, color: 'var(--muted)', marginTop: 2 }}>{l.socialProof}</div>
                  </div>
                </div>
              </div>

              {/* Phone */}
              <div className="phone-col">
                <PhoneMockup />
              </div>
            </div>
          </div>
        </section>

        {/* ── Marquee ──────────────────────────────────── */}
        <div className="marquee-wrap">
          <div className="marquee-track">
            {[0, 1].map(n => (
              <div key={n} className="marquee-inner">
                {(l.marquee as readonly string[]).map((item, i) => (
                  <span key={i} style={{ padding: '0 28px', fontSize: 13, fontWeight: 500, color: i % 3 === 1 ? 'var(--gold)' : 'var(--muted)' }}>
                    {item}
                  </span>
                ))}
              </div>
            ))}
          </div>
        </div>

        {/* ── Stats ────────────────────────────────────── */}
        <div style={{ padding: '56px 0' }}>
          <div className="c">
            <div className="g-stats">
              {(l.stats as readonly { n: string; l: string }[]).map((s, i) => (
                <div key={i}>
                  <div className="stat-n">{s.n}</div>
                  <div style={{ fontSize: 14, color: 'var(--muted)', marginTop: 10, maxWidth: 200, margin: '10px auto 0' }}>{s.l}</div>
                </div>
              ))}
            </div>
          </div>
        </div>

        <div className="hr" />

        {/* ── Cómo funciona ────────────────────────────── */}
        <section className="sec" id="como-funciona">
          <div className="c">
            <div style={{ textAlign: 'center', marginBottom: 72 }}>
              <div className="pill" style={{ marginBottom: 22 }}>
                <Sparkles size={11} /> {l.howPill}
              </div>
              <h2 className="sec-h" style={{ marginBottom: 16 }}>
                {l.howTitle}<span className="gold">{l.howTitleHighlight}</span>
              </h2>
              <p style={{ color: 'var(--muted)', fontSize: 17, maxWidth: 440, margin: '0 auto' }}>
                {l.howSub}
              </p>
            </div>

            <div className="g-3">
              {(l.steps as readonly { title: string; desc: string }[]).map((step, i) => (
                <div key={i} className="card" style={{ padding: 32, position: 'relative', overflow: 'hidden' }}>
                  <div className="step-bg">{i + 1}</div>
                  <div className="f-icon" style={{ marginBottom: 22, position: 'relative', zIndex: 1 }}>{STEP_ICONS[i]}</div>
                  <h3 className="serif" style={{ fontSize: 22, fontWeight: 600, marginBottom: 12, position: 'relative', zIndex: 1, color: 'var(--cream)' }}>
                    {step.title}
                  </h3>
                  <p style={{ color: 'var(--muted)', fontSize: 15, lineHeight: 1.65, position: 'relative', zIndex: 1 }}>
                    {step.desc}
                  </p>
                </div>
              ))}
            </div>
          </div>
        </section>

        <div className="hr" />

        {/* ── Features ─────────────────────────────────── */}
        <section className="sec">
          <div className="c">
            <div style={{ marginBottom: 64 }}>
              <div className="pill" style={{ marginBottom: 22 }}>
                <Zap size={11} /> {l.featuresPill}
              </div>
              <h2 className="sec-h" style={{ maxWidth: 480 }}>
                {l.featuresTitle}<span className="gold">{l.featuresTitleHighlight}</span>
              </h2>
            </div>

            <div className="g-3">
              {(l.features as readonly { title: string; desc: string }[]).map((f, i) => (
                <div key={i} className="card" style={{ padding: 28 }}>
                  <div className="f-icon" style={{ marginBottom: 18 }}>{FEATURE_ICONS[i]}</div>
                  <h3 style={{ fontSize: 17, fontWeight: 600, marginBottom: 9, color: 'var(--cream)' }}>{f.title}</h3>
                  <p style={{ fontSize: 14, color: 'var(--muted)', lineHeight: 1.62 }}>{f.desc}</p>
                </div>
              ))}
            </div>
          </div>
        </section>

        <div className="hr" />

        {/* ── Precios ──────────────────────────────────── */}
        <section className="sec" id="precios">
          <div className="c">
            <div style={{ textAlign: 'center', marginBottom: 64 }}>
              <div className="pill" style={{ marginBottom: 22 }}>
                <Star size={11} /> {l.pricingPill}
              </div>
              <h2 className="sec-h" style={{ marginBottom: 14 }}>
                {l.pricingTitle}<span className="gold">{l.pricingHighlight}</span>{l.pricingTitleSuffix}
              </h2>
              <p style={{ color: 'var(--muted)', fontSize: 17 }}>{l.pricingSub}</p>
            </div>

            <div className="g-2" style={{ maxWidth: 820, margin: '0 auto' }}>
              {/* Starter */}
              <div className="card" style={{ padding: 40 }}>
                <div style={{ fontSize: 12, fontWeight: 700, color: 'var(--muted)', letterSpacing: '.09em', textTransform: 'uppercase', marginBottom: 14 }}>{l.starterName}</div>
                <div style={{ display: 'flex', alignItems: 'flex-end', gap: 4, marginBottom: 8 }}>
                  <span className="serif" style={{ fontSize: 60, fontWeight: 700, color: 'var(--cream)', lineHeight: 1 }}>$50</span>
                  <span style={{ color: 'var(--muted)', marginBottom: 10, fontSize: 15 }}>{l.perMonth}</span>
                </div>
                <p style={{ color: 'var(--muted)', fontSize: 14, marginBottom: 28 }}>{l.starterDesc}</p>
                <div className="hr" style={{ marginBottom: 24 }} />
                <div style={{ display: 'flex', flexDirection: 'column', gap: 13, marginBottom: 32 }}>
                  {(l.starterFeatures as readonly string[]).map((f, i) => (
                    <div key={i} className="chk">
                      <div className="chk-i"><Check size={10} color="#22C55E" strokeWidth={3} /></div>
                      {f}
                    </div>
                  ))}
                </div>
                <Link href="/register" className="btn-g" style={{ display: 'flex', justifyContent: 'center', width: '100%', borderColor: 'var(--border)' }}>
                  {l.startFreeBtn}
                </Link>
              </div>

              {/* Professional */}
              <div className="plan-pro" style={{ padding: 40, position: 'relative', overflow: 'hidden' }}>
                <div style={{ position: 'absolute', top: 16, right: 16, background: 'var(--gold)', color: '#0C0A08', fontSize: 10, fontWeight: 800, letterSpacing: '.07em', textTransform: 'uppercase', padding: '4px 10px', borderRadius: 100 }}>
                  {l.proPopular}
                </div>
                <div style={{ fontSize: 12, fontWeight: 700, color: 'var(--gold)', letterSpacing: '.09em', textTransform: 'uppercase', marginBottom: 14 }}>{l.proName}</div>
                <div style={{ display: 'flex', alignItems: 'flex-end', gap: 4, marginBottom: 8 }}>
                  <span className="serif" style={{ fontSize: 60, fontWeight: 700, color: 'var(--cream)', lineHeight: 1 }}>$100</span>
                  <span style={{ color: 'var(--muted)', marginBottom: 10, fontSize: 15 }}>{l.perMonth}</span>
                </div>
                <p style={{ color: 'var(--muted)', fontSize: 14, marginBottom: 28 }}>{l.proDesc}</p>
                <div className="hr" style={{ marginBottom: 24 }} />
                <div style={{ display: 'flex', flexDirection: 'column', gap: 13, marginBottom: 32 }}>
                  {(l.proFeatures as readonly string[]).map((f, i) => (
                    <div key={i} className="chk">
                      <div className="chk-i"><Check size={10} color="#22C55E" strokeWidth={3} /></div>
                      {f}
                    </div>
                  ))}
                </div>
                <Link href="/register" className="btn-p" style={{ display: 'flex', justifyContent: 'center', width: '100%', fontSize: 15 }}>
                  {l.startFreeBtn} <ArrowRight size={15} />
                </Link>
              </div>
            </div>
          </div>
        </section>

        <div className="hr" />

        {/* ── Testimonios ──────────────────────────────── */}
        <section className="sec">
          <div className="c">
            <div style={{ textAlign: 'center', marginBottom: 64 }}>
              <div className="pill" style={{ marginBottom: 22 }}>
                <Star size={11} /> {l.testimonialsPill}
              </div>
              <h2 className="sec-h">
                {l.testimonialsTitle}<span className="gold-i">{l.testimonialsTitleHighlight}</span>
              </h2>
            </div>

            <div className="g-3">
              {(l.testimonials as readonly { quote: string; name: string; business: string }[]).map((testimonial, i) => (
                <div key={i} className="card" style={{ padding: 32 }}>
                  <div style={{ color: 'var(--gold)', fontSize: 14, letterSpacing: 2, marginBottom: 22 }}>★★★★★</div>
                  <p className="quote" style={{ marginBottom: 28 }}>&ldquo;{testimonial.quote}&rdquo;</p>
                  <div className="hr-l" style={{ marginBottom: 22 }} />
                  <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
                    <div className="av" style={{ background: TESTIMONIAL_META[i].color, width: 40, height: 40, fontSize: 15, marginLeft: 0, flexShrink: 0 }}>{TESTIMONIAL_META[i].initials}</div>
                    <div>
                      <div style={{ fontWeight: 600, fontSize: 14, color: 'var(--cream)' }}>{testimonial.name}</div>
                      <div style={{ fontSize: 13, color: 'var(--muted)', marginTop: 2 }}>{testimonial.business}</div>
                    </div>
                  </div>
                </div>
              ))}
            </div>
          </div>
        </section>

        <div className="hr" />

        {/* ── CTA Final ────────────────────────────────── */}
        <section className="sec" style={{ position: 'relative', overflow: 'hidden' }}>
          <div className="cta-glow" />
          <div className="c" style={{ textAlign: 'center', position: 'relative', zIndex: 1 }}>
            <div className="pill" style={{ marginBottom: 28, display: 'inline-flex' }}>
              <Sparkles size={11} /> {l.ctaPill}
            </div>
            <h2 className="sec-h" style={{ marginBottom: 20, maxWidth: 560, margin: '0 auto 20px' }}>
              {l.ctaTitle}<br />
              <span className="gold-i">{l.ctaTitleHighlight}</span>
            </h2>
            <p style={{ color: 'var(--muted)', fontSize: 18, marginBottom: 40, maxWidth: 440, margin: '0 auto 40px' }}>
              {l.ctaSub}
            </p>
            <div style={{ display: 'flex', justifyContent: 'center', gap: 14, flexWrap: 'wrap' }}>
              <Link href="/register" className="btn-p" style={{ fontSize: 16, padding: '16px 36px' }}>
                {l.ctaBtn} <ArrowRight size={16} />
              </Link>
              <Link href="/login" className="btn-g" style={{ fontSize: 16 }}>
                {l.ctaLoginBtn}
              </Link>
            </div>
          </div>
        </section>

        {/* ── Footer ───────────────────────────────────── */}
        <footer style={{ borderTop: '1px solid var(--border-s)', padding: '36px 0' }}>
          <div className="c" style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', flexWrap: 'wrap', gap: 20 }}>
            <Link href="/" className="logo">CitaSpot<span>.</span></Link>
            <div style={{ display: 'flex', gap: 24, fontSize: 14, color: 'var(--muted)' }}>
              <a href="#" style={{ color: 'inherit', textDecoration: 'none' }}>{l.footerPrivacy}</a>
              <a href="#" style={{ color: 'inherit', textDecoration: 'none' }}>{l.footerTerms}</a>
              <a href="mailto:hola@citaspot.com" style={{ color: 'inherit', textDecoration: 'none' }}>hola@citaspot.com</a>
            </div>
            <span style={{ fontSize: 13, color: 'var(--muted)', opacity: .5 }}>{l.footerCopyright}</span>
          </div>
        </footer>

      </div>
    </>
  );
}
