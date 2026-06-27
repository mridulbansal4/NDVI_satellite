/** api/farmer.js — Step 2 + Step 3 */
import api from './index';

export const saveBasicDetails = (data) =>
  api.post('/farmer/basic-details', data).then((r) => r.data);

export const saveLocation = (data) =>
  api.post('/farmer/location', data).then((r) => r.data);
