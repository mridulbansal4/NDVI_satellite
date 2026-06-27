/**
 * Step2_Location.jsx — Step 2 of 9
 * Fields: pin_code (auto-resolves state/district/taluka), village_name, full_address
 */

import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useOnboarding } from '../../context/OnboardingContext';
import { saveLocation } from '../../api/farmer';
import StepLayout from './StepLayout';
import InputField from '../../components/InputField';
import PrimaryButton from '../../components/PrimaryButton';

export default function Step2_Location() {
  const navigate = useNavigate();
  const { state, setField, merge, advanceStep } = useOnboarding();

  const [form, setForm] = useState({
    pin_code: state.pin_code || '',
    village_name: state.village_name || '',
    full_address: state.full_address || '',
  });
  const [resolved, setResolved] = useState({
    state: state.state || '',
    district: state.district || '',
    taluka: state.taluka || '',
  });
  const [errors, setErrors] = useState({});
  const [loading, setLoading] = useState(false);
  const [pinLoading, setPinLoading] = useState(false);

  const change = (e) => {
    const { name, value } = e.target;
    setForm((p) => ({ ...p, [name]: value }));
    setErrors((p) => ({ ...p, [name]: '' }));
  };

  // Auto-resolve PIN when 6 digits entered
  const handlePinChange = async (e) => {
    const val = e.target.value.replace(/\D/g, '').slice(0, 6);
    setForm((p) => ({ ...p, pin_code: val }));
    setErrors((p) => ({ ...p, pin_code: '' }));

    if (val.length === 6) {
      setPinLoading(true);
      try {
        // Call our own backend to proxy India Post API (bypasses CORS issues and has 100% rural coverage)
        const res = await fetch(`http://localhost:5000/farmer/pincode/${val}`);
        if (res.ok) {
          const data = await res.json();
          setResolved({ state: data.state, district: data.district, taluka: data.taluka });
        } else {
          setErrors((p) => ({ ...p, pin_code: 'PIN code not found. Please check.' }));
        }
      } catch {
        // Silently fail — backend will resolve it
      } finally {
        setPinLoading(false);
      }
    }
  };

  const validate = () => {
    const errs = {};
    if (!/^\d{6}$/.test(form.pin_code)) errs.pin_code = 'Enter a valid 6-digit PIN code.';
    if (!form.village_name.trim()) errs.village_name = 'Village name is required.';
    return errs;
  };

  const handleNext = async () => {
    const errs = validate();
    if (Object.keys(errs).length) return setErrors(errs);
    setLoading(true);
    try {
      const result = await saveLocation({
        pin_code: form.pin_code,
        village_name: form.village_name,
        full_address: form.full_address || undefined,
      });
      merge({ ...form, state: result.state || resolved.state, district: result.district || resolved.district });
      advanceStep(3, { pin_code: form.pin_code, village_name: form.village_name });
      navigate('/onboarding/step3');
    } catch (err) {
      setErrors({ api: 'Failed to save location. Please try again.' });
    } finally {
      setLoading(false);
    }
  };

  return (
    <StepLayout step={2} title="Your Location" subtitle="Where is your farm located?" onBack={() => navigate('/onboarding/step1')}>
      <div className="space-y-4">
        <div className="relative">
          <InputField
            label="PIN Code"
            name="pin_code"
            type="tel"
            inputMode="numeric"
            value={form.pin_code}
            onChange={handlePinChange}
            placeholder="e.g. 422001"
            error={errors.pin_code}
            hint="State and district will be auto-filled"
            required
          />
          {pinLoading && (
            <div className="absolute right-3 top-9 flex items-center">
              <svg className="animate-spin w-4 h-4 text-green-500" fill="none" viewBox="0 0 24 24">
                <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
                <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
              </svg>
            </div>
          )}
        </div>

        {/* Auto-resolved fields */}
        <div className="grid grid-cols-3 gap-2">
          {[
            { label: 'State', value: resolved.state },
            { label: 'District', value: resolved.district },
            { label: 'Taluka', value: resolved.taluka },
          ].map((f) => (
            <div key={f.label} className="bg-[#1a2432] rounded-xl px-3 py-2.5 border border-[#ffffff1a]">
              <p className="text-xs text-[#4a5568] mb-0.5">{f.label}</p>
              <p className="text-sm font-medium text-[#e2e8f0] truncate">{f.value || '—'}</p>
            </div>
          ))}
        </div>

        <InputField
          label="Village / Town Name"
          name="village_name"
          value={form.village_name}
          onChange={change}
          placeholder="e.g. Pimpalgaon Baswant"
          error={errors.village_name}
          required
        />
        <InputField
          label="Full Address"
          name="full_address"
          value={form.full_address}
          onChange={change}
          placeholder="Optional — house no, street, landmark"
          hint="Optional"
        />

        {errors.api && <p className="text-sm text-red-500 text-center">{errors.api}</p>}

        <div className="pt-2">
          <PrimaryButton label="Next →" onClick={handleNext} loading={loading} />
        </div>
      </div>
    </StepLayout>
  );
}
