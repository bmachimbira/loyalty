import React, { createContext, useContext, useState, useEffect } from 'react';
import { api } from '@/lib/api';
import { User } from '@/lib/types';

interface AuthContextType {
  user: User | null;
  login: (email: string, password: string) => Promise<void>;
  logout: () => void;
  isAuthenticated: boolean;
  isLoading: boolean;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [user, setUser] = useState<User | null>(null);
  const [isLoading, setIsLoading] = useState(true);

  useEffect(() => {
    // Check for stored token and attempt to restore session
    const token = localStorage.getItem('auth_token');
    if (token) {
      api.setToken(token);

      // Fetch current user profile
      api.auth.me()
        .then((userInfo) => {
          setUser({
            id: userInfo.id,
            email: userInfo.email,
            full_name: userInfo.full_name,
            role: userInfo.role as 'admin' | 'staff',
            tenant_id: userInfo.tenant_id,
          });
          api.setTenantId(userInfo.tenant_id);
        })
        .catch((error) => {
          console.error('Failed to fetch user profile:', error);
          // Token is invalid, clear it
          api.clearAuth();
        })
        .finally(() => {
          setIsLoading(false);
        });
    } else {
      setIsLoading(false);
    }
  }, []);

  const login = async (email: string, password: string) => {
    try {
      const response = await api.auth.login(email, password);
      api.setToken(response.access_token);
      api.setTenantId(response.user.tenant_id);
      setUser(response.user);

      // Optionally store refresh token
      if (response.refresh_token) {
        localStorage.setItem('refresh_token', response.refresh_token);
      }
    } catch (error) {
      console.error('Login failed:', error);
      throw error;
    }
  };

  const logout = () => {
    api.clearAuth();
    localStorage.removeItem('refresh_token');
    setUser(null);
  };

  return (
    <AuthContext.Provider
      value={{
        user,
        login,
        logout,
        isAuthenticated: !!user,
        isLoading
      }}
    >
      {children}
    </AuthContext.Provider>
  );
}

export const useAuth = () => {
  const context = useContext(AuthContext);
  if (!context) {
    throw new Error('useAuth must be used within AuthProvider');
  }
  return context;
};
