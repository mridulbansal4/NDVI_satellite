/** api/soil.js — Step 8 (optional) */
import api from './index';

export const addSoil = (data) =>
  api.post('/soil', data).then((r) => r.data);
