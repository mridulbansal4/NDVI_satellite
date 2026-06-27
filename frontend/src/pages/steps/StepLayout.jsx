/**
 * StepLayout.jsx — Shared layout wrapper for all onboarding step pages.
 * Renders: ProgressBar (fixed top) + white card + Back button + content.
 */

import ProgressBar from '../../components/ProgressBar';

export default function StepLayout({ step, title, subtitle, onBack, children }) {
  return (
    <div className="min-h-screen bg-[#0a0f14]">
      {/* Fixed progress bar */}
      <ProgressBar currentStep={step} totalSteps={9} />

      {/* Content area — offset for fixed progress bar (~80px) */}
      <div className="pt-20 pb-8 px-4 max-w-md mx-auto">
        {/* Back button */}
        {onBack && (
          <button
            onClick={onBack}
            className="flex items-center gap-1.5 text-[#7a90a8] text-sm mb-3 hover:text-[#e2e8f0] transition-colors"
          >
            <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 19l-7-7 7-7" />
            </svg>
            Back
          </button>
        )}

        {/* White card */}
        <div className="bg-[#111820] rounded-3xl shadow-xl shadow-black/50 border border-[#ffffff1a] p-6">
          {/* Step header */}
          <div className="mb-6">
            <p className="text-xs font-semibold text-[#1A6B3C] uppercase tracking-widest mb-1">
              Step {step} of 9
            </p>
            <h2 className="text-xl font-bold text-white">{title}</h2>
            {subtitle && <p className="text-sm text-[#7a90a8] mt-1">{subtitle}</p>}
          </div>

          {children}
        </div>
      </div>
    </div>
  );
}
