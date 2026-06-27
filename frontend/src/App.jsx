import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import { OnboardingProvider } from './context/OnboardingContext';
import Welcome from './pages/Welcome';
import Login from './pages/Login';
import Signup from './pages/Signup';
import Step1_BasicDetails from './pages/steps/Step1_BasicDetails';
import Step2_Location from './pages/steps/Step2_Location';
import Step3_FarmDetails from './pages/steps/Step3_FarmDetails';
import Step4_MapSelection from './pages/steps/Step4_MapSelection';
import Step5_CropInfo from './pages/steps/Step5_CropInfo';
import Step6_Irrigation from './pages/steps/Step6_Irrigation';
import Step7_Soil from './pages/steps/Step7_Soil';
import Step8_Consent from './pages/steps/Step8_Consent';
import Analysis from './pages/Analysis';

function ProtectedRoute({ children }) {
  const token = localStorage.getItem('agri_token');
  if (!token) return <Navigate to="/" replace />;
  return children;
}

export default function App() {
  return (
    <BrowserRouter>
      <OnboardingProvider>
        <Routes>
          {/* Public — Welcome → Login / Signup */}
          <Route path="/" element={<Welcome />} />
          <Route path="/login" element={<Login />} />
          <Route path="/signup" element={<Signup />} />

          {/* Onboarding steps — protected */}
          <Route path="/onboarding/step1" element={<ProtectedRoute><Step1_BasicDetails /></ProtectedRoute>} />
          <Route path="/onboarding/step2" element={<ProtectedRoute><Step2_Location /></ProtectedRoute>} />
          <Route path="/onboarding/step3" element={<ProtectedRoute><Step3_FarmDetails /></ProtectedRoute>} />
          <Route path="/onboarding/step4" element={<ProtectedRoute><Step4_MapSelection /></ProtectedRoute>} />
          <Route path="/onboarding/step5" element={<ProtectedRoute><Step5_CropInfo /></ProtectedRoute>} />
          <Route path="/onboarding/step6" element={<ProtectedRoute><Step6_Irrigation /></ProtectedRoute>} />
          <Route path="/onboarding/step7" element={<ProtectedRoute><Step7_Soil /></ProtectedRoute>} />
          <Route path="/onboarding/step8" element={<ProtectedRoute><Step8_Consent /></ProtectedRoute>} />

          {/* Map & Analysis — protected */}
          <Route path="/dashboard" element={<Navigate to="/analysis" replace />} />
          <Route path="/analysis" element={<ProtectedRoute><Analysis /></ProtectedRoute>} />

          {/* Catch-all */}
          <Route path="*" element={<Navigate to="/" replace />} />
        </Routes>
      </OnboardingProvider>
    </BrowserRouter>
  );
}
