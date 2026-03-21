import { browser } from '$app/environment';
import { writable } from 'svelte/store';
import type { User } from '$lib/types';
import { api } from '../api/client';

interface AuthState {
	user: User | null;
	isLoading: boolean;
	isAuthenticated: boolean;
}

function createAuthStore() {
	const { subscribe, set, update } = writable<AuthState>({
		user: null,
		isLoading: true,
		isAuthenticated: false,
	});

	return {
		subscribe,
		async initialize() {
			if (!browser) return;
			
			const token = localStorage.getItem('token');
			if (!token) {
				set({ user: null, isLoading: false, isAuthenticated: false });
				return;
			}

			api.setToken(token);
			try {
				const user = await api.getMe();
				set({ user, isLoading: false, isAuthenticated: true });
			} catch {
				localStorage.removeItem('token');
				api.setToken(null);
				set({ user: null, isLoading: false, isAuthenticated: false });
			}
		},
		async login(email: string, password: string) {
			update(s => ({ ...s, isLoading: true }));
			try {
				const response = await api.login(email, password);
				api.setToken(response.token);
				set({ user: response.user, isLoading: false, isAuthenticated: true });
				return { success: true };
			} catch (e) {
				update(s => ({ ...s, isLoading: false }));
				return { success: false, error: (e as Error).message };
			}
		},
		async register(email: string, password: string) {
			update(s => ({ ...s, isLoading: true }));
			try {
				const response = await api.register(email, password);
				api.setToken(response.token);
				set({ user: response.user, isLoading: false, isAuthenticated: true });
				return { success: true };
			} catch (e) {
				update(s => ({ ...s, isLoading: false }));
				return { success: false, error: (e as Error).message };
			}
		},
		logout() {
			api.setToken(null);
			localStorage.removeItem('token');
			set({ user: null, isLoading: false, isAuthenticated: false });
		},
	};
}

export const auth = createAuthStore();