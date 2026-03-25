import { browser } from '$app/environment';
import { writable } from 'svelte/store';
import { client, setToken, type User } from '$lib/api/client';

interface AuthState {
	user: User | null;
	isLoading: boolean;
	isAuthenticated: boolean;
}

function createAuthStore() {
	const { subscribe, set, update } = writable<AuthState>({
		user: null,
		isLoading: true,
		isAuthenticated: false
	});

	return {
		subscribe,
		async initialize() {
			if (!browser) return;

			const storedToken = localStorage.getItem('token');
			if (!storedToken) {
				set({ user: null, isLoading: false, isAuthenticated: false });
				return;
			}

			setToken(storedToken);
			try {
				const { data, error } = await client.GET('/auth/me');
				if (error || !data) throw new Error('Failed to fetch user');
				set({ user: data, isLoading: false, isAuthenticated: true });
			} catch {
				localStorage.removeItem('token');
				setToken(null);
				set({ user: null, isLoading: false, isAuthenticated: false });
			}
		},
		async login(email: string, password: string) {
			update((s) => ({ ...s, isLoading: true }));
			try {
				const { data, error } = await client.POST('/auth/login', {
					body: { email, password }
				});
				if (error || !data) throw new Error(error?.detail || 'Login failed');
				setToken(data.token);
				set({ user: data.user, isLoading: false, isAuthenticated: true });
				return { success: true };
			} catch (e) {
				update((s) => ({ ...s, isLoading: false }));
				return { success: false, error: (e as Error).message };
			}
		},
		async register(email: string, password: string) {
			update((s) => ({ ...s, isLoading: true }));
			try {
				const { data, error } = await client.POST('/auth/register', {
					body: { email, password }
				});
				if (error || !data) throw new Error(error?.detail || 'Registration failed');
				setToken(data.token);
				set({ user: data.user, isLoading: false, isAuthenticated: true });
				return { success: true };
			} catch (e) {
				update((s) => ({ ...s, isLoading: false }));
				return { success: false, error: (e as Error).message };
			}
		},
		logout() {
			setToken(null);
			localStorage.removeItem('token');
			set({ user: null, isLoading: false, isAuthenticated: false });
		}
	};
}

export const auth = createAuthStore();
