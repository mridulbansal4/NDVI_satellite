/**
 * Step5_CropInfo.jsx — Step 5 of 9
 */

import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useOnboarding } from '../../context/OnboardingContext';
import { addCrop } from '../../api/crop';
import StepLayout from './StepLayout';
import InputField from '../../components/InputField';
import SelectField from '../../components/SelectField';
import PrimaryButton from '../../components/PrimaryButton';

const SEASON_OPTIONS = [
  { value: 'kharif', label: 'Kharif (Jun–Oct)' },
  { value: 'rabi',   label: 'Rabi (Nov–Mar)' },
  { value: 'zaid',   label: 'Zaid (Mar–Jun)' },
];
const MONTH_OPTIONS = ['January','February','March','April','May','June','July','August','September','October','November','December'].map((m) => ({ value: m, label: m }));

export default function Step5_CropInfo() {
  const navigate = useNavigate();
  const { state, merge, advanceStep } = useOnboarding();
  const [form, setForm] = useState({ crop_name: state.crop_name||'', crop_variety: state.crop_variety||'', sowing_date: state.sowing_date||'', season: state.season||'kharif', expected_harvest_month: state.expected_harvest_month||'' });
  const [errors, setErrors] = useState({});
  const [loading, setLoading] = useState(false);

  const change = (e) => { setForm((p)=>({...p,[e.target.name]:e.target.value})); setErrors((p)=>({...p,[e.target.name]:''})); };

  const handleNext = async () => {
    const errs = {};
    if (!form.crop_name.trim()) errs.crop_name = 'Crop name is required.';
    if (!form.sowing_date) errs.sowing_date = 'Sowing date is required.';
    if (Object.keys(errs).length) return setErrors(errs);
    setLoading(true);
    try {
      await addCrop({ farm_id: state.farm_id, crop_name: form.crop_name, crop_variety: form.crop_variety||undefined, sowing_date: form.sowing_date, season: form.season, expected_harvest_month: form.expected_harvest_month||undefined });
      merge(form);
      advanceStep(6, { crop_name: form.crop_name });
      navigate('/onboarding/step6');
    } catch { setErrors({ api: 'Failed to save. Please try again.' }); }
    finally { setLoading(false); }
  };

  return (
    <StepLayout step={5} title="Crop Information" subtitle="What are you growing this season?" onBack={() => navigate('/onboarding/step4')}>
      <div className="space-y-4">
        <InputField label="Crop Name" name="crop_name" value={form.crop_name} onChange={change} placeholder="e.g. Wheat, Cotton, Grapes" error={errors.crop_name} required />
        <InputField label="Crop Variety" name="crop_variety" value={form.crop_variety} onChange={change} placeholder="e.g. Thompson Seedless (optional)" hint="Optional" />
        <InputField label="Sowing Date" name="sowing_date" type="date" value={form.sowing_date} onChange={change} error={errors.sowing_date} required />
        <SelectField label="Season" name="season" value={form.season} onChange={change} options={SEASON_OPTIONS} required />
        <SelectField label="Expected Harvest Month" name="expected_harvest_month" value={form.expected_harvest_month} onChange={change} options={MONTH_OPTIONS} placeholder="Select month (optional)" />
        {errors.api && <p className="text-sm text-red-500 text-center">{errors.api}</p>}
        <div className="pt-2"><PrimaryButton label="Next →" onClick={handleNext} loading={loading} /></div>
      </div>
    </StepLayout>
  );
}
