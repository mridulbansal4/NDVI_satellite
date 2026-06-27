/**
 * Step1_BasicDetails.jsx — Step 1 of 9
 * Fields: name (required), age (optional), gender (optional), preferred_language (required)
 * Language toggle visible on this page.
 */

import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useOnboarding } from '../../context/OnboardingContext';
import { saveBasicDetails } from '../../api/farmer';
import StepLayout from './StepLayout';
import InputField from '../../components/InputField';
import SelectField from '../../components/SelectField';
import PrimaryButton from '../../components/PrimaryButton';

const GENDER_OPTIONS = [
  { value: 'male', label: 'Male' },
  { value: 'female', label: 'Female' },
  { value: 'other', label: 'Other' },
];

const LANG_OPTIONS = [
  { value: 'english', label: 'English' },
  { value: 'hindi', label: 'हिन्दी (Hindi)' },
  { value: 'marathi', label: 'मराठी (Marathi)' },
  { value: 'others', label: 'Other' },
];

export default function Step1_BasicDetails() {
  const navigate = useNavigate();
  const { state, setField, advanceStep } = useOnboarding();

  const [form, setForm] = useState({
    name: state.name || '',
    age: state.age || '',
    gender: state.gender || '',
    preferred_language: state.preferred_language || 'english',
  });
  const [errors, setErrors] = useState({});
  const [loading, setLoading] = useState(false);

  const change = (e) => {
    setForm((p) => ({ ...p, [e.target.name]: e.target.value }));
    setErrors((p) => ({ ...p, [e.target.name]: '' }));
  };

  const validate = () => {
    const errs = {};
    if (!form.name.trim()) errs.name = 'Name is required.';
    if (form.age && (isNaN(form.age) || form.age < 1 || form.age > 120))
      errs.age = 'Enter a valid age (1–120).';
    if (!form.preferred_language) errs.preferred_language = 'Select a language.';
    return errs;
  };

  const handleNext = async () => {
    const errs = validate();
    if (Object.keys(errs).length) return setErrors(errs);
    setLoading(true);
    try {
      await saveBasicDetails({
        name: form.name,
        age: form.age ? parseInt(form.age) : undefined,
        gender: form.gender || undefined,
        preferred_language: form.preferred_language,
      });
      Object.entries(form).forEach(([k, v]) => setField(k, v));
      advanceStep(2, { name: form.name, preferred_language: form.preferred_language });
      navigate('/onboarding/step2');
    } catch (err) {
      setErrors({ api: err.response?.data?.errors ? JSON.stringify(err.response.data.errors) : 'Failed to save. Try again.' });
    } finally {
      setLoading(false);
    }
  };

  return (
    <StepLayout step={1} title="Your Basic Details" subtitle="Tell us a bit about yourself">
      <div className="space-y-4">
        <InputField
          label="Full Name"
          name="name"
          value={form.name}
          onChange={change}
          placeholder="e.g. Ramesh Patil"
          error={errors.name}
          required
        />
        <InputField
          label="Age"
          name="age"
          type="number"
          value={form.age}
          onChange={change}
          placeholder="e.g. 42"
          hint="Optional"
          error={errors.age}
        />
        <SelectField
          label="Gender"
          name="gender"
          value={form.gender}
          onChange={change}
          options={GENDER_OPTIONS}
          placeholder="Select gender (optional)"
        />
        <SelectField
          label="Preferred Language"
          name="preferred_language"
          value={form.preferred_language}
          onChange={change}
          options={LANG_OPTIONS}
          error={errors.preferred_language}
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
