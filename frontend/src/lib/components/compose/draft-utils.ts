export interface PostItem {
	id?: string;
	key: string;
	content: string;
	mediaIds: string[];
}

export const THREAD_DRAFT_PREFIX = '__openpost_thread__:';

export function generatePostKey(): string {
	return Math.random().toString(36).substring(2, 10);
}

export function makeEmptyPost(): PostItem {
	return { key: generatePostKey(), content: '', mediaIds: [] };
}

export function encodeThreadDraft(posts: PostItem[]): string {
	const data = posts.map((p) => ({ c: p.content, m: p.mediaIds }));
	return THREAD_DRAFT_PREFIX + JSON.stringify(data);
}

export function isThreadDraft(content: string): boolean {
	return content.startsWith(THREAD_DRAFT_PREFIX);
}

export function decodeThreadDraft(
	content: string
): { content: string; mediaIds: string[] }[] | null {
	try {
		const data = JSON.parse(content.slice(THREAD_DRAFT_PREFIX.length));
		if (!Array.isArray(data)) return null;
		return data.map((item: any) => ({
			content: item.c ?? '',
			mediaIds: item.m ?? []
		}));
	} catch {
		return null;
	}
}

export function getDraftSnapshot(posts: PostItem[]): string {
	return JSON.stringify(posts.map((p) => ({ content: p.content, mediaIds: p.mediaIds })));
}

export function hasAnyContent(posts: PostItem[]): boolean {
	return posts.some((p) => p.content.trim().length > 0 || p.mediaIds.length > 0);
}

export function countTotalChars(posts: PostItem[]): number {
	return posts.reduce((sum, p) => sum + p.content.length, 0);
}

export function getPostMediaIdsForSave(posts: PostItem[], isThread: boolean): string[] {
	if (isThread) {
		return posts.flatMap((p) => p.mediaIds);
	}
	return posts[0]?.mediaIds ?? [];
}
