import { create } from 'zustand';
import { persist } from 'zustand/middleware';
import { authAPI } from '../services/api';
import type { AuthState, User } from '../types';

export const useAuthStore = create<AuthState>()(
  persist(
    (set) => ({
      isAuthenticated: false,
      user: null,
      token: null,

      login: async (username: string, password: string) => {
        try {
          const response = await authAPI.login({ username, password });
          
          // 保存到 localStorage
          localStorage.setItem('token', response.token);
          localStorage.setItem('user', JSON.stringify(response.user));
          
          set({
            isAuthenticated: true,
            user: response.user,
            token: response.token,
          });
        } catch (error: any) {
          throw new Error(error.response?.data?.error || '登录失败');
        }
      },

      logout: () => {
        // 清除 localStorage
        localStorage.removeItem('token');
        localStorage.removeItem('user');
        
        // 调用后端登出 API（可选）
        authAPI.logout().catch(() => {
          // 忽略登出错误
        });
        
        set({
          isAuthenticated: false,
          user: null,
          token: null,
        });
      },

      checkAuth: () => {
        const token = localStorage.getItem('token');
        const userStr = localStorage.getItem('user');
        
        if (token && userStr) {
          try {
            const user: User = JSON.parse(userStr);
            set({
              isAuthenticated: true,
              user,
              token,
            });
          } catch (error) {
            // 数据损坏，清除
            localStorage.removeItem('token');
            localStorage.removeItem('user');
          }
        }
      },
    }),
    {
      name: 'auth-storage',
      partialize: (state) => ({
        isAuthenticated: state.isAuthenticated,
        user: state.user,
        token: state.token,
      }),
    }
  )
);
