import { browser } from '$app/environment';
import { writable } from 'svelte/store';
import { client, setToken, recreateClient, type User } from '$lib/api/client';
import { getPasskeyAssertion } from '$lib/auth/webauthn';

interface AuthState {
	user: User | null;
	isLoading: boolean;
	isAuthenticated: boolean;
}

interface AuthActionResult {
	success: boolean;
	error?: string;
	requiresMfa?: boolean;
	mfaToken?: string;
	mfaMethods?: string[];
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

			// Recreate client in case instance URL was just set
			recreateClient();

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
		async login(email: string, password: string): Promise<AuthActionResult> {
			try {
				const { data, error } = await (client as any).POST('/auth/login', {
					body: { email, password }
				});
				if (error || !data) throw new Error(error?.detail || 'Login failed');
				if (data.requires_mfa) {
					set({ user: null, isLoading: false, isAuthenticated: false });
					return {
						success: false,
						requiresMfa: true,
						mfaToken: data.mfa_token,
						mfaMethods: data.mfa_methods ?? []
					};
				}
				setToken(data.token);
				set({ user: data.user, isLoading: false, isAuthenticated: true });
				return { success: true };
			} catch (e) {
				return { success: false, error: (e as Error).message };
			}
		},
		async register(email: string, password: string) {
			try {
				const { data, error } = await client.POST('/auth/register', {
					body: { email, password }
				});
				if (error || !data) throw new Error(error?.detail || 'Registration failed');
				setToken(data.token);
				set({ user: data.user, isLoading: false, isAuthenticated: true });
				return { success: true };
			} catch (e) {
				return { success: false, error: (e as Error).message };
			}
		},
		async verifyTOTP(mfaToken: string, code: string): Promise<AuthActionResult> {
			try {
				const { data, error } = await (client as any).POST('/auth/login/totp', {
					body: { mfa_token: mfaToken, code }
				});
				if (error || !data) throw new Error(error?.detail || 'Authenticator verification failed');
				setToken(data.token);
				set({ user: data.user, isLoading: false, isAuthenticated: true });
				return { success: true };
			} catch (e) {
				return { success: false, error: (e as Error).message };
			}
		},
		async verifyPasskey(mfaToken: string): Promise<AuthActionResult> {
			try {
				const { data: beginData, error: beginError } = await (client as any).POST(
					'/auth/login/passkey/options',
					{
						body: { mfa_token: mfaToken }
					}
				);
				if (beginError || !beginData) {
					throw new Error(beginError?.detail || 'Unable to start passkey verification');
				}

				const credential = await getPasskeyAssertion(beginData.options);
				const { data, error } = await (client as any).POST('/auth/login/passkey/verify', {
					body: {
						challenge_id: beginData.challenge_id,
						credential
					}
				});
				if (error || !data) throw new Error(error?.detail || 'Passkey verification failed');

				setToken(data.token);
				set({ user: data.user, isLoading: false, isAuthenticated: true });
				return { success: true };
			} catch (e) {
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
