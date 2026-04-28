import { browser } from '$app/environment';
import { client, type Workspace } from '$lib/api/client';

interface WorkspaceSettings {
	timezone: string;
	week_start: number;
	media_cleanup_days: number;
	random_delay_minutes: number;
	slot_start_hour: number;
	slot_end_hour: number;
	slot_interval_minutes: number;
}

const STORAGE_KEY = 'openpost_current_workspace';

class WorkspaceContext {
	currentWorkspace = $state<Workspace | null>(null);
	workspaces = $state<Workspace[]>([]);
	settings = $state<WorkspaceSettings>({
		timezone: 'UTC',
		week_start: 1,
		media_cleanup_days: 0,
		random_delay_minutes: 0,
		slot_start_hour: 5,
		slot_end_hour: 23,
		slot_interval_minutes: 15
	});
	loading = $state(false);

	async initialize() {
		if (!browser) return;

		const stored = localStorage.getItem(STORAGE_KEY);
		if (stored) {
			try {
				this.currentWorkspace = JSON.parse(stored);
			} catch {
				// ignore
			}
		}

		await this.loadWorkspaces();
	}

	async loadWorkspaces() {
		try {
			const { data } = await client.GET('/workspaces', {});
			this.workspaces = data ?? [];

			if (this.workspaces.length > 0 && !this.currentWorkspace) {
				await this.setWorkspace(this.workspaces[0]);
			} else if (this.currentWorkspace) {
				const exists = this.workspaces.find((w) => w.id === this.currentWorkspace?.id);
				if (!exists && this.workspaces.length > 0) {
					await this.setWorkspace(this.workspaces[0]);
				} else if (exists) {
					await this.loadSettings();
				}
			}
		} catch (e) {
			console.error('Failed to load workspaces:', e);
		}
	}

	async setWorkspace(workspace: Workspace) {
		this.currentWorkspace = workspace;
		if (browser) {
			localStorage.setItem(STORAGE_KEY, JSON.stringify(workspace));
		}
		await this.loadSettings();
	}

	async loadSettings() {
		if (!this.currentWorkspace) return;

		try {
			const { data, error } = await (client as any).GET('/workspaces/{id}/settings', {
				params: { path: { id: this.currentWorkspace.id } }
			});
			if (!error && data) {
				this.settings = {
					timezone: data.timezone || 'UTC',
					week_start: data.week_start ?? 1,
					media_cleanup_days: data.media_cleanup_days ?? 0,
					random_delay_minutes: data.random_delay_minutes ?? 0,
					slot_start_hour: data.slot_start_hour ?? 5,
					slot_end_hour: data.slot_end_hour ?? 23,
					slot_interval_minutes: data.slot_interval_minutes ?? 15
				};
			}
		} catch (e) {
			console.error('Failed to load workspace settings:', e);
		}
	}

	async saveSettings(updates: Partial<WorkspaceSettings>) {
		if (!this.currentWorkspace) return;

		try {
			const { error } = await (client as any).PATCH('/workspaces/{id}/settings', {
				params: { path: { id: this.currentWorkspace.id } },
				body: updates
			});
			if (error) throw new Error(error.detail || 'Failed to save settings');

			if (updates.timezone !== undefined) this.settings.timezone = updates.timezone;
			if (updates.week_start !== undefined) this.settings.week_start = updates.week_start;
			if (updates.media_cleanup_days !== undefined)
				this.settings.media_cleanup_days = updates.media_cleanup_days;
			if (updates.random_delay_minutes !== undefined)
				this.settings.random_delay_minutes = updates.random_delay_minutes;
			if (updates.slot_start_hour !== undefined)
				this.settings.slot_start_hour = updates.slot_start_hour;
			if (updates.slot_end_hour !== undefined) this.settings.slot_end_hour = updates.slot_end_hour;
			if (updates.slot_interval_minutes !== undefined)
				this.settings.slot_interval_minutes = updates.slot_interval_minutes;
		} catch (e) {
			console.error('Failed to save workspace settings:', e);
			throw e;
		}
	}

	get weekStartsOn(): 0 | 1 | 2 | 3 | 4 | 5 | 6 {
		return this.settings.week_start as 0 | 1 | 2 | 3 | 4 | 5 | 6;
	}
}

export const workspaceCtx = new WorkspaceContext();
