/**
 * Step6_Irrigation.jsx — Step 6 of 9
 */

import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useOnboarding } from '../../context/OnboardingContext';
import { addIrrigation } from '../../api/irrigation';
import StepLayout from './StepLayout';
import SelectField from '../../components/SelectField';
import InputField from '../../components/InputField';
import PrimaryButton from '../../components/PrimaryButton';

const IRRIGATION_OPTIONS = [
  { value: 'rainfed',        label: 'Rainfed' },
  { value: 'borewell',       label: 'Borewell' },
  { value: 'canal',          label: 'Canal' },
  { value: 'drip_irrigation',label: 'Drip Irrigation' },
  { value: 'sprinkler',      label: 'Sprinkler' },
];

export default function Step6_Irrigation() {
  const navigate = useNavigate();
  const { state, merge, advanceStep } = useOnboarding();
  const [form, setForm] = useState({ irrigation_type: state.irrigation_type||'rainfed', water_source: state.water_source||'' });
  const [errors, setErrors] = useState({});
  const [loading, setLoading] = useState(false);

  const change = (e) => { setForm((p)=>({...p,[e.target.name]:e.target.value})); };

  const handleNext = async () => {
    if (!form.irrigation_type) return setErrors({ irrigation_type: 'Select irrigation type.' });
    setLoading(true);
    try {
      await addIrrigation({ farm_id: state.farm_id, irrigation_type: form.irrigation_type, water_source: form.water_source||undefined });
      merge(form);
      advanceStep(7, { irrigation_type: form.irrigation_type });
      navigate('/onboarding/step7');
    } catch { setErrors({ api: 'Failed to save. Please try again.' }); }
    finally { setLoading(false); }
  };

  return (
    <StepLayout step={6} title="Irrigation Method" subtitle="How do you water your farm?" onBack={() => navigate('/onboarding/step5')}>
      <div className="space-y-4">
        <SelectField label="Irrigation Type" name="irrigation_type" value={form.irrigation_type} onChange={change} options={IRRIGATION_OPTIONS} error={errors.irrigation_type} required />
        <InputField label="Water Source" name="water_source" value={form.water_source} onChange={change} placeholder="e.g. Borewell + 2000L tank (optional)" hint="Optional" />
        {errors.api && <p className="text-sm text-red-500 text-center">{errors.api}</p>}
        <div className="pt-2"><PrimaryButton label="Next →" onClick={handleNext} loading={loading} /></div>
      </div>
    </StepLayout>
  );
}
