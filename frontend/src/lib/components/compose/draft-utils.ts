export interface PostItem {
	id?: string;
	key: string;
	content: string;
	mediaIds: string[];
}

export type ThreadVariantMap = Record<string, Record<string, string>>;

export interface DecodedThreadDraft {
	posts: { key: string; content: string; mediaIds: string[] }[];
	variants: ThreadVariantMap;
}

export const THREAD_DRAFT_PREFIX = '__openpost_thread__:';

export function generatePostKey(): string {
	return Math.random().toString(36).substring(2, 10);
}

export function makeEmptyPost(): PostItem {
	return { key: generatePostKey(), content: '', mediaIds: [] };
}

export function encodeThreadDraft(posts: PostItem[], variants: ThreadVariantMap = {}): string {
	const data = {
		p: posts.map((p) => ({ k: p.key, c: p.content, m: p.mediaIds })),
		v: variants
	};
	return THREAD_DRAFT_PREFIX + JSON.stringify(data);
}

export function isThreadDraft(content: string): boolean {
	return content.startsWith(THREAD_DRAFT_PREFIX);
}

export function decodeThreadDraft(content: string): DecodedThreadDraft | null {
	try {
		const data = JSON.parse(content.slice(THREAD_DRAFT_PREFIX.length));
		if (Array.isArray(data)) {
			return {
				posts: data.map((item: any) => ({
					key: item.k ?? generatePostKey(),
					content: item.c ?? '',
					mediaIds: item.m ?? []
				})),
				variants: {}
			};
		}
		if (!data || !Array.isArray(data.p)) return null;
		return {
			posts: data.p.map((item: any) => ({
				key: item.k ?? generatePostKey(),
				content: item.c ?? '',
				mediaIds: item.m ?? []
			})),
			variants:
				data.v && typeof data.v === 'object'
					? Object.fromEntries(
							Object.entries(data.v).map(([accountId, value]) => [
								accountId,
								normalizeVariantValue(value)
							])
						)
					: {}
		};
	} catch {
		return null;
	}
}

function normalizeVariantValue(value: unknown): Record<string, string> {
	if (Array.isArray(value)) {
		return Object.fromEntries(value.map((item, index) => [String(index), String(item ?? '')]));
	}
	if (!value || typeof value !== 'object') {
		return {};
	}
	return Object.fromEntries(
		Object.entries(value as Record<string, unknown>).map(([postKey, content]) => [
			postKey,
			String(content ?? '')
		])
	);
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
