import { browser } from '$app/environment';
import createClient from 'openapi-fetch';
import type { paths, components } from './types';

const API_BASE = '/api/v1';

// Re-export schema types for convenience
export type User = components['schemas']['UserProfile'];
export type Workspace = components['schemas']['Item'];
export type Post = components['schemas']['PostResponse'];
export type SocialAccount = components['schemas']['AccountResponse'];
export type ScheduleOverview = components['schemas']['ScheduleOverviewOutputBody'];
export type AuthResponse = components['schemas']['AuthOutputBody'];

let token: string | null = null;

if (browser) {
	token = localStorage.getItem('token');
}

export function setToken(newToken: string | null) {
	token = newToken;
	if (browser) {
		if (newToken) {
			localStorage.setItem('token', newToken);
		} else {
			localStorage.removeItem('token');
		}
	}
}

export function getToken(): string | null {
	return token;
}

// Create the typed fetch client
const rawClient = createClient<paths>({ baseUrl: API_BASE });

// Auth middleware that adds Bearer token to every request
rawClient.use({
	async onRequest({ request }) {
		if (token) {
			request.headers.set('Authorization', `Bearer ${token}`);
		}
		return request;
	}
});

export const client = rawClient;
