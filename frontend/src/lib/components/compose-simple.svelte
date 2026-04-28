<script lang="ts">
	import { onMount, tick, type Snippet } from 'svelte';
	import { client, type SocialAccount, type Workspace, getToken } from '$lib/api/client';
	import { getApiBase, getMediaBase } from '$lib/stores/instance.svelte';
	import { workspaceCtx } from '$lib/stores/workspace.svelte';
	import { Button } from '$lib/components/ui/button';
	import { Calendar } from '$lib/components/ui/calendar';
	import * as Popover from '$lib/components/ui/popover';
	import * as Dialog from '$lib/components/ui/dialog';
	import * as DropdownMenu from '$lib/components/ui/dropdown-menu';
	import * as Tooltip from '$lib/components/ui/tooltip';
	import PlatformPreview from './platform-preview.svelte';
	import PlatformIcon from './platform-icon.svelte';
	import { getPlatformKey, getPlatformName } from '$lib/utils';
	import { CalendarDate, getLocalTimeZone, today, isEqualDay } from '@internationalized/date';
	import LoaderIcon from 'lucide-svelte/icons/loader-2';
	import PlusIcon from 'lucide-svelte/icons/plus';
	import XIcon from 'lucide-svelte/icons/x';
	import ClockIcon from 'lucide-svelte/icons/clock';
	import LightbulbIcon from 'lucide-svelte/icons/lightbulb';
	import ShuffleIcon from 'lucide-svelte/icons/shuffle';
	import ImageIcon from 'lucide-svelte/icons/image';
	import SendIcon from 'lucide-svelte/icons/send';
	import ChevronDownIcon from 'lucide-svelte/icons/chevron-down';
	import UnlinkIcon from 'lucide-svelte/icons/unlink';
	import GripVerticalIcon from 'lucide-svelte/icons/grip-vertical';
	import Trash2Icon from 'lucide-svelte/icons/trash-2';
	import TypeIcon from 'lucide-svelte/icons/type';
	import EyeIcon from 'lucide-svelte/icons/eye';
	import { ui } from '$lib/stores/ui.svelte';
	import { Skeleton } from '$lib/components/ui/skeleton/index.js';
	import { ReorderableList } from 'svelte-reorderable-list';
	import * as Sheet from '$lib/components/ui/sheet';
	import {
		type PostItem,
		makeEmptyPost,
		encodeThreadDraft,
		isThreadDraft,
		decodeThreadDraft,
		getDraftSnapshot,
		hasAnyContent
	} from './compose/draft-utils';

	// --------------------------------------------------------------------------
	// Types
	// --------------------------------------------------------------------------
	interface InitialPost {
		id: string;
		workspace_id: string;
		content: string;
		status: string;
		scheduled_at: string;
		media: Array<{ media_id: string; alt_text?: string }>;
		destinations: Array<{ social_account_id: string; platform: string }>;
	}

	interface SocialMediaSet {
		id: string;
		workspace_id: string;
		name: string;
		is_default: boolean;
		accounts: Array<{
			social_account_id: string;
			platform: string;
			account_username: string;
			is_main: boolean;
		}>;
	}

	interface Props {
		initialPost?: InitialPost;
		onSuccess?: () => void;
		onCancel?: () => void;
	}

	// --------------------------------------------------------------------------
	// Props & core state
	// --------------------------------------------------------------------------
	let { initialPost, onSuccess, onCancel }: Props = $props();
	let isEditMode = $derived(!!initialPost);

	let posts = $state<PostItem[]>([makeEmptyPost()]);
	let activePostIndex = $state(0);
	let draftId = $state<string | null>(null);
	let lastInitializedPostId = $state<string | null>(null);
	let isSaving = $state(false);
	let isSubmitting = $state(false);
	let error = $state('');
	let success = $state('');

	let workspaces = $state<Workspace[]>([]);
	let selectedWorkspaceId = $state<string>('');
	let accounts = $state<SocialAccount[]>([]);
	let selectedAccountIds = $state<string[]>([]);
	let loadingWorkspaces = $state(true);
	let loadingAccounts = $state(false);

	let sets = $state<SocialMediaSet[]>([]);
	let selectedSetId = $state<string | null>(null);
	let loadingSets = $state(false);

	let showPreview = $state(true);
	let showMobilePreview = $state(false);

	let selectedDate = $state<CalendarDate | undefined>(undefined);
	let selectedTime = $state<string | null>(null);
	let suggestingSlot = $state(false);
	let showSchedulePopover = $state(false);

	let showPromptCard = $state(false);
	let currentPrompt = $state<{ text: string; category: string } | null>(null);
	let loadingPrompt = $state(false);

	let variants = $state<Map<string, string>>(new Map());
	let showVariantsDialog = $state(false);

	let isDraggingFile = $state(false);
	let isUploading = $state(false);

	let mediaAltTexts = $state<Map<string, string>>(new Map());
	let editingAltMediaId = $state<string | null>(null);

	let autoSaveTimer: ReturnType<typeof setTimeout> | null = null;
	let lastSavedSnapshot = $state('');
	let textareaRefs = $state<Map<number, HTMLTextAreaElement>>(new Map());

	// --------------------------------------------------------------------------
	// Constants & derived values
	// --------------------------------------------------------------------------
	const PLATFORM_CHAR_LIMITS: Record<string, number> = {
		x: 280,
		mastodon: 500,
		bluesky: 300,
		linkedin: 3000,
		threads: 500
	};

	// Generate time slots dynamically from workspace settings
	const allTimeSlots = $derived.by(() => {
		const start = workspaceCtx.settings.slot_start_hour;
		const end = workspaceCtx.settings.slot_end_hour;
		const interval = workspaceCtx.settings.slot_interval_minutes;
		const slots: string[] = [];
		for (let hour = start; hour <= end; hour++) {
			for (let min = 0; min < 60; min += interval) {
				if (hour === end && min > 0) break;
				slots.push(`${hour.toString().padStart(2, '0')}:${min.toString().padStart(2, '0')}`);
			}
		}
		return slots;
	});

	const isToday = $derived(
		selectedDate ? isEqualDay(selectedDate, today(getLocalTimeZone())) : false
	);

	const timeSlots = $derived.by(() => {
		if (!isToday) return allTimeSlots;
		const now = new Date();
		const currentMinutes = now.getHours() * 60 + now.getMinutes();
		return allTimeSlots.filter((slot) => {
			const [h, m] = slot.split(':').map(Number);
			return h * 60 + m > currentMinutes;
		});
	});

	const activePost = $derived(posts[activePostIndex] ?? posts[0]);
	const hasContent = $derived(hasAnyContent(posts));
	const totalChars = $derived(posts.reduce((sum, p) => sum + p.content.length, 0));
	const isThread = $derived(posts.length > 1);

	const selectedAccounts = $derived(accounts.filter((a) => selectedAccountIds.includes(a.id)));

	const selectedPlatformLimits = $derived.by(() => {
		const seen = new Set<string>();
		return accounts
			.filter((a) => selectedAccountIds.includes(a.id))
			.map((a) => {
				const key = getPlatformKey(a.platform);
				return {
					platform: getPlatformName(a.platform),
					key,
					limit: PLATFORM_CHAR_LIMITS[key] ?? 280
				};
			})
			.filter((item) => {
				if (seen.has(item.key)) return false;
				seen.add(item.key);
				return true;
			});
	});

	const maxChars = $derived.by(() => {
		const selected = accounts.filter((a) => selectedAccountIds.includes(a.id));
		if (selected.length === 0) return 280;
		const limits = selected.map((a) => PLATFORM_CHAR_LIMITS[getPlatformKey(a.platform)] ?? 280);
		return Math.min(...limits);
	});

	// --------------------------------------------------------------------------
	// Helpers
	// --------------------------------------------------------------------------
	function getCharCounterColor(count: number, max: number): string {
		const pct = count / max;
		if (pct >= 1) return 'text-red-500';
		if (pct >= 0.8) return 'text-amber-500';
		return 'text-muted-foreground';
	}

	function getCharCounterStrokeColor(count: number, max: number): string {
		const pct = count / max;
		if (pct >= 1) return '#ef4444';
		if (pct >= 0.8) return '#f59e0b';
		return 'currentColor';
	}

	function autoResize(el: HTMLTextAreaElement) {
		el.style.height = 'auto';
		el.style.height = el.scrollHeight + 'px';
	}

	function textareaAction(el: HTMLTextAreaElement, index: number) {
		textareaRefs.set(index, el);
		autoResize(el);
		return {
			update() {
				textareaRefs.set(index, el);
			},
			destroy() {
				textareaRefs.delete(index);
			}
		};
	}

	function getScheduledAt(): string | undefined {
		if (!selectedDate || !selectedTime) return undefined;
		const [hours, minutes] = selectedTime.split(':').map(Number);
		const date = selectedDate.toDate(getLocalTimeZone());
		date.setHours(hours, minutes, 0, 0);
		return date.toISOString();
	}

	// --------------------------------------------------------------------------
	// Initialization
	// --------------------------------------------------------------------------
	async function initializeFromPost(post: InitialPost | undefined) {
		if (!post) {
			draftId = null;
			lastInitializedPostId = null;
			posts = [makeEmptyPost()];
			activePostIndex = 0;
			lastSavedSnapshot = '';
			variants = new Map();
			selectedAccountIds = [];
			selectedSetId = null;
			const tomorrow = today(getLocalTimeZone()).add({ days: 1 });
			selectedDate = new CalendarDate(tomorrow.year, tomorrow.month, tomorrow.day);
			selectedTime = '10:00';
			if (workspaces.length > 0) {
				selectedWorkspaceId = workspaceCtx.currentWorkspace?.id ?? workspaces[0].id;
				await loadAccounts(selectedWorkspaceId);
				await loadSets(selectedWorkspaceId);
			}
			return;
		}

		draftId = post.id;
		lastInitializedPostId = post.id;
		selectedWorkspaceId = post.workspace_id;
		selectedAccountIds = post.destinations?.map((d) => d.social_account_id) ?? [];

		// Load alt texts from media
		const newAlts = new Map<string, string>();
		post.media?.forEach((m) => {
			if (m.alt_text) newAlts.set(m.media_id, m.alt_text);
		});
		mediaAltTexts = newAlts;

		if (isThreadDraft(post.content)) {
			const threadData = decodeThreadDraft(post.content);
			if (threadData && threadData.length > 0) {
				posts = threadData.map((item) => ({
					key: makeEmptyPost().key,
					content: item.content,
					mediaIds: item.mediaIds
				}));
			} else {
				posts = [makeEmptyPost()];
			}
		} else {
			posts = [
				{
					key: makeEmptyPost().key,
					content: post.content,
					mediaIds: post.media?.map((m) => m.media_id) ?? []
				}
			];
		}
		activePostIndex = 0;
		lastSavedSnapshot = getDraftSnapshot(posts);
		variants = new Map();
		selectedSetId = null;

		if (post.scheduled_at && post.scheduled_at !== '0001-01-01T00:00:00Z') {
			const date = new Date(post.scheduled_at);
			selectedDate = new CalendarDate(date.getFullYear(), date.getMonth() + 1, date.getDate());
			selectedTime = `${date.getHours().toString().padStart(2, '0')}:${date.getMinutes().toString().padStart(2, '0')}`;
		} else {
			const tomorrow = today(getLocalTimeZone()).add({ days: 1 });
			selectedDate = new CalendarDate(tomorrow.year, tomorrow.month, tomorrow.day);
			selectedTime = '10:00';
		}

		await loadAccounts(selectedWorkspaceId, selectedAccountIds);
		await loadSets(selectedWorkspaceId, false);
	}

	onMount(async () => {
		try {
			const { data, error: err } = await client.GET('/workspaces');
			if (err || !data) throw new Error('Failed to load workspaces');
			workspaces = data;
			await initializeFromPost(initialPost);
		} catch (e) {
			console.error('Failed to load workspaces:', e);
		} finally {
			loadingWorkspaces = false;
		}
	});

	$effect(() => {
		const post = initialPost;
		if (!loadingWorkspaces && post && lastInitializedPostId !== post.id) {
			initializeFromPost(post);
		}
	});

	$effect(() => {
		tick().then(() => {
			textareaRefs.forEach((el) => {
				if (el) autoResize(el);
			});
		});
	});

	$effect(() => {
		const text = ui.promptText;
		if (text && !initialPost && !loadingWorkspaces) {
			posts = [{ ...makeEmptyPost(), content: text }];
			activePostIndex = 0;
			ui.clearPrompt();
		}
	});

	// --------------------------------------------------------------------------
	// Data loading
	// --------------------------------------------------------------------------
	async function loadAccounts(workspaceId: string, preferredAccountIds?: string[]) {
		if (!workspaceId) return;
		try {
			const { data, error: err } = await client.GET('/accounts', {
				params: { query: { workspace_id: workspaceId } }
			});
			accounts = data ?? [];
			if (preferredAccountIds && preferredAccountIds.length > 0) {
				const validIds = accounts.map((a) => a.id);
				selectedAccountIds = preferredAccountIds.filter((id) => validIds.includes(id));
				if (selectedAccountIds.length === 0) {
					selectedAccountIds = accounts.map((a) => a.id);
				}
			} else {
				selectedAccountIds = accounts.map((a) => a.id);
			}
		} catch (e) {
			console.error('Failed to load accounts:', e);
			accounts = [];
			selectedAccountIds = [];
		}
	}

	async function loadSets(workspaceId: string, autoApplyDefault = true) {
		if (!workspaceId) return;
		try {
			const { data, error: err } = await client.GET('/sets', {
				params: { query: { workspace_id: workspaceId } }
			});
			sets = (data ?? []) as unknown as SocialMediaSet[];
			if (autoApplyDefault && !selectedSetId) {
				const defaultSet = sets.find((s) => s.is_default);
				if (defaultSet) {
					selectedSetId = defaultSet.id;
					applySet(defaultSet);
				}
			}
		} catch (e) {
			console.error('Failed to load sets:', e);
			sets = [];
		}
	}

	function applySet(set: SocialMediaSet) {
		selectedAccountIds = set.accounts.map((a) => a.social_account_id);
	}

	function handleWorkspaceChange(value: string) {
		selectedWorkspaceId = value;
		selectedSetId = null;
		variants = new Map();
		loadAccounts(value);
		loadSets(value);
	}

	function handleSetChange(setId: string | null) {
		selectedSetId = setId;
		if (setId) {
			const set = sets.find((s) => s.id === setId);
			if (set) applySet(set);
		} else {
			selectedAccountIds = accounts.map((a) => a.id);
		}
	}

	function toggleAccount(id: string) {
		if (selectedAccountIds.includes(id)) {
			selectedAccountIds = selectedAccountIds.filter((a) => a !== id);
		} else {
			selectedAccountIds = [...selectedAccountIds, id];
		}
	}

	function selectAllAccounts() {
		selectedAccountIds = accounts.map((a) => a.id);
	}

	function clearAllAccounts() {
		selectedAccountIds = [];
	}

	// --------------------------------------------------------------------------
	// Draft saving
	// --------------------------------------------------------------------------
	function scheduleAutoSave() {
		if (autoSaveTimer) clearTimeout(autoSaveTimer);
		autoSaveTimer = setTimeout(() => {
			if (!hasContent) return;
			const snapshot = getDraftSnapshot(posts);
			if (snapshot !== lastSavedSnapshot) {
				saveDraft();
			}
		}, 2000);
	}

	async function saveDraft() {
		if (!selectedWorkspaceId || !hasContent) return;
		isSaving = true;
		error = '';

		try {
			const draftContent = isThread ? encodeThreadDraft(posts) : posts[0].content;
			const draftMediaIds = isThread ? posts.flatMap((p) => p.mediaIds) : posts[0].mediaIds;

			const defaultDelay = workspaceCtx.settings.random_delay_minutes;
			if (draftId) {
				const { error: patchErr } = await (client as any).PATCH('/posts/{id}', {
					params: { path: { id: draftId } },
					body: {
						content: draftContent,
						scheduled_at: '',
						social_account_ids: selectedAccountIds,
						media_ids: draftMediaIds,
						random_delay_minutes: defaultDelay
					}
				});
				if (patchErr) throw new Error((patchErr as any).detail || 'Failed to update draft');
			} else {
				const { data, error: postErr } = await client.POST('/posts', {
					body: {
						workspace_id: selectedWorkspaceId,
						content: draftContent,
						social_account_ids: selectedAccountIds,
						media_ids: draftMediaIds,
						random_delay_minutes: defaultDelay
					}
				});
				if (postErr) throw new Error((postErr as any).detail || 'Failed to save draft');
				if (data?.id) draftId = data.id;
			}

			lastSavedSnapshot = getDraftSnapshot(posts);
			ui.triggerRefresh();
		} catch (e) {
			console.error('Failed to auto-save draft:', e);
			error = (e as Error).message || 'Failed to save draft';
		} finally {
			isSaving = false;
		}
	}

	// --------------------------------------------------------------------------
	// Publishing
	// --------------------------------------------------------------------------
	async function publish(publishNow: boolean = false) {
		error = '';
		success = '';

		if (!selectedWorkspaceId) {
			error = 'Please select a workspace';
			return;
		}
		if (!hasContent) {
			error = 'Please enter some content';
			return;
		}
		if (selectedAccountIds.length === 0) {
			error = 'Please select at least one account';
			return;
		}

		let scheduledAt: string | undefined;
		if (publishNow) {
			scheduledAt = new Date().toISOString();
		} else {
			scheduledAt = getScheduledAt();
			if (!scheduledAt) {
				error = 'Please select a date and time';
				return;
			}
		}

		const randomDelay = workspaceCtx.settings.random_delay_minutes;
		isSubmitting = true;

		try {
			if (isThread) {
				const validPosts = posts.filter(
					(p) => p.content.trim().length > 0 || p.mediaIds.length > 0
				);
				if (validPosts.length < 2) {
					error = 'A thread must have at least 2 posts with content or media';
					isSubmitting = false;
					return;
				}

				const { data, error: err } = await client.POST('/posts/thread' as any, {
					body: {
						workspace_id: selectedWorkspaceId,
						social_account_ids: selectedAccountIds,
						scheduled_at: scheduledAt,
						random_delay_minutes: randomDelay,
						posts: validPosts.map((p) => ({
							content: p.content,
							media_ids: p.mediaIds
						}))
					}
				});
				if (err) throw new Error((err as any).detail || 'Failed to create thread');

				if (data?.post_ids && variants.size > 0) {
					const firstPostId = data.post_ids[0];
					const variantPayload = Array.from(variants.entries()).map(([accId, variantContent]) => ({
						social_account_id: accId,
						content: variantContent,
						is_unsynced: true
					}));
					await (client as any).PUT('/posts/{id}/variants', {
						params: { path: { id: firstPostId } },
						body: { variants: variantPayload }
					});
				}
			} else {
				const postId = draftId;
				if (postId) {
					const { error: patchErr } = await (client as any).PATCH('/posts/{id}', {
						params: { path: { id: postId } },
						body: {
							content: posts[0].content,
							scheduled_at: scheduledAt,
							social_account_ids: selectedAccountIds,
							media_ids: posts[0].mediaIds,
							random_delay_minutes: randomDelay
						}
					});
					if (patchErr) throw new Error((patchErr as any).detail || 'Failed to update post');
				} else {
					const { data, error: postErr } = await client.POST('/posts', {
						body: {
							workspace_id: selectedWorkspaceId,
							content: posts[0].content,
							social_account_ids: selectedAccountIds,
							scheduled_at: scheduledAt,
							media_ids: posts[0].mediaIds,
							random_delay_minutes: randomDelay
						}
					});
					if (postErr) throw new Error((postErr as any).detail || 'Failed to create post');
					if (data?.id) draftId = data.id;
				}

				if (draftId && variants.size > 0) {
					const variantPayload = Array.from(variants.entries()).map(([accId, variantContent]) => ({
						social_account_id: accId,
						content: variantContent,
						is_unsynced: true
					}));
					await (client as any).PUT('/posts/{id}/variants', {
						params: { path: { id: draftId } },
						body: { variants: variantPayload }
					});
				}
			}

			success = publishNow ? 'Published!' : 'Scheduled!';
			ui.triggerRefresh();

			if (isEditMode && onSuccess) {
				setTimeout(() => onSuccess(), 800);
			} else {
				posts = [makeEmptyPost()];
				activePostIndex = 0;
				draftId = null;
				lastSavedSnapshot = '';
				variants = new Map();
				setTimeout(() => (success = ''), 3000);
			}
		} catch (e) {
			error = (e as Error).message || 'Failed to publish';
		} finally {
			isSubmitting = false;
		}
	}

	// --------------------------------------------------------------------------
	// Thread management
	// --------------------------------------------------------------------------
	function addPost() {
		const newIndex = activePostIndex + 1;
		posts = [...posts.slice(0, newIndex), makeEmptyPost(), ...posts.slice(newIndex)];
		activePostIndex = newIndex;
		scheduleAutoSave();
		tick().then(() => {
			document.getElementById(`post-textarea-${newIndex}`)?.focus();
		});
	}

	function removePost(index: number) {
		if (posts.length <= 1) return;
		posts = posts.filter((_, i) => i !== index);
		if (activePostIndex >= posts.length) {
			activePostIndex = posts.length - 1;
		}
		scheduleAutoSave();
	}

	function handleReorder(newItems: PostItem[]) {
		posts = newItems;
		activePostIndex = Math.min(activePostIndex, newItems.length - 1);
		scheduleAutoSave();
	}

	// --------------------------------------------------------------------------
	// Media
	// --------------------------------------------------------------------------
	async function handleFileUpload(
		files: FileList | File[],
		targetPostIndex: number = activePostIndex
	) {
		if (!selectedWorkspaceId || isSubmitting) return;

		isUploading = true;
		try {
			for (const file of Array.from(files)) {
				if (!file.type.startsWith('image/') && !file.type.startsWith('video/')) continue;
				if (posts[targetPostIndex].mediaIds.length >= 4) break;

				const formData = new FormData();
				formData.append('file', file);
				formData.append('workspace_id', selectedWorkspaceId);

				const token = getToken();
				const resp = await fetch(`${getApiBase()}/media/upload`, {
					method: 'POST',
					headers: token ? { Authorization: `Bearer ${token}` } : {},
					body: formData
				});

				const data = await resp.json();
				if (resp.ok) {
					posts = posts.map((p, i) =>
						i === targetPostIndex ? { ...p, mediaIds: [...p.mediaIds, data.id] } : p
					);
					scheduleAutoSave();
				} else {
					error = data.error || 'Failed to upload media';
				}
			}
		} catch (e) {
			console.error('Failed to upload media:', e);
		} finally {
			isUploading = false;
		}
	}

	function handlePaste(e: ClipboardEvent, postIndex: number = activePostIndex) {
		const items = e.clipboardData?.items;
		if (!items) return;

		const files: File[] = [];
		for (const item of Array.from(items)) {
			if (item.kind === 'file') {
				const file = item.getAsFile();
				if (file) files.push(file);
			}
		}
		if (files.length > 0) {
			e.preventDefault();
			handleFileUpload(files, postIndex);
		}
	}

	function handleDragOver(e: DragEvent) {
		e.preventDefault();
		isDraggingFile = true;
	}

	function handleDragLeave(e: DragEvent) {
		e.preventDefault();
		isDraggingFile = false;
	}

	function handleDrop(e: DragEvent, postIndex: number = activePostIndex) {
		e.preventDefault();
		isDraggingFile = false;
		const files = e.dataTransfer?.files;
		if (files && files.length > 0) {
			handleFileUpload(files, postIndex);
		}
	}

	function removeMedia(postIndex: number, mediaIndex: number) {
		const mediaId = posts[postIndex]?.mediaIds[mediaIndex];
		posts = posts.map((p, i) =>
			i === postIndex ? { ...p, mediaIds: p.mediaIds.filter((_, mi) => mi !== mediaIndex) } : p
		);
		if (mediaId) {
			const newAlts = new Map(mediaAltTexts);
			newAlts.delete(mediaId);
			mediaAltTexts = newAlts;
		}
		scheduleAutoSave();
	}

	function setMediaAltText(mediaId: string, alt: string) {
		const newAlts = new Map(mediaAltTexts);
		if (alt.trim()) {
			newAlts.set(mediaId, alt.trim());
		} else {
			newAlts.delete(mediaId);
		}
		mediaAltTexts = newAlts;

		// Persist to backend
		(client as any)
			.PATCH('/media/{id}', {
				params: { path: { id: mediaId } },
				body: { alt_text: alt.trim() }
			})
			.catch((e: any) => {
				console.error('Failed to save alt text:', e);
			});
	}

	// --------------------------------------------------------------------------
	// Prompts
	// --------------------------------------------------------------------------
	async function fetchRandomPrompt() {
		if (!selectedWorkspaceId) return;
		loadingPrompt = true;
		try {
			const { data, error: err } = await (client as any).GET('/prompts/random', {
				params: { query: { workspace_id: selectedWorkspaceId } }
			});
			if (err) throw err;
			if (data) {
				currentPrompt = { text: data.text, category: data.category };
				showPromptCard = true;
			}
		} catch (e) {
			console.error('Failed to fetch prompt:', e);
		} finally {
			loadingPrompt = false;
		}
	}

	function dismissPrompt() {
		showPromptCard = false;
		currentPrompt = null;
	}

	// --------------------------------------------------------------------------
	// Variants
	// --------------------------------------------------------------------------
	function handleVariantChange(accountId: string, value: string) {
		const newVariants = new Map(variants);
		if (value === posts[0].content) {
			newVariants.delete(accountId);
		} else {
			newVariants.set(accountId, value);
		}
		variants = newVariants;
	}

	function toggleUnsync(accountId: string) {
		if (variants.has(accountId)) {
			const newVariants = new Map(variants);
			newVariants.delete(accountId);
			variants = newVariants;
		} else {
			const account = accounts.find((a) => a.id === accountId);
			if (account) {
				variants = new Map([...variants, [accountId, posts[0].content]]);
				showVariantsDialog = true;
			}
		}
	}

	// --------------------------------------------------------------------------
	// Scheduling
	// --------------------------------------------------------------------------
	async function suggestNextSlot() {
		if (!selectedWorkspaceId) return;
		suggestingSlot = true;
		try {
			const { data, error: err } = await (client as any).GET('/posting-schedules/next-slot', {
				params: { query: { workspace_id: selectedWorkspaceId } }
			});
			if (err) throw err;
			if (data?.slot_time) {
				// Parse date directly from ISO string to avoid timezone conversion issues
				const iso = data.slot_time as string;
				const [datePart, timePart] = iso.split('T');
				const [year, month, day] = datePart.split('-').map(Number);
				const rawHours = parseInt(timePart.split(':')[0], 10);
				const rawMinutes = parseInt(timePart.split(':')[1], 10);

				selectedDate = new CalendarDate(year, month, day);
				selectedTime = `${rawHours.toString().padStart(2, '0')}:${rawMinutes.toString().padStart(2, '0')}`;

				// Guard: if the slot is in the past, advance by one day
				const slotDateTime = selectedDate.toDate(getLocalTimeZone());
				slotDateTime.setHours(rawHours, rawMinutes, 0, 0);
				if (slotDateTime.getTime() <= Date.now()) {
					selectedDate = selectedDate.add({ days: 1 });
				}
			}
		} catch (e) {
			console.error('Failed to get next available slot:', e);
		} finally {
			suggestingSlot = false;
		}
	}

	function formatScheduledDisplay(): string {
		if (!selectedDate || !selectedTime) return 'Schedule';
		const now = today(getLocalTimeZone());
		const diffDays = selectedDate.compare(now);

		if (diffDays === 0) return `Today ${selectedTime}`;
		if (diffDays === 1) return `Tomorrow ${selectedTime}`;
		const date = selectedDate.toDate(getLocalTimeZone());
		return `${date.toLocaleDateString('en-US', { month: 'short', day: 'numeric' })} ${selectedTime}`;
	}

	// --------------------------------------------------------------------------
	// Snippets
	// --------------------------------------------------------------------------
	function setPostContent(index: number, value: string) {
		posts = posts.map((p, pi) => (pi === index ? { ...p, content: value } : p));
		scheduleAutoSave();
	}

	function setActivePost(index: number) {
		activePostIndex = index;
	}
</script>

<!-- ====================================================================== -->
<!-- Top Bar -->
<!-- ====================================================================== -->
<div class="flex flex-1 flex-col overflow-hidden">
	<div class="flex flex-wrap items-center justify-between gap-2 border-b px-3 py-2 md:px-4 md:py-3">
		<div class="flex flex-wrap items-center gap-2">
			{#if isEditMode && onCancel}
				<Button variant="ghost" size="sm" class="text-xs" onclick={onCancel}>Back</Button>
			{/if}

			<!-- Workspace selector -->
			{#if workspaces.length > 1}
				<DropdownMenu.Root>
					<DropdownMenu.Trigger>
						{#snippet child({ props })}
							<Button {...props} variant="ghost" size="sm" class="gap-1 text-xs">
								<span class="hidden text-muted-foreground sm:inline">
									{workspaces.find((w) => w.id === selectedWorkspaceId)?.name ?? 'Workspace'}
								</span>
								<span class="text-muted-foreground sm:hidden">
									{workspaces
										.find((w) => w.id === selectedWorkspaceId)
										?.name?.slice(0, 2)
										.toUpperCase() ?? 'WS'}
								</span>
								<ChevronDownIcon class="h-3 w-3" />
							</Button>
						{/snippet}
					</DropdownMenu.Trigger>
					<DropdownMenu.Content class="w-52" align="start">
						<DropdownMenu.Label class="text-xs tracking-wider text-muted-foreground uppercase"
							>Workspace</DropdownMenu.Label
						>
						<DropdownMenu.Separator />
						{#each workspaces as ws}
							<DropdownMenu.Item
								onclick={() => handleWorkspaceChange(ws.id)}
								class="gap-2 {selectedWorkspaceId === ws.id ? 'bg-muted' : ''}"
							>
								<div
									class="flex h-6 w-6 items-center justify-center rounded-md bg-primary/10 text-xs font-bold text-primary"
								>
									{ws.name.slice(0, 2).toUpperCase()}
								</div>
								<span class="text-sm">{ws.name}</span>
							</DropdownMenu.Item>
						{/each}
					</DropdownMenu.Content>
				</DropdownMenu.Root>
			{/if}

			<!-- Set selector -->
			{#if sets.length > 0}
				<div class="h-4 w-px bg-border"></div>
				<DropdownMenu.Root>
					<DropdownMenu.Trigger>
						{#snippet child({ props })}
							<Button {...props} variant="ghost" size="sm" class="gap-1 text-xs">
								<span class="text-muted-foreground"
									>{sets.find((s) => s.id === selectedSetId)?.name ?? 'All'}</span
								>
								<ChevronDownIcon class="h-3 w-3" />
							</Button>
						{/snippet}
					</DropdownMenu.Trigger>
					<DropdownMenu.Content class="w-56" align="start">
						<DropdownMenu.Label class="text-xs tracking-wider text-muted-foreground uppercase"
							>Social Set</DropdownMenu.Label
						>
						<DropdownMenu.Separator />
						<DropdownMenu.Item
							onclick={() => handleSetChange(null)}
							class="gap-2 {selectedSetId === null ? 'bg-muted' : ''}"
						>
							<div class="flex h-6 w-6 items-center justify-center rounded-full bg-muted">
								<span class="text-xs">All</span>
							</div>
							<span class="text-sm">All accounts</span>
						</DropdownMenu.Item>
						{#each sets as set}
							<DropdownMenu.Item
								onclick={() => handleSetChange(set.id)}
								class="gap-2 {selectedSetId === set.id ? 'bg-muted' : ''}"
							>
								<div
									class="flex h-6 w-6 items-center justify-center rounded-full bg-primary/10 text-xs font-bold text-primary"
								>
									{set.name.slice(0, 2).toUpperCase()}
								</div>
								<div class="flex flex-col">
									<span class="text-sm">{set.name}</span>
									<span class="text-xs text-muted-foreground"
										>{set.accounts.length} account{set.accounts.length !== 1 ? 's' : ''}</span
									>
								</div>
								{#if set.is_default}<span class="ml-auto text-xs text-muted-foreground"
										>Default</span
									>{/if}
							</DropdownMenu.Item>
						{/each}
					</DropdownMenu.Content>
				</DropdownMenu.Root>
			{/if}

			<!-- Account selector -->
			{#if accounts.length > 0}
				<DropdownMenu.Root>
					<DropdownMenu.Trigger>
						{#snippet child({ props })}
							<Button {...props} variant="ghost" size="sm" class="gap-1.5 text-xs">
								<span class="hidden text-muted-foreground sm:inline">
									{selectedAccountIds.length === accounts.length
										? 'All accounts'
										: `${selectedAccountIds.length} account${selectedAccountIds.length !== 1 ? 's' : ''}`}
								</span>
								<span class="text-muted-foreground sm:hidden"
									>{selectedAccountIds.length}/{accounts.length}</span
								>
								<ChevronDownIcon class="h-3 w-3" />
							</Button>
						{/snippet}
					</DropdownMenu.Trigger>
					<DropdownMenu.Content class="w-64" align="start">
						<div class="flex items-center justify-between px-2 py-1.5">
							<span class="text-sm font-medium text-muted-foreground">Publish to</span>
							<div class="flex gap-1">
								<Button variant="ghost" size="xs" onclick={selectAllAccounts} class="h-5 text-xs"
									>All</Button
								>
								<Button variant="ghost" size="xs" onclick={clearAllAccounts} class="h-5 text-xs"
									>None</Button
								>
							</div>
						</div>
						<DropdownMenu.Separator />
						{#each accounts as account}
							{@const isSelected = selectedAccountIds.includes(account.id)}
							{@const isUnsynced = variants.has(account.id)}
							<DropdownMenu.CheckboxItem
								checked={isSelected}
								onCheckedChange={() => toggleAccount(account.id)}
								class="gap-2"
							>
								<PlatformIcon platform={getPlatformKey(account.platform)} class="h-4 w-4" />
								<div class="flex flex-1 items-center gap-1.5">
									<span class="text-sm">{getPlatformName(account.platform)}</span>
									{#if account.account_username}<span class="text-xs text-muted-foreground"
											>@{account.account_username}</span
										>{/if}
								</div>
								{#if isUnsynced}<span class="text-xs text-amber-500">custom</span>{/if}
							</DropdownMenu.CheckboxItem>
						{/each}
						<DropdownMenu.Separator />
						<DropdownMenu.Item onclick={() => (showVariantsDialog = true)} class="gap-2">
							<UnlinkIcon class="h-3.5 w-3.5" />
							<span class="text-sm">Customize per platform</span>
						</DropdownMenu.Item>
					</DropdownMenu.Content>
				</DropdownMenu.Root>
			{/if}
		</div>

		<div class="flex flex-wrap items-center gap-1.5 md:gap-2">
			<!-- Mobile preview toggle -->
			{#if selectedAccounts.length > 0}
				<Button
					variant="ghost"
					size="icon"
					class="h-8 w-8 lg:hidden"
					onclick={() => (showMobilePreview = true)}
					title="Show preview"
				>
					<EyeIcon class="h-4 w-4" />
				</Button>
			{/if}

			<!-- Prompt -->
			<Tooltip.Root>
				<Tooltip.Trigger>
					{#snippet child({ props })}
						<Button
							{...props}
							variant="ghost"
							size="icon"
							class="h-8 w-8 {showPromptCard ? 'text-amber-500' : ''}"
							onclick={() => (showPromptCard ? dismissPrompt() : fetchRandomPrompt())}
						>
							<LightbulbIcon class="h-4 w-4" />
						</Button>
					{/snippet}
				</Tooltip.Trigger>
				<Tooltip.Content>
					<p class="text-sm">{showPromptCard ? 'Dismiss inspiration' : 'Need inspiration?'}</p>
				</Tooltip.Content>
			</Tooltip.Root>

			<!-- Suggest next slot -->
			<Tooltip.Root>
				<Tooltip.Trigger>
					{#snippet child({ props })}
						<Button
							{...props}
							variant="ghost"
							size="sm"
							class="h-8 gap-1 text-xs"
							onclick={suggestNextSlot}
							disabled={suggestingSlot || isSubmitting}
						>
							{#if suggestingSlot}<span
									class="inline-block h-1.5 w-1.5 animate-pulse rounded-full bg-current"
								></span>{:else}<ShuffleIcon class="h-3 w-3" />{/if}
							<span class="hidden sm:inline">Suggest</span>
						</Button>
					{/snippet}
				</Tooltip.Trigger>
				<Tooltip.Content><p class="text-sm">Fill next available time slot</p></Tooltip.Content>
			</Tooltip.Root>

			<!-- Schedule picker -->
			<Popover.Root bind:open={showSchedulePopover}>
				<Popover.Trigger>
					{#snippet child({ props })}
						<Button
							{...props}
							variant="outline"
							size="sm"
							class="gap-1.5 text-xs"
							disabled={isSubmitting || !hasContent}
						>
							<ClockIcon class="h-3.5 w-3.5" />
							<span class="hidden sm:inline">{formatScheduledDisplay()}</span>
						</Button>
					{/snippet}
				</Popover.Trigger>
				<Popover.Content class="w-auto max-w-[calc(100vw-2rem)] p-0" align="end">
					<div class="p-3 md:p-4">
						<div class="mb-3 flex items-center justify-between">
							<span class="text-sm font-medium">Schedule</span>
						</div>
						<Calendar
							type="single"
							bind:value={selectedDate}
							minValue={today(getLocalTimeZone())}
							class="bg-transparent p-0 [--cell-size:--spacing(8)]"
							weekdayFormat="short"
							weekStartsOn={workspaceCtx.weekStartsOn}
						/>
						<div class="mt-3 max-h-48 overflow-y-auto">
							<div class="grid grid-cols-3 gap-1.5 sm:grid-cols-4">
								{#each timeSlots as time}
									<Button
										variant={selectedTime === time ? 'default' : 'outline'}
										size="sm"
										onclick={() => {
											selectedTime = time;
											showSchedulePopover = false;
										}}
										class="h-8 text-xs"
									>
										{time}
									</Button>
								{/each}
							</div>
						</div>
					</div>
				</Popover.Content>
			</Popover.Root>

			<!-- Schedule button -->
			<Button
				size="sm"
				class="gap-1.5"
				onclick={() => publish(false)}
				disabled={isSubmitting || !hasContent || selectedAccountIds.length === 0}
			>
				{#if isSubmitting}<LoaderIcon class="h-3.5 w-3.5 animate-spin" />{:else}<SendIcon
						class="h-3.5 w-3.5"
					/>{/if}
				<span class="hidden sm:inline">Schedule</span>
			</Button>

			<!-- Publish now -->
			<Button
				size="sm"
				variant="secondary"
				onclick={() => publish(true)}
				disabled={isSubmitting || !hasContent || selectedAccountIds.length === 0}
				class="gap-1.5"
			>
				{#if isSubmitting}<LoaderIcon class="h-3.5 w-3.5 animate-spin" />{/if}
				<span class="hidden sm:inline">Publish Now</span>
				<span class="sm:hidden">Now</span>
			</Button>
		</div>
	</div>

	<!-- ====================================================================== -->
	<!-- Messages -->
	<!-- ====================================================================== -->
	{#if error}
		<div
			class="mx-3 mt-2 rounded-md border border-destructive/20 bg-destructive/10 px-3 py-2 text-sm text-destructive md:mx-4 md:mt-3"
		>
			{error}
		</div>
	{/if}
	{#if success}
		<div
			class="mx-3 mt-2 rounded-md border border-green-500/20 bg-green-500/10 px-3 py-2 text-sm text-green-600 md:mx-4 md:mt-3"
		>
			{success}
		</div>
	{/if}

	<!-- ====================================================================== -->
	<!-- Main Content Area -->
	<!-- ====================================================================== -->
	<div class="flex flex-1 overflow-hidden">
		<!-- Compose Column -->
		<div class="flex flex-1 flex-col overflow-y-auto">
			<div class="mx-auto w-full max-w-2xl px-3 py-4 md:px-6 md:py-6">
				<!-- Prompt Card -->
				{#if showPromptCard}
					<div class="relative mb-5 rounded border bg-muted/30 p-4">
						<div class="absolute top-2 right-2 flex items-center gap-1">
							<button
								type="button"
								class="rounded p-1 text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
								onclick={fetchRandomPrompt}
								disabled={loadingPrompt}
								title="Shuffle"
							>
								<ShuffleIcon class="h-3.5 w-3.5" />
							</button>
							<button
								type="button"
								class="rounded p-1 text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
								onclick={dismissPrompt}
								title="Close"
							>
								<XIcon class="h-3.5 w-3.5" />
							</button>
						</div>
						{#if loadingPrompt}
							<div class="space-y-2 py-2">
								<Skeleton class="h-3 w-full" />
								<Skeleton class="h-3 w-3/4" />
							</div>
						{:else if currentPrompt}
							<p class="text-sm leading-relaxed text-foreground/80">{currentPrompt.text}</p>
						{:else}
							<p class="text-sm text-muted-foreground">No prompts available.</p>
						{/if}
					</div>
				{/if}

				<!-- Posts -->
				<div class="space-y-0">
					<ReorderableList
						items={posts}
						getKey={(post) => post.key}
						onUpdate={handleReorder}
						cssSelectorHandle=".drag-handle"
						direction="vertical"
					>
						{#snippet item(post, i)}
							<div
								class="group/post relative {isDraggingFile && activePostIndex === i
									? 'bg-primary/5'
									: ''}"
								role="region"
								aria-label="Drop zone for post {i + 1}"
								ondragover={handleDragOver}
								ondragleave={handleDragLeave}
								ondrop={(e) => handleDrop(e, i)}
							>
								{#if isThread && i < posts.length - 1}
									<div class="absolute top-0 bottom-0 left-3 w-px bg-border"></div>
								{/if}

								<div class="relative flex gap-3 {isThread ? 'pl-7' : ''}">
									{#if isThread}
										<div class="relative flex flex-col items-center pt-3">
											<button
												type="button"
												class="drag-handle -ml-4 cursor-grab rounded p-0.5 text-muted-foreground opacity-0 transition-opacity group-hover/post:opacity-60 hover:opacity-100 active:cursor-grabbing"
												title="Drag to reorder"
											>
												<GripVerticalIcon class="h-4 w-4" />
											</button>
										</div>
									{/if}

									<div class="min-w-0 flex-1">
										<div class="relative">
											<textarea
												id="post-textarea-{i}"
												use:textareaAction={i}
												value={post.content}
												oninput={(e) => {
													const target = e.target as HTMLTextAreaElement;
													setPostContent(i, target.value);
													autoResize(target);
												}}
												onpaste={(e) => handlePaste(e, i)}
												onfocus={() => setActivePost(i)}
												placeholder={i === 0 ? "What's on your mind?" : 'Add to your thread...'}
												class="w-full resize-none border-0 bg-transparent py-2 pr-3 text-base leading-relaxed text-foreground placeholder:text-muted-foreground/50 focus:ring-0 focus:outline-none md:py-3 md:pr-4 md:text-lg"
												style="min-height: {i === 0 ? '120px' : '56px'};"
												disabled={isSubmitting}
											></textarea>

											{#if isUploading && activePostIndex === i}
												<div
													class="absolute inset-0 flex items-center justify-center bg-background/80"
												>
													<LoaderIcon class="h-5 w-5 animate-spin text-primary" />
												</div>
											{/if}
										</div>

										<!-- Media grid -->
										{#if post.mediaIds.length > 0}
											<div
												class="mb-3 {post.mediaIds.length === 1 ? '' : 'grid grid-cols-2 gap-1.5'}"
											>
												{#each post.mediaIds as mediaId, mi}
													{@const isFirstOfThree = post.mediaIds.length === 3 && mi === 0}
													<div
														class="group/media relative overflow-hidden rounded-lg {isFirstOfThree
															? 'col-span-2'
															: ''}"
													>
														<img
															src="{getMediaBase()}/media/{mediaId}"
															alt={mediaAltTexts.get(mediaId) || ''}
															class="{post.mediaIds.length === 1
																? 'aspect-video'
																: 'aspect-square'} w-full object-cover"
														/>
														<div
															class="absolute inset-0 flex items-start justify-end gap-1 bg-black/0 p-2 opacity-0 transition-all group-hover/media:bg-black/40 group-hover/media:opacity-100"
														>
															<button
																type="button"
																class="rounded-md bg-black/60 px-2 py-1 text-xs text-white backdrop-blur-sm transition-colors hover:bg-black/80"
																onclick={(e) => {
																	e.stopPropagation();
																	editingAltMediaId =
																		editingAltMediaId === mediaId ? null : mediaId;
																}}
															>
																<TypeIcon class="h-3 w-3" />
															</button>
															<button
																type="button"
																class="rounded-md bg-black/60 p-1 text-white backdrop-blur-sm transition-colors hover:bg-red-500/80"
																onclick={(e) => {
																	e.stopPropagation();
																	removeMedia(i, mi);
																}}
															>
																<XIcon class="h-3 w-3" />
															</button>
														</div>
														{#if editingAltMediaId === mediaId}
															<div
																class="absolute inset-x-0 bottom-0 bg-black/70 p-2 backdrop-blur-sm"
															>
																<textarea
																	value={mediaAltTexts.get(mediaId) || ''}
																	oninput={(e) =>
																		setMediaAltText(
																			mediaId,
																			(e.target as HTMLTextAreaElement).value
																		)}
																	placeholder="Alt text..."
																	rows={2}
																	class="w-full resize-none rounded bg-white/10 px-2 py-1 text-xs text-white placeholder:text-white/50 focus:outline-none"
																></textarea>
																<div class="mt-1 flex justify-end gap-1">
																	<button
																		type="button"
																		class="text-[10px] text-white/70 hover:text-white"
																		onclick={() => (editingAltMediaId = null)}>Done</button
																	>
																</div>
															</div>
														{/if}
													</div>
												{/each}
											</div>
										{/if}

										<!-- Bottom bar -->
										<div
											class="flex items-center gap-2 pb-2 transition-opacity {activePostIndex === i
												? 'opacity-100'
												: 'pointer-events-none opacity-0'}"
										>
											{#if isThread}<span
													class="text-[10px] font-medium text-muted-foreground/60 tabular-nums"
													>#{i + 1}</span
												>{/if}

											<label class="cursor-pointer">
												<input
													type="file"
													accept="image/*,video/*"
													multiple
													class="hidden"
													onchange={(e) => {
														const target = e.target as HTMLInputElement;
														if (target.files) handleFileUpload(target.files, i);
													}}
												/>
												<div
													class="flex h-7 w-7 items-center justify-center rounded-md text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
												>
													<ImageIcon class="h-3.5 w-3.5" />
												</div>
											</label>

											<Tooltip.Root>
												<Tooltip.Trigger>
													{#snippet child({ props })}
														<div {...props} class="flex cursor-default items-center gap-1.5">
															<svg
																class="h-4 w-4 {getCharCounterColor(post.content.length, maxChars)}"
																viewBox="0 0 20 20"
															>
																<circle
																	cx="10"
																	cy="10"
																	r="8"
																	fill="none"
																	stroke="currentColor"
																	stroke-width="2.5"
																	opacity="0.15"
																/>
																<circle
																	cx="10"
																	cy="10"
																	r="8"
																	fill="none"
																	stroke={getCharCounterStrokeColor(post.content.length, maxChars)}
																	stroke-width="2.5"
																	stroke-linecap="round"
																	stroke-dasharray={50.27}
																	stroke-dashoffset={50.27 *
																		Math.max(0, 1 - post.content.length / maxChars)}
																	transform="rotate(-90 10 10)"
																/>
															</svg>
															<span class="text-[10px] text-muted-foreground/60 tabular-nums"
																>{post.content.length}/{maxChars}</span
															>
														</div>
													{/snippet}
												</Tooltip.Trigger>
												<Tooltip.Content class="w-48">
													<div class="space-y-1">
														<p class="text-xs font-medium text-muted-foreground">
															Character limits
														</p>
														{#each selectedPlatformLimits as pl}
															<div class="flex items-center justify-between gap-2 text-xs">
																<div class="flex items-center gap-1.5">
																	<PlatformIcon platform={pl.key} class="h-3 w-3" /><span
																		>{pl.platform}</span
																	>
																</div>
																<span
																	class="tabular-nums {post.content.length > pl.limit
																		? 'text-red-500'
																		: 'text-muted-foreground'}"
																	>{post.content.length}/{pl.limit}</span
																>
															</div>
														{/each}
													</div>
												</Tooltip.Content>
											</Tooltip.Root>

											<button
												type="button"
												class="flex items-center gap-1.5 text-xs text-muted-foreground/60 transition-colors hover:text-foreground"
												onclick={addPost}
											>
												<PlusIcon class="h-3 w-3" />Add post
											</button>
										</div>

										{#if isThread}
											<button
												type="button"
												class="absolute top-3 right-0 rounded p-1 text-muted-foreground opacity-0 transition-opacity group-hover/post:opacity-100 hover:bg-muted hover:text-destructive"
												onclick={() => removePost(i)}
												title="Remove post"
											>
												<Trash2Icon class="h-3.5 w-3.5" />
											</button>
										{/if}
									</div>
								</div>
							</div>
						{/snippet}
					</ReorderableList>
				</div>
			</div>
		</div>

		<!-- Preview Column -->
		{#if showPreview && selectedAccounts.length > 0}
			<div class="hidden w-[420px] border-l bg-muted/20 px-6 py-6 lg:block">
				<div class="sticky top-6">
					<div class="mb-4 flex items-center justify-between">
						<span class="text-xs font-medium tracking-wider text-muted-foreground uppercase"
							>Preview</span
						>
						<button
							type="button"
							class="text-xs text-muted-foreground hover:text-foreground"
							onclick={() => (showPreview = false)}>Hide</button
						>
					</div>
					<div class="space-y-5">
						{#each selectedPlatformLimits as pl (pl.key)}
							{@const account = selectedAccounts.find((a) => getPlatformKey(a.platform) === pl.key)}
							{#if account}
								<div>
									<div class="mb-1.5 flex items-center gap-1.5 text-xs text-muted-foreground">
										<PlatformIcon platform={pl.key} class="h-3 w-3" />
										<span>{pl.platform}</span>
										{#if account.account_username}<span class="text-muted-foreground/60"
												>@{account.account_username}</span
											>{/if}
									</div>
									{#if isThread}
										<div class="space-y-3">
											{#each posts.filter((p) => p.content.trim().length > 0 || p.mediaIds.length > 0) as post, pi}
												<PlatformPreview
													platform={pl.key}
													content={variants.get(account.id) ?? post.content}
													mediaIds={post.mediaIds}
													username={account.account_username || 'username'}
													displayName={account.account_username || 'Display Name'}
												/>
											{/each}
										</div>
									{:else}
										<PlatformPreview
											platform={pl.key}
											content={activePost.content}
											mediaIds={activePost.mediaIds}
											username={account.account_username || 'username'}
											displayName={account.account_username || 'Display Name'}
											variantContent={variants.get(account.id) ?? null}
											isUnsynced={variants.has(account.id)}
										/>
									{/if}
								</div>
							{/if}
						{/each}
					</div>
				</div>
			</div>
		{:else if !showPreview && selectedAccounts.length > 0}
			<div
				class="hidden w-10 border-l bg-muted/20 lg:flex lg:items-start lg:justify-center lg:pt-4"
			>
				<button
					type="button"
					class="text-muted-foreground hover:text-foreground"
					onclick={() => (showPreview = true)}
					title="Show preview"
				>
					<PlatformIcon platform={getPlatformKey(selectedAccounts[0].platform)} class="h-4 w-4" />
				</button>
			</div>
		{/if}
	</div>
</div>

<!-- ====================================================================== -->
<!-- Mobile Preview Sheet -->
<!-- ====================================================================== -->
<Sheet.Root bind:open={showMobilePreview}>
	<Sheet.Content side="bottom" class="h-[85vh] rounded-t-xl p-0">
		<Sheet.Header class="border-b px-4 py-3">
			<Sheet.Title class="text-sm font-medium">Preview</Sheet.Title>
		</Sheet.Header>
		<div class="overflow-y-auto px-4 py-4">
			<div class="space-y-5">
				{#each selectedPlatformLimits as pl (pl.key)}
					{@const account = selectedAccounts.find((a) => getPlatformKey(a.platform) === pl.key)}
					{#if account}
						<div>
							<div class="mb-1.5 flex items-center gap-1.5 text-xs text-muted-foreground">
								<PlatformIcon platform={pl.key} class="h-3 w-3" />
								<span>{pl.platform}</span>
								{#if account.account_username}<span class="text-muted-foreground/60"
										>@{account.account_username}</span
									>{/if}
							</div>
							{#if isThread}
								<div class="space-y-3">
									{#each posts.filter((p) => p.content.trim().length > 0 || p.mediaIds.length > 0) as post, pi}
										<PlatformPreview
											platform={pl.key}
											content={variants.get(account.id) ?? post.content}
											mediaIds={post.mediaIds}
											username={account.account_username || 'username'}
											displayName={account.account_username || 'Display Name'}
										/>
									{/each}
								</div>
							{:else}
								<PlatformPreview
									platform={pl.key}
									content={activePost.content}
									mediaIds={activePost.mediaIds}
									username={account.account_username || 'username'}
									displayName={account.account_username || 'Display Name'}
									variantContent={variants.get(account.id) ?? null}
									isUnsynced={variants.has(account.id)}
								/>
							{/if}
						</div>
					{/if}
				{/each}
			</div>
		</div>
	</Sheet.Content>
</Sheet.Root>

<!-- ====================================================================== -->
<!-- Variants Dialog -->
<!-- ====================================================================== -->
<Dialog.Root bind:open={showVariantsDialog}>
	<Dialog.Content class="max-w-[calc(100vw-2rem)] sm:max-w-lg">
		<Dialog.Header>
			<Dialog.Title class="flex items-center gap-2">
				<UnlinkIcon class="h-4 w-4" />Customize per platform
			</Dialog.Title>
		</Dialog.Header>
		<div class="space-y-4 py-2">
			<p class="text-sm text-muted-foreground">
				Override content for specific platforms. Leave empty to use the default content.
			</p>
			{#each selectedAccountIds as accId}
				{@const account = accounts.find((a) => a.id === accId)}
				{#if account}
					<div class="space-y-1.5">
						<div class="flex items-center gap-2">
							<PlatformIcon platform={getPlatformKey(account.platform)} class="h-4 w-4" />
							<span class="text-sm font-medium">{getPlatformName(account.platform)}</span>
							{#if variants.has(account.id)}<span
									class="rounded bg-primary/10 px-1.5 py-0.5 text-xs text-primary">Customized</span
								>{/if}
						</div>
						<textarea
							value={variants.get(accId) ?? posts[0].content}
							oninput={(e) => handleVariantChange(accId, (e.target as HTMLTextAreaElement).value)}
							rows={3}
							placeholder="Use default content..."
							class="w-full resize-none rounded-md border border-border bg-transparent px-3 py-2 text-sm outline-none focus:border-primary"
						></textarea>
						<div class="flex justify-end">
							<span class="text-sm text-muted-foreground"
								>{(variants.get(accId) ?? posts[0].content).length} characters</span
							>
						</div>
					</div>
				{/if}
			{/each}
		</div>
		<Dialog.Footer>
			<Button onclick={() => (showVariantsDialog = false)} size="sm">Done</Button>
		</Dialog.Footer>
	</Dialog.Content>
</Dialog.Root>
