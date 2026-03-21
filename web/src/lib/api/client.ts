import { browser } from '$app/environment';
import type { AuthResponse, User, Workspace, Post, SocialAccount, ScheduleOverview } from '$lib/types';

const API_BASE = '/api/v1';

class ApiClient {
	private token: string | null = null;

	constructor() {
		if (browser) {
			this.token = localStorage.getItem('token');
		}
	}

	setToken(token: string | null) {
		this.token = token;
		if (browser) {
			if (token) {
				localStorage.setItem('token', token);
			} else {
				localStorage.removeItem('token');
			}
		}
	}

	getToken(): string | null {
		return this.token;
	}

	private async request<T>(endpoint: string, options: RequestInit = {}): Promise<T> {
		const headers: Record<string, string> = {
			'Content-Type': 'application/json',
			...((options.headers as Record<string, string>) || {}),
		};

		if (this.token) {
			headers['Authorization'] = `Bearer ${this.token}`;
		}

		const response = await fetch(`${API_BASE}${endpoint}`, {
			...options,
			headers,
		});

		if (!response.ok) {
			const error = await response.json().catch(() => ({ error: 'Unknown error' }));
			throw new Error(error.error || `HTTP ${response.status}`);
		}

		return response.json();
	}

	async register(email: string, password: string): Promise<AuthResponse> {
		return this.request('/auth/register', {
			method: 'POST',
			body: JSON.stringify({ email, password }),
		});
	}

	async login(email: string, password: string): Promise<AuthResponse> {
		return this.request('/auth/login', {
			method: 'POST',
			body: JSON.stringify({ email, password }),
		});
	}

	async getMe(): Promise<User> {
		return this.request('/auth/me');
	}

	async createWorkspace(name: string): Promise<Workspace> {
		return this.request('/workspaces', {
			method: 'POST',
			body: JSON.stringify({ name }),
		});
	}

	async listWorkspaces(): Promise<Workspace[]> {
		return this.request('/workspaces');
	}

	async createPost(workspaceId: string, content: string, socialAccountIds: string[], scheduledAt?: string): Promise<Post> {
		return this.request('/posts', {
			method: 'POST',
			body: JSON.stringify({
				workspace_id: workspaceId,
				content,
				social_account_ids: socialAccountIds,
				scheduled_at: scheduledAt,
			}),
		});
	}

	async listPosts(workspaceId: string): Promise<Post[]> {
		return this.request(`/posts?workspace_id=${workspaceId}`);
	}

	async getScheduleOverview(params?: { workspaceId?: string; platform?: string; month?: string }): Promise<ScheduleOverview> {
		const searchParams = new URLSearchParams();
		if (params?.workspaceId) {
			searchParams.set('workspace_id', params.workspaceId);
		}
		if (params?.platform) {
			searchParams.set('platform', params.platform);
		}
		if (params?.month) {
			searchParams.set('month', params.month);
		}
		const query = searchParams.toString();
		return this.request(`/posts/schedule-overview${query ? `?${query}` : ''}`);
	}

	async getTwitterAuthUrl(workspaceId: string): Promise<{ url: string }> {
		return this.request(`/accounts/x/auth-url?workspace_id=${workspaceId}`);
	}

	async getMastodonAuthUrl(workspaceId: string, instance: string): Promise<{ url: string }> {
		return this.request(`/accounts/mastodon/auth-url?workspace_id=${workspaceId}&instance=${encodeURIComponent(instance)}`);
	}

	async exchangeMastodonCode(workspaceId: string, instance: string, code: string): Promise<void> {
		return this.request('/accounts/mastodon/exchange', {
			method: 'POST',
			body: JSON.stringify({ workspace_id: workspaceId, instance, code }),
		});
	}

	async listAccounts(workspaceId: string): Promise<SocialAccount[]> {
		return this.request(`/accounts?workspace_id=${workspaceId}`);
	}
}

export const api = new ApiClient();
