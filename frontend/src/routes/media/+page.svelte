<script lang="ts">
	import { onMount } from 'svelte';
	import { client, type Workspace, getToken } from '$lib/api/client';
	import { getApiBase } from '$lib/stores/instance.svelte';
	import { workspaceCtx } from '$lib/stores/workspace.svelte';
	import { Button } from '$lib/components/ui/button';
	import { Checkbox } from '$lib/components/ui/checkbox';
	import * as Select from '$lib/components/ui/select';
	import * as Dialog from '$lib/components/ui/dialog';
	import PageContainer from '$lib/components/page-container.svelte';
	import EmptyState from '$lib/components/empty-state.svelte';
	import LoaderIcon from 'lucide-svelte/icons/loader-2';
	import ImageIcon from 'lucide-svelte/icons/image';
	import { Skeleton } from '$lib/components/ui/skeleton';
	import VideoIcon from 'lucide-svelte/icons/video';
	import HeartIcon from 'lucide-svelte/icons/heart';
	import TrashIcon from 'lucide-svelte/icons/trash-2';
	import UploadIcon from 'lucide-svelte/icons/upload';
	import XIcon from 'lucide-svelte/icons/x';
	import ExternalLinkIcon from 'lucide-svelte/icons/external-link';
	import CheckIcon from 'lucide-svelte/icons/check';
	import ChevronLeftIcon from 'lucide-svelte/icons/chevron-left';
	import ChevronRightIcon from 'lucide-svelte/icons/chevron-right';
	import Grid2X2Icon from 'lucide-svelte/icons/grid-2x2';

	interface MediaItem {
		id: string;
		workspace_id: string;
		mime_type: string;
		size: number;
		original_filename: string;
		width: number;
		height: number;
		alt_text: string;
		is_favorite: boolean;
		created_at: string;
		url: string;
		thumbnail_url: string;
		usage_count: number;
		processing_status: string;
	}

	interface MediaUsage {
		post_id: string;
		content: string;
		status: string;
		scheduled: string;
	}

	interface BatchDeleteResult {
		deleted: number;
		failed_ids: string[];
	}

	let workspaces = $state<Workspace[]>([]);
	let selectedWorkspaceId = $state('');
	let loading = $state(true);
	let error = $state('');
	let toastMessage = $state('');

	let mediaItems = $state<MediaItem[]>([]);
	let mediaLoading = $state(false);
	let totalCount = $state(0);
	let currentPage = $state(0);
	const pageSize = 40;

	let filter = $state<string>('all');
	let sort = $state<string>('newest');

	let uploadDialogOpen = $state(false);
	let uploadLoading = $state(false);
	let uploadError = $state('');
	let uploadProgress = $state('');

	let usageDialogOpen = $state(false);
	let selectedMedia = $state<MediaItem | null>(null);
	let mediaUsage = $state<MediaUsage[]>([]);
	let usageLoading = $state(false);

	let selectedMediaIds = $state<Set<string>>(new Set());
	let isSelectionMode = $state(false);

	async function loadWorkspaces() {
		try {
			const { data } = await client.GET('/workspaces', {});
			workspaces = data ?? [];
			if (workspaces.length > 0 && !selectedWorkspaceId) {
				selectedWorkspaceId = workspaces[0].id;
			}
		} catch (e) {
			console.error('Failed to load workspaces:', e);
		} finally {
			loading = false;
		}
	}

	async function loadMedia() {
		if (!selectedWorkspaceId) return;
		mediaLoading = true;
		error = '';
		selectedMediaIds.clear();
		isSelectionMode = false;
		try {
			const { data, error: err } = await (client as any).GET('/media', {
				params: {
					query: {
						workspace_id: selectedWorkspaceId,
						filter: filter,
						sort: sort,
						limit: pageSize,
						offset: currentPage * pageSize
					}
				}
			});
			if (err) throw new Error(err.detail || 'Failed to load media');
			mediaItems = (data?.media ?? []) as unknown as MediaItem[];
			totalCount = data?.total ?? 0;
		} catch (e) {
			error = (e as Error).message;
			mediaItems = [];
		} finally {
			mediaLoading = false;
		}
	}

	async function toggleFavorite(mediaId: string) {
		try {
			const { data, error: err } = await (client as any).PATCH('/media/{id}/favorite', {
				params: { path: { id: mediaId } }
			});
			if (err) throw new Error(err.detail || 'Failed to update favorite');
			const item = mediaItems.find((m) => m.id === mediaId);
			if (item) {
				item.is_favorite = data?.is_favorite ?? !item.is_favorite;
			}
		} catch (e) {
			toastMessage = (e as Error).message;
		}
	}

	async function toggleFavoriteBatch() {
		const ids = Array.from(selectedMediaIds);
		for (const id of ids) {
			await toggleFavorite(id);
		}
		selectedMediaIds.clear();
		isSelectionMode = false;
	}

	async function deleteMedia(mediaId: string) {
		if (!confirm('Delete this media? This cannot be undone.')) return;
		try {
			const { error: err } = await (client as any).DELETE('/media/{id}', {
				params: { path: { id: mediaId } }
			});
			if (err) throw new Error(err.detail || 'Failed to delete media');
			mediaItems = mediaItems.filter((m) => m.id !== mediaId);
			totalCount--;
			toastMessage = 'Media deleted successfully';
		} catch (e) {
			toastMessage = (e as Error).message;
		}
	}

	async function deleteSelectedBatch() {
		if (selectedMediaIds.size === 0) return;
		if (!confirm(`Delete ${selectedMediaIds.size} selected media items? This cannot be undone.`))
			return;

		try {
			const { data, error: err } = await (client as any).POST('/media/batch-delete', {
				body: {
					media_ids: Array.from(selectedMediaIds)
				}
			});
			if (err) throw new Error(err.detail || 'Failed to delete media');

			const result = data as BatchDeleteResult;
			mediaItems = mediaItems.filter(
				(m) => !result.deleted || !selectedMediaIds.has(m.id) || result.failed_ids?.includes(m.id)
			);
			totalCount -= result.deleted;
			toastMessage = `Deleted ${result.deleted} media items`;
			selectedMediaIds.clear();
			isSelectionMode = false;
			await loadMedia();
		} catch (e) {
			toastMessage = (e as Error).message;
		}
	}

	async function showUsage(media: MediaItem) {
		selectedMedia = media;
		usageDialogOpen = true;
		usageLoading = true;
		mediaUsage = [];
		try {
			const { data, error: err } = await (client as any).GET('/media/{id}/usage', {
				params: { path: { id: media.id } }
			});
			if (err) throw new Error(err.detail || 'Failed to load usage');
			mediaUsage = (data?.usage ?? []) as unknown as MediaUsage[];
		} catch (e) {
			toastMessage = (e as Error).message;
		} finally {
			usageLoading = false;
		}
	}

	async function handleUpload() {
		if (!selectedWorkspaceId) return;
		uploadLoading = true;
		uploadError = '';
		uploadProgress = 'Uploading...';

		const fileInput = document.getElementById('file-upload') as HTMLInputElement;
		if (!fileInput?.files?.length) {
			uploadError = 'Please select a file';
			uploadLoading = false;
			return;
		}

		const file = fileInput.files[0];
		const formData = new FormData();
		formData.append('workspace_id', selectedWorkspaceId);
		formData.append('file', file);

		try {
			const token = getToken();
			const response = await fetch(`${getApiBase()}/media/upload`, {
				method: 'POST',
				headers: token ? { Authorization: `Bearer ${token}` } : {},
				body: formData
			});

			if (!response.ok) {
				const errData = await response.json();
				throw new Error(errData.error || 'Upload failed');
			}

			uploadDialogOpen = false;
			fileInput.value = '';
			toastMessage = 'File uploaded successfully';
			await loadMedia();
		} catch (e) {
			uploadError = (e as Error).message;
		} finally {
			uploadLoading = false;
			uploadProgress = '';
		}
	}

	async function handleBatchUpload() {
		if (!selectedWorkspaceId) return;
		uploadLoading = true;
		uploadError = '';
		uploadProgress = 'Uploading...';

		const fileInput = document.getElementById('batch-file-upload') as HTMLInputElement;
		if (!fileInput?.files?.length) {
			uploadError = 'Please select files';
			uploadLoading = false;
			return;
		}

		const formData = new FormData();
		formData.append('workspace_id', selectedWorkspaceId);
		for (const file of fileInput.files) {
			formData.append('files', file);
		}

		try {
			const token = getToken();
			const response = await fetch(`${getApiBase()}/media/batch-upload`, {
				method: 'POST',
				headers: token ? { Authorization: `Bearer ${token}` } : {},
				body: formData
			});

			if (!response.ok) {
				const errData = await response.json();
				throw new Error(errData.error || 'Upload failed');
			}

			const result = await response.json();
			uploadDialogOpen = false;
			fileInput.value = '';
			toastMessage = `Uploaded ${result.uploaded?.length || 0} files`;
			await loadMedia();
		} catch (e) {
			uploadError = (e as Error).message;
		} finally {
			uploadLoading = false;
			uploadProgress = '';
		}
	}

	function formatSize(bytes: number): string {
		if (bytes < 1024) return bytes + ' B';
		if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB';
		return (bytes / (1024 * 1024)).toFixed(1) + ' MB';
	}

	function formatDate(dateStr: string): string {
		const date = new Date(dateStr);
		return date.toLocaleDateString('en-US', {
			month: 'short',
			day: 'numeric',
			timeZone: workspaceCtx.settings.timezone || 'UTC'
		});
	}

	function isImage(mimeType: string): boolean {
		return mimeType.startsWith('image/');
	}

	function toggleSelection(mediaId: string) {
		if (selectedMediaIds.has(mediaId)) {
			selectedMediaIds.delete(mediaId);
		} else {
			selectedMediaIds.add(mediaId);
		}
		selectedMediaIds = new Set(selectedMediaIds);
		isSelectionMode = selectedMediaIds.size > 0;
	}

	function selectAll() {
		const unusedMedia = mediaItems.filter((m) => m.usage_count === 0);
		if (unusedMedia.length === selectedMediaIds.size) {
			selectedMediaIds.clear();
		} else {
			unusedMedia.forEach((m) => selectedMediaIds.add(m.id));
		}
		selectedMediaIds = new Set(selectedMediaIds);
		isSelectionMode = selectedMediaIds.size > 0;
	}

	function cancelSelection() {
		selectedMediaIds.clear();
		selectedMediaIds = new Set(selectedMediaIds);
		isSelectionMode = false;
	}

	function nextPage() {
		if ((currentPage + 1) * pageSize < totalCount) {
			currentPage++;
			loadMedia();
		}
	}

	function prevPage() {
		if (currentPage > 0) {
			currentPage--;
			loadMedia();
		}
	}

	onMount(() => {
		loadWorkspaces();
	});

	$effect(() => {
		if (selectedWorkspaceId) {
			currentPage = 0;
			loadMedia();
		}
	});

	$effect(() => {
		const _trigger = [filter, sort];
		if (_trigger && selectedWorkspaceId) {
			currentPage = 0;
			loadMedia();
		}
	});

	const filterTabs = [
		{ value: 'all', label: 'All' },
		{ value: 'used', label: 'Used' },
		{ value: 'unused', label: 'Unused' },
		{ value: 'favorites', label: 'Favorites' }
	];

	const totalPages = $derived(Math.ceil(totalCount / pageSize));
	const unusedCount = $derived(mediaItems.filter((m) => m.usage_count === 0).length);

	const descriptionText = $derived.by(() => {
		if (totalCount > 0) {
			let text = `${totalCount} file${totalCount !== 1 ? 's' : ''}`;
			if (filter === 'unused') {
				text += ` (${unusedCount} unused)`;
			}
			return text;
		}
		return 'Manage your media attachments';
	});
</script>

<svelte:head>
	<title>Media Library - OpenPost</title>
</svelte:head>

{#if toastMessage}
	<div
		class="pointer-events-auto fixed right-4 bottom-4 z-50 mb-4 flex items-center gap-2 rounded-lg border bg-background px-4 py-3 shadow-lg"
	>
		<span class="text-sm">{toastMessage}</span>
		<button onclick={() => (toastMessage = '')}>
			<XIcon class="size-4" />
		</button>
	</div>
{/if}

<PageContainer
	title="Media Library"
	description={descriptionText}
	icon={ImageIcon}
	{loading}
	loadingMessage="Loading media library..."
>
	{#snippet actions()}
		<div class="flex items-center gap-2">
			{#if workspaces && workspaces.length > 1}
				<Select.Root type="single" bind:value={selectedWorkspaceId}>
					<Select.Trigger class="w-[160px]">
						{workspaces.find((w) => w.id === selectedWorkspaceId)?.name || 'Workspace'}
					</Select.Trigger>
					<Select.Content>
						{#each workspaces as workspace (workspace.id)}
							<Select.Item value={workspace.id}>{workspace.name}</Select.Item>
						{/each}
					</Select.Content>
				</Select.Root>
			{/if}
			<Button onclick={() => (uploadDialogOpen = true)} class="gap-2">
				<UploadIcon class="h-4 w-4" />
				Upload
			</Button>
		</div>
	{/snippet}

	{#if error}
		<div
			class="mb-4 flex items-center gap-2 rounded-lg border border-destructive/20 bg-destructive/10 p-3 text-sm text-destructive"
		>
			{error}
			<button class="ml-auto" onclick={() => (error = '')}>
				<XIcon class="size-4" />
			</button>
		</div>
	{/if}

	<!-- Filter Tabs + Sort -->
	<div class="mb-6 flex flex-wrap items-center gap-4">
		<div class="flex items-center gap-0.5 rounded-lg border bg-muted/30 p-1">
			{#each filterTabs as tab}
				<button
					class="rounded-md px-3 py-1.5 text-sm font-medium transition-colors {filter === tab.value
						? 'bg-background text-foreground shadow-sm'
						: 'text-muted-foreground hover:text-foreground'}"
					onclick={() => (filter = tab.value)}
				>
					{tab.label}
				</button>
			{/each}
		</div>

		<div class="ml-auto flex items-center gap-2">
			<Select.Root type="single" bind:value={sort}>
				<Select.Trigger class="h-9 w-[120px] text-sm">
					{sort === 'newest' ? 'Newest' : sort === 'oldest' ? 'Oldest' : 'Size'}
				</Select.Trigger>
				<Select.Content>
					<Select.Item value="newest">Newest</Select.Item>
					<Select.Item value="oldest">Oldest</Select.Item>
					<Select.Item value="size">Size</Select.Item>
				</Select.Content>
			</Select.Root>
		</div>
	</div>

	<!-- Selection Toolbar -->
	{#if isSelectionMode}
		<div class="mb-4 flex items-center gap-4 rounded-lg border bg-muted/50 p-3">
			<span class="text-sm font-medium">{selectedMediaIds.size} selected</span>
			{#if unusedCount > 0}
				<Button variant="outline" size="sm" onclick={selectAll}>
					{unusedCount === selectedMediaIds.size ? 'Deselect All' : 'Select All Unused'}
				</Button>
			{/if}
			<div class="ml-auto flex items-center gap-2">
				<Button variant="outline" size="sm" onclick={toggleFavoriteBatch}>
					<HeartIcon class="mr-1 h-4 w-4" />
					Toggle Favorite
				</Button>
				{#if selectedMediaIds.size > 0}
					<Button variant="destructive" size="sm" onclick={deleteSelectedBatch}>
						<TrashIcon class="mr-1 h-4 w-4" />
						Delete Selected
					</Button>
				{/if}
				<Button variant="ghost" size="sm" onclick={cancelSelection}>Cancel</Button>
			</div>
		</div>
	{/if}

	<!-- Media Grid -->
	{#if mediaLoading}
		<div class="grid grid-cols-2 gap-4 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5">
			{#each Array(10) as _}
				<div class="space-y-2">
					<Skeleton class="aspect-square rounded-lg" />
					<Skeleton class="h-3 w-3/4" />
					<Skeleton class="h-3 w-1/2" />
				</div>
			{/each}
		</div>
	{:else if mediaItems.length === 0}
		{#if filter !== 'all'}
			<EmptyState
				icon={ImageIcon}
				title="No media found"
				description="Try changing your filters"
				actionLabel="Show All"
				onAction={() => (filter = 'all')}
				variant="dashed"
				size="lg"
			/>
		{:else}
			<EmptyState
				icon={ImageIcon}
				title="No media found"
				description="Upload some files to get started"
				actionLabel="Upload"
				onAction={() => (uploadDialogOpen = true)}
				variant="dashed"
				size="lg"
			/>
		{/if}
	{:else}
		<div class="grid grid-cols-2 gap-4 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5">
			{#each mediaItems as media (media.id)}
				<div
					class="group relative overflow-hidden rounded-lg border bg-card transition-all hover:shadow-sm {selectedMediaIds.has(
						media.id
					)
						? 'ring-2 ring-primary'
						: ''}"
				>
					<div class="relative aspect-square overflow-hidden bg-muted/30">
						{#if isImage(media.mime_type)}
							<img
								src={media.thumbnail_url || media.url}
								alt={media.alt_text || 'Media'}
								class="size-full object-cover transition-transform group-hover:scale-105"
							/>
						{:else}
							<div class="flex size-full items-center justify-center">
								<VideoIcon class="size-10 text-muted-foreground/40" />
							</div>
						{/if}

						<!-- Selection checkbox -->
						<button
							class="absolute top-2 left-2 rounded-md bg-background/80 p-1 opacity-0 transition-opacity group-hover:opacity-100 hover:bg-background {!isSelectionMode &&
							media.usage_count === 0
								? 'opacity-0 group-hover:opacity-100'
								: ''}"
							onclick={(e) => {
								e.stopPropagation();
								toggleSelection(media.id);
							}}
						>
							{#if selectedMediaIds.has(media.id)}
								<CheckIcon class="size-4 text-primary" />
							{:else}
								<div class="size-4 rounded-sm border-2 border-muted-foreground"></div>
							{/if}
						</button>

						<!-- Hover Actions -->
						<div
							class="absolute inset-0 flex items-center justify-center gap-2 bg-black/40 opacity-0 transition-opacity group-hover:opacity-100"
						>
							<button
								class="rounded-full bg-white/20 p-2 backdrop-blur-sm transition-colors hover:bg-white/30"
								onclick={() => showUsage(media)}
								title="View usage"
							>
								<ExternalLinkIcon class="size-4 text-white" />
							</button>
							<button
								class="rounded-full bg-white/20 p-2 backdrop-blur-sm transition-colors hover:bg-white/30"
								onclick={() => toggleFavorite(media.id)}
								title={media.is_favorite ? 'Unfavorite' : 'Favorite'}
							>
								<HeartIcon
									class="size-4 text-white"
									fill={media.is_favorite ? 'currentColor' : 'none'}
								/>
							</button>
							{#if media.usage_count === 0}
								<button
									class="rounded-full bg-white/20 p-2 backdrop-blur-sm transition-colors hover:bg-red-500/80"
									onclick={() => deleteMedia(media.id)}
									title="Delete"
								>
									<TrashIcon class="size-4 text-white" />
								</button>
							{/if}
						</div>

						{#if media.is_favorite}
							<div class="absolute top-2 right-2">
								<HeartIcon class="size-4 fill-red-500 text-red-500 drop-shadow-sm" />
							</div>
						{/if}
					</div>

					<div class="p-2.5">
						{#if media.original_filename}
							<p class="truncate text-sm font-medium" title={media.original_filename}>
								{media.original_filename}
							</p>
						{/if}
						<p class="truncate text-sm text-muted-foreground">
							{formatSize(media.size)} · {formatDate(media.created_at)}
							{#if media.width && media.height}
								· {media.width}×{media.height}
							{/if}
						</p>
						<div class="mt-1.5">
							{#if media.usage_count > 0}
								<span
									class="inline-flex items-center rounded-full bg-primary/10 px-2 py-0.5 text-xs font-medium text-primary"
								>
									Used in {media.usage_count}
									{media.usage_count === 1 ? 'post' : 'posts'}
								</span>
							{:else}
								<span
									class="inline-flex items-center rounded-full bg-muted px-2 py-0.5 text-xs text-muted-foreground"
								>
									Unused
								</span>
							{/if}
						</div>
					</div>
				</div>
			{/each}
		</div>

		<!-- Pagination -->
		{#if totalPages > 1}
			<div class="mt-6 flex items-center justify-center gap-4">
				<Button variant="outline" size="sm" onclick={prevPage} disabled={currentPage === 0}>
					<ChevronLeftIcon class="mr-1 h-4 w-4" />
					Previous
				</Button>
				<span class="text-sm text-muted-foreground">
					Page {currentPage + 1} of {totalPages}
				</span>
				<Button
					variant="outline"
					size="sm"
					onclick={nextPage}
					disabled={currentPage >= totalPages - 1}
				>
					Next
					<ChevronRightIcon class="ml-1 h-4 w-4" />
				</Button>
			</div>
		{/if}
	{/if}
</PageContainer>

<!-- Upload Dialog -->
<Dialog.Root bind:open={uploadDialogOpen}>
	<Dialog.Content class="sm:max-w-md">
		<Dialog.Header>
			<Dialog.Title>Upload Media</Dialog.Title>
			<Dialog.Description>Upload images or videos to use in your posts.</Dialog.Description>
		</Dialog.Header>

		<div class="space-y-4 py-4">
			<div class="space-y-2">
				<label class="text-sm font-medium">Single Upload</label>
				<label
					class="flex cursor-pointer flex-col items-center justify-center rounded-lg border-2 border-dashed border-muted-foreground/25 p-6 transition-colors hover:border-primary/50"
					for="file-upload"
				>
					<UploadIcon class="mb-2 h-8 w-8 text-muted-foreground/40" />
					<p class="text-sm font-medium">Click to select a file</p>
					<p class="text-sm text-muted-foreground">Image or video (max 50MB)</p>
				</label>
				<input id="file-upload" type="file" accept="image/*,video/*" class="hidden" />
			</div>

			<div class="relative">
				<div class="absolute inset-0 flex items-center">
					<div class="w-full border-t"></div>
				</div>
				<div class="relative flex justify-center text-xs uppercase">
					<span class="bg-background px-2 text-muted-foreground">Or</span>
				</div>
			</div>

			<div class="space-y-2">
				<label class="text-sm font-medium">Batch Upload (up to 10)</label>
				<label
					class="flex cursor-pointer flex-col items-center justify-center rounded-lg border-2 border-dashed border-muted-foreground/25 p-6 transition-colors hover:border-primary/50"
					for="batch-file-upload"
				>
					<Grid2X2Icon class="mb-2 h-8 w-8 text-muted-foreground/40" />
					<p class="text-sm font-medium">Select multiple files</p>
					<p class="text-sm text-muted-foreground">Images or videos</p>
				</label>
				<input
					id="batch-file-upload"
					type="file"
					accept="image/*,video/*"
					multiple
					class="hidden"
				/>
			</div>

			{#if uploadError}
				<p class="text-sm text-destructive">{uploadError}</p>
			{/if}

			{#if uploadProgress}
				<p class="text-sm text-muted-foreground">{uploadProgress}</p>
			{/if}
		</div>

		<Dialog.Footer>
			<Button variant="outline" onclick={() => (uploadDialogOpen = false)}>Cancel</Button>
			<Button onclick={handleUpload} disabled={uploadLoading}>
				{#if uploadLoading}
					<LoaderIcon class="mr-2 size-4 animate-spin" />
				{/if}
				Upload
			</Button>
		</Dialog.Footer>
	</Dialog.Content>
</Dialog.Root>

<!-- Usage Dialog -->
<Dialog.Root bind:open={usageDialogOpen}>
	<Dialog.Content class="sm:max-w-lg">
		<Dialog.Header>
			<Dialog.Title>Media Usage</Dialog.Title>
			<Dialog.Description>
				{#if selectedMedia}
					{selectedMedia.usage_count}
					{selectedMedia.usage_count === 1 ? 'post' : 'posts'} using this media
				{/if}
			</Dialog.Description>
		</Dialog.Header>

		<div class="max-h-[400px] space-y-2 overflow-y-auto py-4">
			{#if usageLoading}
				<div class="space-y-2 py-4">
					<Skeleton class="h-16 rounded-lg" />
					<Skeleton class="h-16 rounded-lg" />
					<Skeleton class="h-16 rounded-lg" />
				</div>
			{:else if mediaUsage.length === 0}
				<p class="py-8 text-center text-sm text-muted-foreground">
					This media is not used in any posts.
				</p>
			{:else}
				{#each mediaUsage as usage (usage.post_id)}
					<div class="rounded-lg border p-3">
						<p class="line-clamp-2 text-sm">{usage.content}</p>
						<div class="mt-2 flex items-center gap-3 text-sm text-muted-foreground">
							<span class="rounded-full bg-muted px-2 py-0.5 text-xs">{usage.status}</span>
							{#if usage.scheduled}
								<span
									>{new Date(usage.scheduled).toLocaleString('en-US', {
										timeZone: workspaceCtx.settings.timezone || 'UTC'
									})}</span
								>
							{/if}
						</div>
					</div>
				{/each}
			{/if}
		</div>

		<Dialog.Footer>
			<Button variant="outline" onclick={() => (usageDialogOpen = false)}>Close</Button>
		</Dialog.Footer>
	</Dialog.Content>
</Dialog.Root>
