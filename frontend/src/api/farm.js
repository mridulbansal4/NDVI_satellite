/** api/farm.js — Step 4+5 */
import api from './index';

export const createFarm = (data) =>
  api.post('/farm', data).then((r) => r.data);
