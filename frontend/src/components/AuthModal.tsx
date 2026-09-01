import React, { useState } from 'react';
import { apiRequest } from '../services/api';
import { useAuth } from '../context/AuthContext';
import { useToast } from '../context/ToastContext';
import type { AuthResponse } from '../types';

interface AuthModalProps {
  isOpen: boolean;
  onClose: () => void;
}

export const AuthModal: React.FC<AuthModalProps> = ({ isOpen, onClose }) => {
  const { login } = useAuth();
  const { showToast } = useToast();

  const [isRegister, setIsRegister] = useState(false);
  const [name, setName] = useState('');
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [phone, setPhone] = useState('');
  const [address, setAddress] = useState('');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');

  if (!isOpen) return null;

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    setLoading(true);

    const endpoint = isRegister ? '/auth/register' : '/auth/login';
    const payload = isRegister
      ? { name, email, password, phone, address }
      : { email, password };

    const res = await apiRequest<AuthResponse>(endpoint, {
      method: 'POST',
      body: JSON.stringify(payload),
    });

    setLoading(false);

    if (res.success && res.data) {
      login(res.data.user, res.data.token);
      showToast(`Welcome, ${res.data.user.name}!`, 'success');
      onClose();
    } else {
      setError(res.error || 'Authentication failed');
    }
  };

  const handleDemoLogin = async (demoEmail: string, demoPass: string) => {
    setEmail(demoEmail);
    setPassword(demoPass);
    setIsRegister(false);
    setError('');
    setLoading(true);

    const res = await apiRequest<AuthResponse>('/auth/login', {
      method: 'POST',
      body: JSON.stringify({ email: demoEmail, password: demoPass }),
    });

    setLoading(false);

    if (res.success && res.data) {
      login(res.data.user, res.data.token);
      showToast(`Welcome, ${res.data.user.name}!`, 'success');
      onClose();
    } else {
      setError(res.error || 'Demo login failed');
    }
  };

  return (
    <div className="fixed inset-0 bg-slate-900/50 backdrop-blur-xs z-50 flex items-center justify-center p-4">
      <div data-testid="auth-modal" className="bg-white rounded-2xl w-full max-w-md p-6 sm:p-8 relative shadow-xl border border-slate-200 animate-modal">
        {/* Close Button */}
        <button
          data-testid="auth-close"
          onClick={onClose}
          className="absolute top-4 right-4 text-slate-400 hover:text-slate-700 w-8 h-8 rounded-full bg-slate-100 flex items-center justify-center cursor-pointer transition-colors"
        >
          ✕
        </button>

        <h2 className="text-xl font-bold text-slate-900 text-center mb-1">
          {isRegister ? 'Create Account' : 'Sign In'}
        </h2>
        <p className="text-xs text-slate-500 text-center mb-4">
          {isRegister ? 'Register to place orders and manage profile' : 'Sign in to access your saved orders'}
        </p>

        {/* Quick Demo Logins */}
        <div className="bg-slate-50 border border-slate-200 p-3 rounded-xl mb-4 text-xs">
          <span className="font-semibold text-slate-600 block mb-2 text-center uppercase tracking-wider text-[10px]">
            1-Click Demo Login
          </span>
          <div className="grid grid-cols-2 gap-2">
            <button
              type="button"
              data-testid="demo-admin-login"
              onClick={() => handleDemoLogin('admin@gocommerce.com', 'Admin@123')}
              className="bg-purple-50 hover:bg-purple-100 text-purple-700 border border-purple-200 font-semibold py-1.5 px-2.5 rounded-lg text-xs transition-colors cursor-pointer"
            >
              👑 Admin Demo
            </button>
            <button
              type="button"
              data-testid="demo-shopper-login"
              onClick={() => handleDemoLogin('shopper@gocommerce.com', 'Shopper@123')}
              className="bg-emerald-50 hover:bg-emerald-100 text-emerald-700 border border-emerald-200 font-semibold py-1.5 px-2.5 rounded-lg text-xs transition-colors cursor-pointer"
            >
              🛍️ Shopper Demo
            </button>
          </div>
        </div>

        {error && (
          <div className="bg-rose-50 text-rose-700 p-3 rounded-lg border border-rose-200 text-xs font-medium mb-4">
            ⚠️ {error}
          </div>
        )}

        <form onSubmit={handleSubmit} className="space-y-3 text-xs">
          {isRegister && (
            <div>
              <label className="font-medium text-slate-700 block mb-1">Full Name</label>
              <input
                type="text"
                data-testid="auth-name-input"
                className="w-full bg-slate-50 border border-slate-200 rounded-lg px-3 py-2 text-slate-900 outline-none focus:border-blue-600 focus:bg-white"
                placeholder="Full Name"
                value={name}
                onChange={(e) => setName(e.target.value)}
                required
              />
            </div>
          )}

          <div>
            <label className="font-medium text-slate-700 block mb-1">Email</label>
            <input
              type="email"
              data-testid="auth-email-input"
              className="w-full bg-slate-50 border border-slate-200 rounded-lg px-3 py-2 text-slate-900 outline-none focus:border-blue-600 focus:bg-white"
              placeholder="name@example.com"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              required
            />
          </div>

          <div>
            <label className="font-medium text-slate-700 block mb-1">Password</label>
            <input
              type="password"
              data-testid="auth-password-input"
              className="w-full bg-slate-50 border border-slate-200 rounded-lg px-3 py-2 text-slate-900 outline-none focus:border-blue-600 focus:bg-white"
              placeholder="••••••••"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              required
            />
          </div>

          <button
            type="submit"
            data-testid="auth-submit-button"
            disabled={loading}
            className="w-full bg-blue-600 hover:bg-blue-700 text-white font-semibold text-xs py-2.5 rounded-lg shadow-xs cursor-pointer transition-colors mt-2"
          >
            {loading ? 'Processing...' : isRegister ? 'Create Account' : 'Sign In'}
          </button>
        </form>

        <div className="text-center mt-3">
          <button
            type="button"
            className="text-xs text-slate-500 hover:text-blue-600 hover:underline cursor-pointer"
            onClick={() => {
              setIsRegister(!isRegister);
              setError('');
            }}
          >
            {isRegister ? 'Already have an account? Sign In' : "Don't have an account? Sign Up"}
          </button>
        </div>
      </div>
    </div>
  );
};
