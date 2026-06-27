/**
 * Step7_Soil.jsx — Step 7 of 9 (OPTIONAL — has Skip button)
 */

import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useOnboarding } from '../../context/OnboardingContext';
import { addSoil } from '../../api/soil';
import StepLayout from './StepLayout';
import SelectField from '../../components/SelectField';
import PrimaryButton from '../../components/PrimaryButton';

const SOIL_OPTIONS = [
  { value: 'black',   label: 'Black Soil (Cotton soil)' },
  { value: 'red',     label: 'Red Soil' },
  { value: 'sandy',   label: 'Sandy Soil' },
  { value: 'mixed',   label: 'Mixed Soil' },
  { value: 'unknown', label: "Don't Know" },
];

export default function Step7_Soil() {
  const navigate = useNavigate();
  const { state, setField, advanceStep } = useOnboarding();
  const [soilType, setSoilType] = useState(state.soil_type || '');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);

  const handleNext = async () => {
    if (!soilType) return setError('Please select a soil type or skip this step.');
    setLoading(true);
    try {
      await addSoil({ farm_id: state.farm_id, soil_type: soilType });
      setField('soil_type', soilType);
      advanceStep(8, { soil_type: soilType });
      navigate('/onboarding/step8');
    } catch { setError('Failed to save. Please try again.'); }
    finally { setLoading(false); }
  };

  const handleSkip = () => {
    advanceStep(8, {});
    navigate('/onboarding/step8');
  };

  return (
    <StepLayout step={7} title="Soil Information" subtitle="This step is optional — you can skip it" onBack={() => navigate('/onboarding/step6')}>
      <div className="space-y-4">
        <div className="bg-amber-50 border border-amber-100 rounded-xl p-3 text-sm text-amber-700">
          Knowing your soil type helps us give better crop health recommendations.
        </div>
        <SelectField label="Soil Type" name="soil_type" value={soilType} onChange={(e) => { setSoilType(e.target.value); setError(''); }} options={SOIL_OPTIONS} placeholder="Select soil type" error={error} />
        <div className="pt-2 space-y-3">
          <PrimaryButton label="Save & Next →" onClick={handleNext} loading={loading} />
          <button onClick={handleSkip} className="w-full text-[#4a5568] text-sm underline underline-offset-2 py-2 hover:text-[#7a90a8] transition-colors">
            Skip this step
          </button>
        </div>
      </div>
    </StepLayout>
  );
}
