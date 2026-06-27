/** api/auth.js — POST /auth/login and POST /auth/signup */
import api from './index';

export const login = (mobile_number, password) =>
  api.post('/auth/login', { mobile_number, password }).then((r) => r.data);

export const signup = (mobile_number, password, name = null) =>
  api.post('/auth/signup', { mobile_number, password, ...(name ? { name } : {}) }).then((r) => r.data);
