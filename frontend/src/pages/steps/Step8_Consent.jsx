/**
 * Step8_Consent.jsx — Step 8 of 9 (Final step)
 * Checkbox + consent text. Submit button disabled until checked.
 */

import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useOnboarding } from '../../context/OnboardingContext';
import { submitConsent } from '../../api/consent';
import StepLayout from './StepLayout';
import PrimaryButton from '../../components/PrimaryButton';

const CONSENT_TEXT = `By checking this box, I, the farmer, voluntarily give my informed consent to allow the Satellite Agronomy Intelligence Platform to perform satellite-based monitoring of my registered farm(s) using Sentinel-2 imagery and related remote-sensing data.

I understand that:
• My farm boundary coordinates will be used to fetch crop health data.
• All data collected is used solely for agricultural advisory purposes.
• I can withdraw my consent at any time by contacting support.
• My personal information is protected under applicable data privacy laws.`;

export default function Step8_Consent() {
  const navigate = useNavigate();
  const { advanceStep, logout } = useOnboarding();
  const [agreed, setAgreed] = useState(false);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');

  const handleSubmit = async () => {
    if (!agreed) return;
    setLoading(true);
    try {
      await submitConsent(true);
      advanceStep(9, {});
      navigate('/dashboard');
    } catch { setError('Submission failed. Please try again.'); }
    finally { setLoading(false); }
  };

  return (
    <StepLayout step={8} title="Consent & Agreement" subtitle="Last step — review and agree to continue" onBack={() => navigate('/onboarding/step7')}>
      <div className="space-y-5">
        {/* Consent text */}
        <div className="bg-[#1a2432] rounded-xl p-4 border border-[#ffffff1a] max-h-52 overflow-y-auto">
          <p className="text-sm text-[#7a90a8] whitespace-pre-line leading-relaxed">{CONSENT_TEXT}</p>
        </div>

        {/* Checkbox */}
        <label className="flex items-start gap-3 cursor-pointer group">
          <div className={`w-5 h-5 mt-0.5 rounded border-2 flex-shrink-0 flex items-center justify-center transition-colors ${agreed ? 'bg-[#1A6B3C] border-[#1A6B3C]' : 'border-[#ffffff1a] group-hover:border-green-400'}`}>
            {agreed && (
              <svg className="w-3 h-3 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={3} d="M5 13l4 4L19 7" />
              </svg>
            )}
          </div>
          <input type="checkbox" className="sr-only" checked={agreed} onChange={(e) => setAgreed(e.target.checked)} />
          <span className="text-sm text-[#e2e8f0] leading-snug">
            I allow satellite monitoring of my farm to get crop health insights and I agree to the terms above.
          </span>
        </label>

        {error && <p className="text-sm text-red-500 text-center">{error}</p>}

        <div className="pt-1">
          <PrimaryButton
            label="I Agree & Submit ✓"
            onClick={handleSubmit}
            disabled={!agreed}
            loading={loading}
          />
        </div>

        <p className="text-xs text-[#4a5568] text-center">
          Your data is encrypted and never sold to third parties.
        </p>
      </div>
    </StepLayout>
  );
}
