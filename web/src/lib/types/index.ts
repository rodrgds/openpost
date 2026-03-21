export interface User {
	id: string;
	email: string;
	created_at: string;
}

export interface Workspace {
	id: string;
	name: string;
	created_at: string;
}

export interface SocialAccount {
	id: string;
	workspace_id: string;
	platform: string;
	account_id: string;
	account_username: string;
	account_avatar_url: string;
	instance_url: string;
	is_active: boolean;
	error_message: string;
}

export interface Post {
	id: string;
	workspace_id: string;
	created_by: string;
	content: string;
	status: 'draft' | 'scheduled' | 'publishing' | 'published' | 'failed';
	scheduled_at: string | null;
	published_at: string | null;
	created_at: string;
}

export interface PostDestination {
	id: string;
	post_id: string;
	social_account_id: string;
	external_id: string;
	status: 'pending' | 'success' | 'failed';
	error_message: string;
}

export interface ScheduleDay {
	date: string;
	count: number;
	platforms: {
		platform: string;
		count: number;
	}[];
}

export interface ScheduleOverview {
	year: number;
	month: number;
	selected_workspace_id: string;
	selected_platform: string;
	workspaces: Workspace[];
	platforms: string[];
	days: ScheduleDay[];
}

export interface AuthResponse {
	token: string;
	user: User;
}
