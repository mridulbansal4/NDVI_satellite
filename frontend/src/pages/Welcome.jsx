/**
 * Welcome.jsx — PRAGYA landing page
 * Professional, no emojis, no gradient cards.
 * Color theme: #1A6B3C
 * Structure: Sticky Nav → Hero → Features Grid → How It Works → Footer
 */

import { useNavigate } from 'react-router-dom';
import { useEffect } from 'react';

// ── Nav ───────────────────────────────────────────────────────────────────────
function Nav() {
  const navigate = useNavigate();
  return (
    <nav className="sticky top-0 z-50 bg-[#111820] border-b border-[#ffffff1a]">
      <div className="max-w-6xl mx-auto px-6 h-16 flex items-center justify-between">
        {/* Wordmark */}
        <div className="flex items-center gap-2.5">
          <svg className="w-7 h-7" fill="none" stroke="#1A6B3C" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.8}
              d="M3.055 11H5a2 2 0 012 2v1a2 2 0 002 2 2 2 0 012 2v2.945M8 3.935V5.5A2.5 2.5 0 0010.5 8h.5a2 2 0 012 2 2 2 0 104 0 2 2 0 012-2h1.064M15 20.488V18a2 2 0 012-2h3.064M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
          </svg>
          <span className="text-lg font-black tracking-tight text-white">PRAGYA</span>
        </div>

        {/* Right nav */}
        <div className="flex items-center gap-3">
          <button
            onClick={() => navigate('/login')}
            className="text-sm font-semibold text-[#7a90a8] hover:text-white transition-colors px-3 py-2"
          >
            Sign In
          </button>
          <button
            onClick={() => navigate('/signup')}
            className="h-9 px-5 rounded-lg text-white text-sm font-bold transition-colors"
            style={{ backgroundColor: '#1A6B3C' }}
            onMouseEnter={(e) => (e.currentTarget.style.backgroundColor = '#155c33')}
            onMouseLeave={(e) => (e.currentTarget.style.backgroundColor = '#1A6B3C')}
          >
            Get Started
          </button>
        </div>
      </div>
    </nav>
  );
}

// ── Hero ──────────────────────────────────────────────────────────────────────
function Hero() {
  const navigate = useNavigate();
  return (
    <section
      className="relative min-h-[88vh] flex flex-col justify-center overflow-hidden"
      style={{ backgroundColor: '#0a1a0f' }}
    >
      {/* Satellite farm background image (via Unsplash) */}
      <div
        className="absolute inset-0 bg-cover bg-center opacity-30"
        style={{
          backgroundImage:
            "url('https://images.unsplash.com/photo-1625246333195-78d9c38ad449?w=1600&q=80')",
        }}
      />
      {/* Dark overlay gradient */}
      <div className="absolute inset-0" style={{ background: 'linear-gradient(to bottom, rgba(10,26,15,0.7) 0%, rgba(10,26,15,0.85) 100%)' }} />

      <div className="relative z-10 max-w-4xl mx-auto px-6 py-20 text-center">
        {/* Badge */}
        <div className="inline-flex items-center gap-2 border border-green-700 rounded-full px-4 py-1.5 mb-8">
          <span className="w-1.5 h-1.5 rounded-full bg-green-400 animate-pulse" />
          <span className="text-xs font-semibold text-green-300 uppercase tracking-widest">
            Powered by Sentinel-2 · Google Earth Engine
          </span>
        </div>

        <h1 className="text-4xl sm:text-5xl font-black text-white leading-tight mb-6">
          Satellite Intelligence<br />
          <span style={{ color: '#4ade80' }}>for Every Indian Farm</span>
        </h1>

        <p className="text-lg text-gray-300 max-w-2xl mx-auto mb-10 leading-relaxed">
          PRAGYA delivers weekly crop health reports using multi-spectral satellite data
          — designed for rural farmers, available in Hindi, Marathi, and English.
        </p>

        {/* CTA buttons */}
        <div className="flex flex-col sm:flex-row gap-4 justify-center">
          <button
            onClick={() => navigate('/signup')}
            className="h-12 px-8 rounded-xl text-white font-bold text-sm transition-all"
            style={{ backgroundColor: '#1A6B3C' }}
            onMouseEnter={(e) => (e.currentTarget.style.backgroundColor = '#155c33')}
            onMouseLeave={(e) => (e.currentTarget.style.backgroundColor = '#1A6B3C')}
          >
            Register Your Farm — Free
          </button>
          <button
            onClick={() => navigate('/login')}
            className="h-12 px-8 rounded-xl font-bold text-sm border border-white/30 text-white hover:bg-[#111820]/10 transition-all"
          >
            Sign In to Dashboard
          </button>
        </div>

        {/* Stats bar */}
        <div className="mt-16 border-t border-white/10 pt-10 grid grid-cols-3 gap-6 max-w-xl mx-auto">
          {[
            { value: '6',   label: 'Vegetation Indices' },
            { value: 'Weekly', label: 'Satellite Updates' },
            { value: '3',   label: 'Languages Supported' },
          ].map(({ value, label }) => (
            <div key={label} className="text-center">
              <p className="text-2xl font-black text-white">{value}</p>
              <p className="text-xs text-[#4a5568] mt-1 leading-tight">{label}</p>
            </div>
          ))}
        </div>
      </div>
    </section>
  );
}

// ── Features grid ─────────────────────────────────────────────────────────────
const FEATURES = [
  {
    label: 'NDVI',
    title: 'Crop Greenness',
    desc:  'Measures photosynthetic activity and biomass density to identify stressed or healthy zones.',
    icon: (
      <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.6}
          d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z" />
      </svg>
    ),
  },
  {
    label: 'EVI',
    title: 'Enhanced Vegetation',
    desc:  'Reduces atmospheric and soil noise for more accurate crop health assessment in dense canopies.',
    icon: (
      <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.6}
          d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z" />
      </svg>
    ),
  },
  {
    label: 'SAVI',
    title: 'Soil-Adjusted Index',
    desc:  'Minimises soil background interference — essential for early-season crops with sparse cover.',
    icon: (
      <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.6}
          d="M19.428 15.428a2 2 0 00-1.022-.547l-2.387-.477a6 6 0 00-3.86.517l-.318.158a6 6 0 01-3.86.517L6.05 15.21a2 2 0 00-1.806.547M8 4h8l-1 1v5.172a2 2 0 00.586 1.414l5 5c1.26 1.26.367 3.414-1.415 3.414H4.828c-1.782 0-2.674-2.154-1.414-3.414l5-5A2 2 0 009 10.172V5L8 4z" />
      </svg>
    ),
  },
  {
    label: 'NDMI',
    title: 'Moisture Stress',
    desc:  'Detects water content in plant tissue — early warning for drought stress and irrigation needs.',
    icon: (
      <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.6}
          d="M12 3v1m0 16v1m9-9h-1M4 12H3m15.364 6.364l-.707-.707M6.343 6.343l-.707-.707m12.728 0l-.707.707M6.343 17.657l-.707.707M16 12a4 4 0 11-8 0 4 4 0 018 0z" />
      </svg>
    ),
  },
  {
    label: 'NDWI',
    title: 'Water Detection',
    desc:  'Identifies waterlogging, surface water bodies, and irrigation channel coverage from orbit.',
    icon: (
      <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.6}
          d="M20.354 15.354A9 9 0 018.646 3.646 9.003 9.003 0 0012 21a9.003 9.003 0 008.354-5.646z" />
      </svg>
    ),
  },
  {
    label: 'GNDVI',
    title: 'Chlorophyll & Nutrients',
    desc:  'Green-band NDVI is highly sensitive to chlorophyll content, indicating nitrogen deficiency.',
    icon: (
      <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.6}
          d="M9.663 17h4.673M12 3v1m6.364 1.636l-.707.707M21 12h-1M4 12H3m3.343-5.657l-.707-.707m2.828 9.9a5 5 0 117.072 0l-.548.547A3.374 3.374 0 0014 18.469V19a2 2 0 11-4 0v-.531c0-.895-.356-1.754-.988-2.386l-.548-.547z" />
      </svg>
    ),
  },
];

function FeaturesGrid() {
  return (
    <section className="bg-[#1a2432] py-20 px-6">
      <div className="max-w-6xl mx-auto">
        <div className="text-center mb-12">
          <p className="text-xs font-bold uppercase tracking-widest mb-2" style={{ color: '#1A6B3C' }}>
            What We Measure
          </p>
          <h2 className="text-3xl font-black text-white">Six Indices. One Complete Picture.</h2>
          <p className="text-[#7a90a8] mt-3 max-w-xl mx-auto text-sm leading-relaxed">
            Each vegetation index reveals a different dimension of crop health — combined into a single
            Crop Vegetation Index (CVI) score delivered weekly.
          </p>
        </div>

        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
          {FEATURES.map((f) => (
            <div
              key={f.label}
              className="bg-[#111820] border border-[#ffffff1a] rounded-xl p-5 hover:border-[#1A6B3C]/50 transition-colors"
            >
              <div
                className="w-10 h-10 rounded-xl flex items-center justify-center mb-4"
                style={{ backgroundColor: '#f0fdf4', color: '#1A6B3C' }}
              >
                {f.icon}
              </div>
              <div className="flex items-center gap-2 mb-2">
                <span
                  className="text-[10px] font-black uppercase tracking-widest px-2 py-0.5 rounded border"
                  style={{ color: '#1A6B3C', borderColor: '#bbf7d0', backgroundColor: '#f0fdf4' }}
                >
                  {f.label}
                </span>
              </div>
              <h3 className="text-sm font-bold text-white mb-1">{f.title}</h3>
              <p className="text-xs text-[#7a90a8] leading-relaxed">{f.desc}</p>
            </div>
          ))}
        </div>
      </div>
    </section>
  );
}

// ── How It Works ──────────────────────────────────────────────────────────────
const STEPS = [
  {
    num: '01',
    title: 'Register & Add Farm',
    desc:  'Create a free account with your mobile number. Fill in basic details and draw your farm boundary on the satellite map.',
    icon: (
      <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.6}
          d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z" />
      </svg>
    ),
  },
  {
    num: '02',
    title: 'Draw Farm Boundary',
    desc:  'Use our interactive satellite map to precisely outline your farm. Area and centroid coordinates are auto-calculated.',
    icon: (
      <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.6}
          d="M9 20l-5.447-2.724A1 1 0 013 16.382V5.618a1 1 0 011.447-.894L9 7m0 13l6-3m-6 3V7m6 10l4.553 2.276A1 1 0 0021 18.382V7.618a1 1 0 00-.553-.894L15 4m0 13V4m0 0L9 7" />
      </svg>
    ),
  },
  {
    num: '03',
    title: 'Receive Weekly Reports',
    desc:  'Our VI Engine processes Sentinel-2 imagery every week and delivers crop health scores directly to your dashboard.',
    icon: (
      <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.6}
          d="M9 17v-2m3 2v-4m3 4v-6m2 10H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
      </svg>
    ),
  },
];

function HowItWorks() {
  return (
    <section className="bg-[#111820] py-20 px-6">
      <div className="max-w-6xl mx-auto">
        <div className="text-center mb-12">
          <p className="text-xs font-bold uppercase tracking-widest mb-2" style={{ color: '#1A6B3C' }}>
            How It Works
          </p>
          <h2 className="text-3xl font-black text-white">Three Steps to Satellite Insights</h2>
        </div>

        <div className="grid grid-cols-1 md:grid-cols-3 gap-6 relative">
          {/* Connector line (desktop) */}
          <div className="hidden md:block absolute top-10 left-[calc(16.67%+1px)] right-[calc(16.67%+1px)] h-px bg-gray-200" />

          {STEPS.map((s, i) => (
            <div key={s.num} className="relative flex flex-col items-center text-center px-4">
              {/* Step number circle */}
              <div
                className="w-20 h-20 rounded-2xl flex items-center justify-center border-2 mb-5 relative z-10 bg-[#111820]"
                style={{ borderColor: '#1A6B3C', color: '#1A6B3C' }}
              >
                {s.icon}
                <span
                  className="absolute -top-2.5 -right-2.5 w-6 h-6 rounded-full text-[10px] font-black text-white flex items-center justify-center"
                  style={{ backgroundColor: '#1A6B3C' }}
                >
                  {i + 1}
                </span>
              </div>
              <h3 className="text-sm font-bold text-white mb-2">{s.title}</h3>
              <p className="text-xs text-[#7a90a8] leading-relaxed max-w-[200px]">{s.desc}</p>
            </div>
          ))}
        </div>
      </div>
    </section>
  );
}

// ── Footer ────────────────────────────────────────────────────────────────────
function Footer() {
  const navigate = useNavigate();
  return (
    <footer className="bg-gray-900 text-white py-12 px-6">
      <div className="max-w-6xl mx-auto flex flex-col md:flex-row items-start justify-between gap-8">
        {/* Brand */}
        <div className="max-w-xs">
          <div className="flex items-center gap-2 mb-3">
            <svg className="w-6 h-6" fill="none" stroke="#4ade80" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.8}
                d="M3.055 11H5a2 2 0 012 2v1a2 2 0 002 2 2 2 0 012 2v2.945M8 3.935V5.5A2.5 2.5 0 0010.5 8h.5a2 2 0 012 2 2 2 0 104 0 2 2 0 012-2h1.064M15 20.488V18a2 2 0 012-2h3.064M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
            </svg>
            <span className="text-base font-black tracking-tight">PRAGYA</span>
          </div>
          <p className="text-xs text-[#4a5568] leading-relaxed">
            Satellite-powered agronomy intelligence for rural India.
            Built for low-connectivity, multi-language environments.
          </p>
        </div>

        {/* Links */}
        <div className="flex gap-12">
          <div>
            <p className="text-[10px] font-bold uppercase tracking-widest text-[#7a90a8] mb-3">Platform</p>
            <ul className="space-y-2">
              {['Sign In', 'Register', 'Dashboard'].map((l) => (
                <li key={l}>
                  <button
                    onClick={() => navigate(l === 'Sign In' ? '/login' : l === 'Register' ? '/signup' : '/dashboard')}
                    className="text-xs text-[#4a5568] hover:text-white transition-colors"
                  >
                    {l}
                  </button>
                </li>
              ))}
            </ul>
          </div>
          <div>
            <p className="text-[10px] font-bold uppercase tracking-widest text-[#7a90a8] mb-3">Technology</p>
            <ul className="space-y-2">
              {['Sentinel-2', 'Google Earth Engine', 'PostGIS', 'React'].map((l) => (
                <li key={l}>
                  <span className="text-xs text-[#4a5568]">{l}</span>
                </li>
              ))}
            </ul>
          </div>
        </div>
      </div>

      <div className="max-w-6xl mx-auto mt-10 pt-6 border-t border-gray-800 flex flex-col sm:flex-row items-center justify-between gap-3">
        <p className="text-[10px] text-[#7a90a8]">
          © 2025 PRAGYA Agronomy Intelligence. All rights reserved.
        </p>
        <p className="text-[10px] text-[#7a90a8]">
          Powered by Sentinel-2 · Google Earth Engine
        </p>
      </div>
    </footer>
  );
}

// ── Page export ───────────────────────────────────────────────────────────────
export default function Welcome() {
  const navigate = useNavigate();

  useEffect(() => {
    if (localStorage.getItem('agri_token')) navigate('/dashboard', { replace: true });
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  return (
    <div style={{ fontFamily: 'Inter, sans-serif' }}>
      <Nav />
      <Hero />
      <FeaturesGrid />
      <HowItWorks />
      <Footer />
    </div>
  );
}
