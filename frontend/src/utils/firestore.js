/**
 * utils/firestore.js
 * Client-side Firestore helpers for onboarding session writes.
 * Uses Firebase Web SDK already initialized in firebase.js.
 */

import { initializeApp, getApps } from 'firebase/app';
import { getFirestore, doc, setDoc, getDoc, deleteDoc, serverTimestamp } from 'firebase/firestore';

const firebaseConfig = {
  apiKey:            import.meta.env.VITE_FIREBASE_API_KEY,
  authDomain:        import.meta.env.VITE_FIREBASE_AUTH_DOMAIN,
  projectId:         import.meta.env.VITE_FIREBASE_PROJECT_ID,
  storageBucket:     import.meta.env.VITE_FIREBASE_STORAGE_BUCKET,
  messagingSenderId: import.meta.env.VITE_FIREBASE_MESSAGING_SENDER_ID,
  appId:             import.meta.env.VITE_FIREBASE_APP_ID,
};

let db = null;

function getDB() {
  if (db) return db;
  if (!firebaseConfig.apiKey || firebaseConfig.apiKey === 'undefined') {
    console.info('[Firestore] No API key — session writes disabled.');
    return null;
  }
  const app = getApps().length ? getApps()[0] : initializeApp(firebaseConfig);
  db = getFirestore(app);
  return db;
}

/**
 * Write or merge an onboarding session document.
 * /farmer_sessions/{farmer_id}
 */
export async function writeSession(farmer_id, current_step, partial_data = {}) {
  const firestore = getDB();
  if (!firestore) return;
  const ref = doc(firestore, 'farmer_sessions', farmer_id);
  await setDoc(ref, {
    current_step,
    partial_data,
    last_active: serverTimestamp(),
  }, { merge: true });
}

/**
 * Read an existing onboarding session.
 * Returns the document data or null.
 */
export async function readSession(farmer_id) {
  const firestore = getDB();
  if (!firestore) return null;
  const ref = doc(firestore, 'farmer_sessions', farmer_id);
  const snap = await getDoc(ref);
  return snap.exists() ? snap.data() : null;
}

/**
 * Delete the onboarding session after Step 9 completes.
 */
export async function deleteSession(farmer_id) {
  const firestore = getDB();
  if (!firestore) return;
  await deleteDoc(doc(firestore, 'farmer_sessions', farmer_id));
}
