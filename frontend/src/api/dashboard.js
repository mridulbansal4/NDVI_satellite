/** api/dashboard.js — GET /dashboard */
import api from './index';

export const fetchDashboard = () =>
  api.get('/dashboard').then((r) => r.data);
