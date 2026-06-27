/** api/crop.js — Step 6 */
import api from './index';

export const addCrop = (data) =>
  api.post('/crop', data).then((r) => r.data);
