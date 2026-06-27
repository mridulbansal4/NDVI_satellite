/** api/irrigation.js — Step 7 */
import api from './index';

export const addIrrigation = (data) =>
  api.post('/irrigation', data).then((r) => r.data);
