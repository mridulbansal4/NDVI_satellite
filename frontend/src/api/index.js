/**
 * api/index.js — Axios instance with base URL and JWT interceptor.
 */
import axios from 'axios';

const api = axios.create({
  baseURL: import.meta.env.VITE_API_URL || 'http://localhost:5000',
  headers: { 'Content-Type': 'application/json' },
  timeout: 15000,
});

// Attach JWT token from localStorage to every request
api.interceptors.request.use((config) => {
  const token = localStorage.getItem('agri_token');
  if (token) config.headers.Authorization = `Bearer ${token}`;
  return config;
});

export default api;
