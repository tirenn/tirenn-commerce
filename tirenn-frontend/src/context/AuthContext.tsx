import React, { createContext, useContext, useState, useEffect } from 'react';
import type { User } from '../types';
import { apiRequest, setAuthToken, getAuthToken } from '../services/api';

interface AuthContextType {
  currentUser: User | null;
  token: string | null;
  loading: boolean;
  login: (user: User, token: string) => void;
  logout: () => void;
  refreshProfile: () => Promise<void>;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

export const AuthProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const [currentUser, setCurrentUser] = useState<User | null>(() => {
    try {
      const saved = localStorage.getItem('tirenn_user');
      return saved ? JSON.parse(saved) : null;
    } catch {
      return null;
    }
  });
  const [token, setToken] = useState<string | null>(getAuthToken());
  const [loading, setLoading] = useState(true);

  const login = (user: User, newToken: string) => {
    setCurrentUser(user);
    setToken(newToken);
    setAuthToken(newToken);
    try {
      localStorage.setItem('tirenn_user', JSON.stringify(user));
    } catch (err) {
      console.error('Failed to save user in storage', err);
    }
  };

  const logout = () => {
    setCurrentUser(null);
    setToken(null);
    setAuthToken(null);
    try {
      localStorage.removeItem('tirenn_user');
    } catch (err) {
      console.error('Failed to remove user from storage', err);
    }
  };

  const refreshProfile = async () => {
    if (!getAuthToken()) {
      setLoading(false);
      return;
    }
    try {
      const res = await apiRequest<User>('/auth/me');
      if (res.success && res.data) {
        setCurrentUser(res.data);
        localStorage.setItem('tirenn_user', JSON.stringify(res.data));
      } else {
        logout();
      }
    } catch {
      logout();
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    refreshProfile();
  }, []);

  return (
    <AuthContext.Provider value={{ currentUser, token, loading, login, logout, refreshProfile }}>
      {children}
    </AuthContext.Provider>
  );
};

export const useAuth = () => {
  const context = useContext(AuthContext);
  if (!context) {
    throw new Error('useAuth must be used within an AuthProvider');
  }
  return context;
};
