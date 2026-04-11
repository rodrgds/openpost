import { browser } from '$app/environment';
import createClient from 'openapi-fetch';
import type { paths, components } from './types';
import { getApiBase } from '$lib/stores/instance.svelte';

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

function createApiClient() {
	const c = createClient<paths>({ baseUrl: getApiBase() });
	c.use({
		async onRequest({ request }) {
			if (token) {
				request.headers.set('Authorization', `Bearer ${token}`);
			}
			return request;
		}
	});
	return c;
}

let rawClient = createApiClient();

export function recreateClient() {
	rawClient = createApiClient();
}

export const client = new Proxy(rawClient, {
	get(_target, prop) {
		const val = Reflect.get(rawClient, prop, rawClient);
		if (typeof val === 'function') {
			return val.bind(rawClient);
		}
		return val;
	}
});
