import { atom, computed } from 'nanostores';
import type { User } from '../types';
import { apiClient } from '../api/client';

// Auth state atoms
export const $user = atom<User | null>(null);
export const $isAuthenticated = computed($user, (user) => user !== null);
export const $isLoading = atom(false);
export const $authError = atom<string | null>(null);
export const $isInitialized = atom(false);

// Auth actions
export const authActions = {
  async login(username: string, password: string) {
    console.log('Login action started', { username });
    $isLoading.set(true);
    $authError.set(null);

    try {
      const response = await apiClient.login({ username, password });
      console.log('Login successful, setting user:', response.user);
      $user.set(response.user);
      return response;
    } catch (error) {
      console.error('Login error in auth store:', error);
      const message = error instanceof Error ? error.message : 'ログインに失敗しました';
      $authError.set(message);
      throw error;
    } finally {
      $isLoading.set(false);
    }
  },

  async register(username: string, password: string, email?: string) {
    $isLoading.set(true);
    $authError.set(null);

    try {
      const response = await apiClient.register({ username, password, email });
      $user.set(response.user);
      return response;
    } catch (error) {
      const message = error instanceof Error ? error.message : 'アカウント作成に失敗しました';
      $authError.set(message);
      throw error;
    } finally {
      $isLoading.set(false);
    }
  },

  async loadCurrentUser() {
    console.log('Loading current user...');
    if (!apiClient.isAuthenticated()) {
      console.log('No token found, skipping user load');
      $isInitialized.set(true);
      return;
    }

    $isLoading.set(true);
    $authError.set(null);

    try {
      const user = await apiClient.getCurrentUser();
      console.log('Current user loaded:', user);
      $user.set(user);
      return user;
    } catch (error) {
      console.error('Failed to load current user:', error);
      // Token might be invalid, logout
      this.logout();
      const message = error instanceof Error ? error.message : 'ユーザー情報の取得に失敗しました';
      $authError.set(message);
    } finally {
      $isLoading.set(false);
      $isInitialized.set(true);
    }
  },

  logout() {
    console.log('Logout action');
    apiClient.logout();
    $user.set(null);
    $authError.set(null);
    $isInitialized.set(true);
  },

  clearError() {
    $authError.set(null);
  },
};

// Initialize auth state on client
if (typeof window !== 'undefined') {
  console.log('Initializing auth state...');
  authActions.loadCurrentUser();
} 