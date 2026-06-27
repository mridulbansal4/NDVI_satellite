/** api/consent.js — Step 9 */
import api from './index';

export const submitConsent = (satellite_monitoring) =>
  api.post('/consent', { satellite_monitoring }).then((r) => r.data);
