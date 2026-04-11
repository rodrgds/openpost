import { browser } from '$app/environment';
import { IS_CAPACITOR } from '$lib/env';

const STORAGE_KEY = 'openpost_instance_url';

let instanceUrl = $state<string | null>(null);
let isLoading = $state(true);

function normalizeUrl(raw: string): string {
	let url = raw.trim();
	if (!url) return '';
	if (!url.startsWith('http://') && !url.startsWith('https://')) {
		url = 'https://' + url;
	}
	return url.replace(/\/+$/, '');
}

export function getInstanceUrl(): string | null {
	return instanceUrl;
}

export function getApiBase(): string {
	if (IS_CAPACITOR && instanceUrl) {
		return `${instanceUrl}/api/v1`;
	}
	return '/api/v1';
}

export function getMediaBase(): string {
	if (IS_CAPACITOR && instanceUrl) {
		return instanceUrl;
	}
	return '';
}

export function isInstanceConfigured(): boolean {
	if (!IS_CAPACITOR) return true;
	return instanceUrl !== null && instanceUrl.length > 0;
}

export function instanceStore() {
	return {
		get instanceUrl() {
			return instanceUrl;
		},
		get isLoading() {
			return isLoading;
		},

		initialize() {
			if (!browser) {
				isLoading = false;
				return;
			}

			const stored = localStorage.getItem(STORAGE_KEY);
			if (stored) {
				instanceUrl = stored;
			} else if (!IS_CAPACITOR) {
				// Web mode: no instance URL needed
			}
			isLoading = false;
		},

		async setInstanceUrl(raw: string): Promise<{ success: boolean; error?: string }> {
			const url = normalizeUrl(raw);
			if (!url) return { success: false, error: 'Please enter a server URL' };

			const result = await testConnection(url);
			if (!result.ok) {
				return { success: false, error: result.error };
			}

			instanceUrl = url;
			if (browser) {
				localStorage.setItem(STORAGE_KEY, url);
			}
			return { success: true };
		},

		clearInstanceUrl() {
			instanceUrl = null;
			if (browser) {
				localStorage.removeItem(STORAGE_KEY);
			}
		}
	};
}

async function testConnection(url: string): Promise<{ ok: boolean; error?: string }> {
	try {
		const controller = new AbortController();
		const timeout = setTimeout(() => controller.abort(), 10000);

		const resp = await fetch(`${url}/api/v1/health`, {
			signal: controller.signal
		});
		clearTimeout(timeout);

		if (!resp.ok) {
			return { ok: false, error: `Server responded with ${resp.status}` };
		}

		const data = await resp.json();
		if (data?.status !== 'ok') {
			return { ok: false, error: 'Server is not a valid OpenPost instance' };
		}

		return { ok: true };
	} catch (e) {
		const err = e as Error;
		if (err.name === 'AbortError') {
			return { ok: false, error: 'Connection timed out. Check the URL and try again.' };
		}
		return { ok: false, error: `Could not connect: ${err.message}` };
	}
}
