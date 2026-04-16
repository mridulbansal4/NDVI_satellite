import React, { useState, useEffect, useRef } from 'react';
import { Leaf, ArrowLeft, Shield, AlertTriangle } from 'lucide-react';
import { RecaptchaVerifier, signInWithPhoneNumber } from 'firebase/auth';
import { auth } from './firebase';
import { apiUrl } from './api';

/* ─────────────────────────────────────────────────
   Inline animation styles injected into <head>
───────────────────────────────────────────────── */
const AUTH_STYLES = `
  @keyframes authShake {
    0%, 100% { transform: translateX(0); }
    10%, 30%, 50%, 70%, 90% { transform: translateX(-5px); }
    20%, 40%, 60%, 80% { transform: translateX(5px); }
  }
  .auth-shake { animation: authShake 0.4s ease-in-out; }

  @keyframes authDrawCheck {
    to { stroke-dashoffset: 0; }
  }
  .auth-check-path {
    stroke-dasharray: 100;
    stroke-dashoffset: 100;
    animation: authDrawCheck 0.8s 0.2s ease-out forwards;
  }
  .auth-check-circle {
    stroke-dasharray: 283;
    stroke-dashoffset: 283;
    animation: authDrawCheck 0.6s ease-out forwards;
  }

  @keyframes authSlideUp {
    from { opacity: 0; transform: translateY(18px); }
    to   { opacity: 1; transform: translateY(0); }
  }
  .auth-screen { animation: authSlideUp 0.35s ease-out both; }

  @keyframes authFadeIn {
    from { opacity: 0; transform: scale(0.96); }
    to   { opacity: 1; transform: scale(1); }
  }
  .auth-fade-in { animation: authFadeIn 0.5s ease-out both; }

  @keyframes authSpinRing {
    to { transform: rotate(360deg); }
  }
  .auth-spin { animation: authSpinRing 0.8s linear infinite; }

  .dark-input:focus {
    outline: none;
    border-color: #22c55e;
    box-shadow: 0 0 0 1px #22c55e;
  }
  
  /* Remove default arrows from number inputs in some browsers */
  input::-webkit-outer-spin-button,
  input::-webkit-inner-spin-button {
    -webkit-appearance: none;
    margin: 0;
  }
  input[type=number] {
    -moz-appearance: textfield;
  }
`;

/* ─────────────────────────────────────────────────
   SVG animated checkmark (Minimal Dark)
───────────────────────────────────────────────── */
function AnimatedCheck() {
  return (
    <div className="relative auth-fade-in" style={{ width: 80, height: 80 }}>
      <svg viewBox="0 0 100 100" fill="none" xmlns="http://www.w3.org/2000/svg"
        className="w-full h-full drop-shadow-md">
        <circle cx="50" cy="50" r="45" stroke="#22c55e" strokeWidth="4"
          className="auth-check-circle" fill="rgba(34,197,94,0.05)" />
        <path d="M30 52 L44 66 L70 36" stroke="#22c55e" strokeWidth="5"
          strokeLinecap="round" strokeLinejoin="round" className="auth-check-path" />
      </svg>
      <div className="absolute inset-0 bg-green-500 opacity-5 rounded-full filter blur-xl" />
    </div>
  );
}

/* ─────────────────────────────────────────────────
   Main Auth Flow Component (Minimal Dark)
───────────────────────────────────────────────── */
export default function PremiumAuthFlow({ onAuthSuccess }) {
  const [screen, setScreen]         = useState('phone');   // phone | otp | success
  const [phone, setPhone]           = useState('');
  const [otp, setOtp]               = useState(['','','','','','']);
  const [activeIdx, setActiveIdx]   = useState(0);
  const [countdown, setCountdown]   = useState(60);
  const [isError, setIsError]       = useState(false);
  const [errorMsg, setErrorMsg]     = useState('');
  const [isLoading, setIsLoading]   = useState(false);
  const [confirmRes, setConfirmRes] = useState(null);  
  const [useDemo, setUseDemo]       = useState(false);

  const otpRefs = useRef([]);

  /* Inject CSS once */
  useEffect(() => {
    const el = document.createElement('style');
    el.textContent = AUTH_STYLES;
    document.head.appendChild(el);
    return () => document.head.removeChild(el);
  }, []);

  /* Check if Firebase is configured */
  useEffect(() => {
    try {
      const cfg = auth?.app?.options;
      if (!cfg?.apiKey || cfg.apiKey === 'undefined') setUseDemo(true);
    } catch { setUseDemo(true); }
  }, []);

  /* OTP countdown timer */
  useEffect(() => {
    if (screen !== 'otp' || countdown <= 0) return;
    const t = setInterval(() => setCountdown(v => v - 1), 1000);
    return () => clearInterval(t);
  }, [screen, countdown]);

  /* Focus first OTP input */
  useEffect(() => {
    if (screen === 'otp') setTimeout(() => otpRefs.current[0]?.focus(), 120);
  }, [screen]);

  const rawPhone = phone.replace(/\s/g, '');
  const otpFull  = otp.join('');

  const formatPhone = (val) => {
    const d = val.replace(/\D/g, '').slice(0, 10);
    if (d.length > 6) return `${d.slice(0,3)} ${d.slice(3,6)} ${d.slice(6)}`;
    if (d.length > 3) return `${d.slice(0,3)} ${d.slice(3)}`;
    return d;
  };

  const goTo = (s) => {
    setScreen(null);
    setTimeout(() => setScreen(s), 60);
  };

  const handleSendOtp = async () => {
    if (rawPhone.length !== 10 || isLoading) return;
    setIsLoading(true);
    setIsError(false);

    if (useDemo) {
      setTimeout(() => {
        setIsLoading(false);
        setOtp(['','','','','','']);
        setCountdown(60);
        setIsError(false);
        goTo('otp');
      }, 900);
      return;
    }

    try {
      if (!auth) {
        setIsError(true);
        setErrorMsg('Firebase is not initialized. Ensure frontend/.env defines VITE_FIREBASE_* and restart the dev server.');
        return;
      }
      if (!window.recaptchaVerifier) {
        window.recaptchaVerifier = new RecaptchaVerifier(auth, 'recaptcha-anchor', {
          size: 'invisible',
        });
      }
      const result = await signInWithPhoneNumber(auth, `+91${rawPhone}`, window.recaptchaVerifier);
      setConfirmRes(result);
      setOtp(['','','','','','']);
      setCountdown(60);
      goTo('otp');
    } catch (err) {
      setIsError(true);
      setErrorMsg(err.message || 'Failed to send OTP.');
      window.recaptchaVerifier?.render().then(id => window.grecaptcha?.reset(id)).catch(() => {});
    } finally {
      setIsLoading(false);
    }
  };

  const handleVerify = async () => {
    if (otpFull.length !== 6 || isLoading) return;
    setIsLoading(true);
    setIsError(false);

    if (useDemo) {
      setTimeout(() => {
        setIsLoading(false);
        goTo('success');
        setTimeout(() => onAuthSuccess?.({ uid: 'demo', phone_number: `+91${rawPhone}` }), 1800);
      }, 1500);
      return;
    }

    try {
      const result = await confirmRes.confirm(otpFull);
      const user = result.user;
      const idToken = await user.getIdToken();

      try {
        await fetch(apiUrl('/api/auth/verify-token'), {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ idToken }),
        });
      } catch {}

      goTo('success');
      setTimeout(() => onAuthSuccess?.({
        uid: user.uid,
        phone_number: user.phoneNumber,
      }), 1800);
    } catch (err) {
      setIsError(true);
      setErrorMsg('Incorrect OTP.');
    } finally {
      setIsLoading(false);
    }
  };

  const handleOtpChange = (e, i) => {
    const v = e.target.value.replace(/\D/g, '').slice(-1);
    const next = [...otp]; next[i] = v; setOtp(next);
    setIsError(false);
    if (v && i < 5) { otpRefs.current[i + 1]?.focus(); setActiveIdx(i + 1); }
  };

  const handleOtpKey = (e, i) => {
    if (e.key === 'Backspace') {
      const next = [...otp];
      if (!next[i] && i > 0) { next[i - 1] = ''; setOtp(next); otpRefs.current[i - 1]?.focus(); setActiveIdx(i - 1); }
      else { next[i] = ''; setOtp(next); }
      setIsError(false);
    } else if (e.key === 'ArrowLeft' && i > 0) { otpRefs.current[i - 1]?.focus(); }
    else if (e.key === 'ArrowRight' && i < 5) { otpRefs.current[i + 1]?.focus(); }
  };

  const handlePaste = (e) => {
    e.preventDefault();
    const d = e.clipboardData.getData('text').replace(/\D/g, '').slice(0, 6);
    const next = [...otp];
    [...d].forEach((c, i) => { next[i] = c; });
    setOtp(next);
    const ni = Math.min(d.length, 5);
    otpRefs.current[ni]?.focus(); setActiveIdx(ni);
  };

  const phoneReady = rawPhone.length === 10 && !isLoading;
  const otpReady = otpFull.length === 6 && !isLoading;

  return (
    <div className="auth-flow">
      <div className="auth-flow__blob auth-flow__blob--tr" aria-hidden />
      <div className="auth-flow__blob auth-flow__blob--bl" aria-hidden />

      {useDemo && (
        <div className="auth-flow__demo" role="status">
          <AlertTriangle size={14} aria-hidden /> Demo mode — Firebase not configured
        </div>
      )}

      <div className="auth-flow__shell">
        <header className="auth-flow__brand">
          <div className="auth-flow__logo" aria-hidden>
            <Leaf style={{ color: 'var(--c-accent)', width: 20, height: 20 }} strokeWidth={2} />
          </div>
          <div className="auth-flow__brand-copy">
            <span className="auth-flow__brand-title">Satellite farm monitoring</span>
            <span className="auth-flow__brand-sub">Vegetation maps and field health analytics</span>
          </div>
        </header>

        <div className="auth-flow__card">
          {screen === 'phone' && (
            <div className="auth-screen">
              <h1 className="auth-flow__h1">Sign in</h1>
              <p className="auth-flow__lead">
                Enter your mobile number. We will send a one-time code to verify it.
              </p>

              <div className="auth-flow__phone-row">
                <div className="auth-flow__cc" aria-hidden>+91</div>
                <input
                  className="auth-flow__tel"
                  type="tel"
                  inputMode="numeric"
                  autoComplete="tel-national"
                  placeholder="98765 43210"
                  value={phone}
                  onChange={(e) => setPhone(formatPhone(e.target.value))}
                  onKeyDown={(e) => e.key === 'Enter' && phoneReady && handleSendOtp()}
                  aria-label="Mobile number"
                />
              </div>

              {isError && <p className="auth-flow__err">{errorMsg}</p>}

              <button
                type="button"
                className={`auth-flow__btn ${phoneReady ? 'auth-flow__btn--primary' : 'auth-flow__btn--muted'}`}
                onClick={handleSendOtp}
                disabled={!phoneReady}
              >
                {isLoading ? (
                  <div
                    className="auth-spin"
                    style={{
                      width: 18,
                      height: 18,
                      border: '2px solid rgba(10,15,20,0.2)',
                      borderTopColor: 'var(--c-bg)',
                      borderRadius: '50%',
                    }}
                  />
                ) : (
                  'Continue'
                )}
              </button>
            </div>
          )}

          {screen === 'otp' && (
            <div className="auth-screen">
              <div className="auth-flow__otp-top">
                <button type="button" className="auth-flow__back" onClick={() => goTo('phone')} aria-label="Back">
                  <ArrowLeft size={18} />
                </button>
                <h2 className="auth-flow__h2">Enter verification code</h2>
              </div>

              <p className="auth-flow__sent-to">
                Code sent to <strong>+91 {phone}</strong>
              </p>

              <div className={`auth-flow__otp-grid ${isError ? 'auth-shake' : ''}`}>
                {otp.map((d, i) => (
                  <input
                    key={i}
                    className={`auth-flow__otp-digit ${isError ? 'is-error' : ''}`}
                    ref={(el) => {
                      otpRefs.current[i] = el;
                    }}
                    type="tel"
                    inputMode="numeric"
                    maxLength={1}
                    value={d}
                    onChange={(e) => handleOtpChange(e, i)}
                    onKeyDown={(e) => handleOtpKey(e, i)}
                    onPaste={handlePaste}
                    onFocus={() => setActiveIdx(i)}
                    aria-label={`Digit ${i + 1}`}
                  />
                ))}
              </div>

              {isError && <p className="auth-flow__err">{errorMsg}</p>}

              <button
                type="button"
                className={`auth-flow__btn ${otpReady ? 'auth-flow__btn--primary' : 'auth-flow__btn--muted'}`}
                onClick={handleVerify}
                disabled={!otpReady}
              >
                {isLoading ? (
                  <div
                    className="auth-spin"
                    style={{
                      width: 18,
                      height: 18,
                      border: '2px solid rgba(10,15,20,0.2)',
                      borderTopColor: 'var(--c-bg)',
                      borderRadius: '50%',
                    }}
                  />
                ) : (
                  'Verify'
                )}
              </button>

              <div className="auth-flow__resend" style={{ marginTop: 16 }}>
                {countdown > 0 ? (
                  <span>Resend code in {String(countdown).padStart(2, '0')}s</span>
                ) : (
                  <button
                    type="button"
                    onClick={() => {
                      setCountdown(60);
                      setOtp(['', '', '', '', '', '']);
                      setIsError(false);
                      handleSendOtp();
                    }}
                  >
                    Resend code
                  </button>
                )}
              </div>
            </div>
          )}

          {screen === 'success' && (
            <div className="auth-flow__success auth-screen">
              <AnimatedCheck />
              <h2>Signed in</h2>
              <p>Opening your dashboard…</p>
            </div>
          )}
        </div>

        <footer className="auth-flow__foot">
          <Shield size={12} aria-hidden /> Phone verification · secure session
        </footer>
      </div>
      <div id="recaptcha-anchor" />
    </div>
  );
}
