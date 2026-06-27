/**
 * OnboardingContext.jsx
 * Global state manager for the multi-step onboarding form.
 * Uses useContext + useReducer.
 * Writes partial_data to Firestore after each step for low-connectivity resume.
 */

import { createContext, useContext, useReducer, useCallback } from 'react';
import { writeSession } from '../utils/firestore';

// ─── Initial state ────────────────────────────────────────────────────────────
const initialState = {
  // Auth
  farmer_id: localStorage.getItem('agri_farmer_id') || null,
  token: localStorage.getItem('agri_token') || null,
  is_new_user: false,

  // Step 1 — Basic Details
  name: '',
  age: '',
  gender: '',
  preferred_language: 'english',

  // Step 2 — Location
  pin_code: '',
  village_name: '',
  full_address: '',
  state: '',
  district: '',
  taluka: '',

  // Step 3+4 — Farm
  farm_id: null,
  farm_name: '',
  total_area: '',
  area_unit: 'acres',
  land_ownership: 'own_land',
  latitude: '',
  longitude: '',
  boundary_geom: null,
  boundary_area_ha: null,
  location_photo_url: '',

  // Step 5 — Crop
  crop_name: '',
  crop_variety: '',
  sowing_date: '',
  season: 'kharif',
  expected_harvest_month: '',

  // Step 6 — Irrigation
  irrigation_type: 'rainfed',
  water_source: '',

  // Step 7 — Soil
  soil_type: '',

  // Nav
  current_step: 1,
};

// ─── Reducer ──────────────────────────────────────────────────────────────────
function reducer(state, action) {
  switch (action.type) {
    case 'SET_AUTH':
      return {
        ...state,
        farmer_id: action.payload.farmer_id,
        token: action.payload.token,
        is_new_user: action.payload.is_new_user,
      };

    case 'SET_FIELD':
      return { ...state, [action.field]: action.value };

    case 'MERGE':
      return { ...state, ...action.payload };

    case 'SET_STEP':
      return { ...state, current_step: action.step };

    case 'RESET':
      return { ...initialState };

    default:
      return state;
  }
}

// ─── Context ──────────────────────────────────────────────────────────────────
const OnboardingContext = createContext(null);

export function OnboardingProvider({ children }) {
  const [state, dispatch] = useReducer(reducer, initialState);

  const setAuth = useCallback((farmer_id, token, is_new_user) => {
    localStorage.setItem('agri_token', token);
    localStorage.setItem('agri_farmer_id', farmer_id);
    dispatch({ type: 'SET_AUTH', payload: { farmer_id, token, is_new_user } });
  }, []);

  const setField = useCallback((field, value) => {
    dispatch({ type: 'SET_FIELD', field, value });
  }, []);

  const merge = useCallback((payload) => {
    dispatch({ type: 'MERGE', payload });
  }, []);

  const advanceStep = useCallback((step, partial_data = {}) => {
    dispatch({ type: 'SET_STEP', step });
    // Write session to Firestore for resume support
    const farmer_id = localStorage.getItem('agri_farmer_id');
    if (farmer_id) {
      writeSession(farmer_id, step, partial_data).catch((err) =>
        console.warn('[Firestore] Session write failed:', err)
      );
    }
  }, []);

  const logout = useCallback(() => {
    localStorage.removeItem('agri_token');
    localStorage.removeItem('agri_farmer_id');
    dispatch({ type: 'RESET' });
  }, []);

  return (
    <OnboardingContext.Provider value={{ state, setAuth, setField, merge, advanceStep, logout }}>
      {children}
    </OnboardingContext.Provider>
  );
}

// ─── Hook ─────────────────────────────────────────────────────────────────────
export function useOnboarding() {
  const ctx = useContext(OnboardingContext);
  if (!ctx) throw new Error('useOnboarding must be used inside <OnboardingProvider>');
  return ctx;
}
