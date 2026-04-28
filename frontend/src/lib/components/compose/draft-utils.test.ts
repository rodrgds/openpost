import { describe, it, expect } from 'vitest';
import {
	makeEmptyPost,
	encodeThreadDraft,
	decodeThreadDraft,
	isThreadDraft,
	hasAnyContent,
	getDraftSnapshot,
	THREAD_DRAFT_PREFIX
} from './draft-utils';

describe('draft-utils', () => {
	describe('makeEmptyPost', () => {
		it('creates a post with empty content and no media', () => {
			const post = makeEmptyPost();
			expect(post.content).toBe('');
			expect(post.mediaIds).toEqual([]);
			expect(post.key).toBeTruthy();
		});
	});

	describe('encodeThreadDraft', () => {
		it('encodes posts to thread draft format', () => {
			const posts = [
				{ key: 'a', content: 'Hello', mediaIds: ['m1'] },
				{ key: 'b', content: 'World', mediaIds: [] }
			];
			const encoded = encodeThreadDraft(posts);
			expect(encoded.startsWith(THREAD_DRAFT_PREFIX)).toBe(true);
			const decoded = decodeThreadDraft(encoded);
			expect(decoded).toEqual([
				{ content: 'Hello', mediaIds: ['m1'] },
				{ content: 'World', mediaIds: [] }
			]);
		});
	});

	describe('isThreadDraft', () => {
		it('returns true for thread draft content', () => {
			expect(isThreadDraft(THREAD_DRAFT_PREFIX + '[]')).toBe(true);
		});
		it('returns false for regular content', () => {
			expect(isThreadDraft('Hello world')).toBe(false);
		});
	});

	describe('decodeThreadDraft', () => {
		it('returns null for invalid content', () => {
			expect(decodeThreadDraft('not a thread')).toBeNull();
		});
		it('returns null for invalid JSON', () => {
			expect(decodeThreadDraft(THREAD_DRAFT_PREFIX + 'invalid')).toBeNull();
		});
	});

	describe('hasAnyContent', () => {
		it('returns true if any post has text', () => {
			expect(hasAnyContent([{ key: 'a', content: 'Hi', mediaIds: [] }])).toBe(true);
		});
		it('returns true if any post has media', () => {
			expect(hasAnyContent([{ key: 'a', content: '', mediaIds: ['m1'] }])).toBe(true);
		});
		it('returns false for empty posts', () => {
			expect(hasAnyContent([{ key: 'a', content: '', mediaIds: [] }])).toBe(false);
		});
	});

	describe('getDraftSnapshot', () => {
		it('returns consistent snapshot for same posts', () => {
			const posts = [{ key: 'a', content: 'Hello', mediaIds: ['m1'] }];
			expect(getDraftSnapshot(posts)).toBe(getDraftSnapshot(posts));
		});
	});
});
