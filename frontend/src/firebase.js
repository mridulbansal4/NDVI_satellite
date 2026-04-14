import { initializeApp, getApps } from 'firebase/app';
import { getAuth } from 'firebase/auth';

/** Strip accidental quotes from .env values (common when copying from console). */
function envStr(key) {
  const v = import.meta.env[key];
  if (v == null || typeof v !== 'string') return '';
  const t = v.trim();
  if ((t.startsWith('"') && t.endsWith('"')) || (t.startsWith("'") && t.endsWith("'"))) {
    return t.slice(1, -1);
  }
  return t;
}

const firebaseConfig = {
  apiKey:            envStr('VITE_FIREBASE_API_KEY'),
  authDomain:        envStr('VITE_FIREBASE_AUTH_DOMAIN'),
  projectId:         envStr('VITE_FIREBASE_PROJECT_ID'),
  storageBucket:     envStr('VITE_FIREBASE_STORAGE_BUCKET'),
  messagingSenderId: envStr('VITE_FIREBASE_MESSAGING_SENDER_ID'),
  appId:             envStr('VITE_FIREBASE_APP_ID'),
};

// Only initialize if a real apiKey exists — prevents crash when .env is not set
let auth = null;

if (firebaseConfig.apiKey && firebaseConfig.apiKey !== 'undefined') {
  try {
    const app = getApps().length
      ? getApps()[0]
      : initializeApp(firebaseConfig);
    auth = getAuth(app);
  } catch (e) {
    console.warn('[Firebase] Init failed — running in demo mode.', e.message);
  }
} else {
  console.info('[Firebase] No API key found — running in demo mode.');
}

export { auth };
