/**
 * Step3_FarmDetails.jsx — Step 3 of 9
 * Fields: farm_name, total_area, area_unit, land_ownership
 */

import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useOnboarding } from '../../context/OnboardingContext';
import { createFarm } from '../../api/farm';
import StepLayout from './StepLayout';
import InputField from '../../components/InputField';
import SelectField from '../../components/SelectField';
import PrimaryButton from '../../components/PrimaryButton';

const AREA_UNIT_OPTIONS = [
  { value: 'acres', label: 'Acres' },
  { value: 'hectares', label: 'Hectares' },
];
const OWNERSHIP_OPTIONS = [
  { value: 'own_land', label: 'Own Land' },
  { value: 'leased_land', label: 'Leased Land' },
  { value: 'contract_farming', label: 'Contract Farming' },
];

export default function Step3_FarmDetails() {
  const navigate = useNavigate();
  const { state, merge, advanceStep } = useOnboarding();

  const [form, setForm] = useState({
    farm_name: state.farm_name || '',
    total_area: state.total_area || '',
    area_unit: state.area_unit || 'acres',
    land_ownership: state.land_ownership || 'own_land',
  });
  const [errors, setErrors] = useState({});
  const [loading, setLoading] = useState(false);

  const change = (e) => {
    setForm((p) => ({ ...p, [e.target.name]: e.target.value }));
    setErrors((p) => ({ ...p, [e.target.name]: '' }));
  };

  const validate = () => {
    const errs = {};
    if (!form.farm_name.trim()) errs.farm_name = 'Farm name is required.';
    if (!form.total_area || isNaN(form.total_area) || Number(form.total_area) <= 0)
      errs.total_area = 'Enter a valid area greater than 0.';
    return errs;
  };

  const handleNext = async () => {
    const errs = validate();
    if (Object.keys(errs).length) return setErrors(errs);
    setLoading(true);
    try {
      // Step 4 (map) will provide lat/lng — use defaults here, updated in Step 4
      const lat = state.latitude ? parseFloat(state.latitude) : 20.5937;
      const lng = state.longitude ? parseFloat(state.longitude) : 78.9629;

      const result = await createFarm({
        farm_name: form.farm_name,
        total_area: parseFloat(form.total_area),
        area_unit: form.area_unit,
        land_ownership: form.land_ownership,
        latitude: lat,
        longitude: lng,
      });

      merge({ ...form, farm_id: result.id, latitude: lat, longitude: lng });
      advanceStep(4, { farm_name: form.farm_name });
      navigate('/onboarding/step4');
    } catch (err) {
      setErrors({ api: 'Failed to save farm details. Please try again.' });
    } finally {
      setLoading(false);
    }
  };

  return (
    <StepLayout step={3} title="Farm Details" subtitle="Tell us about your farm" onBack={() => navigate('/onboarding/step2')}>
      <div className="space-y-4">
        <InputField
          label="Farm Name"
          name="farm_name"
          value={form.farm_name}
          onChange={change}
          placeholder="e.g. Ramesh's Vineyard"
          error={errors.farm_name}
          required
        />
        <div className="flex gap-3">
          <div className="flex-1">
            <InputField
              label="Total Area"
              name="total_area"
              type="number"
              value={form.total_area}
              onChange={change}
              placeholder="e.g. 2.5"
              error={errors.total_area}
              required
            />
          </div>
          <div className="w-36">
            <SelectField
              label="Unit"
              name="area_unit"
              value={form.area_unit}
              onChange={change}
              options={AREA_UNIT_OPTIONS}
            />
          </div>
        </div>
        <SelectField
          label="Land Ownership"
          name="land_ownership"
          value={form.land_ownership}
          onChange={change}
          options={OWNERSHIP_OPTIONS}
          required
        />

        {errors.api && <p className="text-sm text-red-500 text-center">{errors.api}</p>}
        <div className="pt-2">
          <PrimaryButton label="Next →" onClick={handleNext} loading={loading} />
        </div>
      </div>
    </StepLayout>
  );
}
