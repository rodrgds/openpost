<script lang="ts">
	import { onMount } from 'svelte';
	import { client, type Workspace } from '$lib/api/client';
	import { Button } from '$lib/components/ui/button';
	import { Card, CardContent } from '$lib/components/ui/card';
	import { Input } from '$lib/components/ui/input';
	import * as Select from '$lib/components/ui/select';
	import * as Dialog from '$lib/components/ui/dialog';
	import LoaderIcon from 'lucide-svelte/icons/loader-2';
	import ImageIcon from 'lucide-svelte/icons/image';
	import VideoIcon from 'lucide-svelte/icons/video';
	import FileIcon from 'lucide-svelte/icons/file';
	import HeartIcon from 'lucide-svelte/icons/heart';
	import TrashIcon from 'lucide-svelte/icons/trash-2';
	import UploadIcon from 'lucide-svelte/icons/upload';
	import XIcon from 'lucide-svelte/icons/x';
	import ExternalLinkIcon from 'lucide-svelte/icons/external-link';

	interface MediaItem {
		id: string;
		workspace_id: string;
		mime_type: string;
		size: number;
		alt_text: string;
		is_favorite: boolean;
		created_at: string;
		url: string;
		usage_count: number;
		processing_status: string;
	}

	interface MediaUsage {
		post_id: string;
		content: string;
		status: string;
		scheduled: string;
	}

	let workspaces = $state<Workspace[] | null>(null);
	let selectedWorkspaceId = $state('');
	let loading = $state(true);
	let error = $state('');

	let mediaItems = $state<MediaItem[]>([]);
	let mediaLoading = $state(false);
	let totalCount = $state(0);

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

	let selectedWorkspaceName = $derived(
		workspaces?.find((w) => w.id === selectedWorkspaceId)?.name || 'Select workspace'
	);

	async function loadWorkspaces() {
		try {
			const { data } = await client.GET('/workspaces', {});
			workspaces = data ?? [];
			if (workspaces.length > 0 && !selectedWorkspaceId) {
				selectedWorkspaceId = workspaces[0].id;
			}
		} catch (e) {
			console.error('Failed to load workspaces:', e);
		}
	}

	async function loadMedia() {
		if (!selectedWorkspaceId) return;
		mediaLoading = true;
		error = '';
		try {
			const { data, error: err } = await (client as any).GET('/media', {
				params: {
					query: {
						workspace_id: selectedWorkspaceId,
						filter: filter,
						sort: sort
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
			error = (e as Error).message;
		}
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
		} catch (e) {
			error = (e as Error).message;
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
			error = (e as Error).message;
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
			const response = await fetch('/api/v1/media/upload', {
				method: 'POST',
				body: formData
			});

			if (!response.ok) {
				const errData = await response.json();
				throw new Error(errData.error || 'Upload failed');
			}

			uploadDialogOpen = false;
			fileInput.value = '';
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
			year: 'numeric'
		});
	}

	function getMediaIcon(mimeType: string) {
		if (mimeType.startsWith('video/')) return VideoIcon;
		if (mimeType.startsWith('image/')) return ImageIcon;
		return FileIcon;
	}

	function isImage(mimeType: string): boolean {
		return mimeType.startsWith('image/');
	}

	onMount(() => {
		loadWorkspaces();
	});

	$effect(() => {
		if (selectedWorkspaceId) {
			loadMedia();
		}
	});
</script>

<div class="flex flex-1 flex-col gap-4 p-4">
	<div class="flex items-center justify-between">
		<div>
			<h1 class="text-2xl font-bold tracking-tight">Media Library</h1>
			<p class="text-sm text-muted-foreground">Manage your media attachments</p>
		</div>
		<div class="flex items-center gap-2">
			<Button variant="outline" onclick={() => (uploadDialogOpen = true)}>
				<UploadIcon class="mr-2 size-4" />
				Upload
			</Button>
		</div>
	</div>

	{#if error}
		<div class="flex items-center gap-2 rounded-md bg-destructive/10 p-3 text-sm text-destructive">
			<XIcon class="size-4" />
			{error}
			<button class="ml-auto" onclick={() => (error = '')}>
				<XIcon class="size-4" />
			</button>
		</div>
	{/if}

	<div class="flex flex-wrap items-center gap-4">
		<div class="flex items-center gap-2">
			<label class="text-sm font-medium">Workspace:</label>
			<Select.Root type="single" bind:value={selectedWorkspaceId}>
				<Select.Trigger class="w-[200px]">
					{workspaces?.find((w) => w.id === selectedWorkspaceId)?.name || 'Select workspace'}
				</Select.Trigger>
				<Select.Content>
					{#each workspaces ?? [] as workspace}
						<Select.Item value={workspace.id}>{workspace.name}</Select.Item>
					{/each}
				</Select.Content>
			</Select.Root>
		</div>

		<div class="flex items-center gap-2">
			<label class="text-sm font-medium">Filter:</label>
			<Select.Root type="single" bind:value={filter}>
				<Select.Trigger class="w-[140px]">
					{filter === 'all'
						? 'All'
						: filter === 'used'
							? 'Used'
							: filter === 'unused'
								? 'Unused'
								: 'Favorites'}
				</Select.Trigger>
				<Select.Content>
					<Select.Item value="all">All</Select.Item>
					<Select.Item value="used">Used</Select.Item>
					<Select.Item value="unused">Unused</Select.Item>
					<Select.Item value="favorites">Favorites</Select.Item>
				</Select.Content>
			</Select.Root>
		</div>

		<div class="flex items-center gap-2">
			<label class="text-sm font-medium">Sort:</label>
			<Select.Root type="single" bind:value={sort}>
				<Select.Trigger class="w-[140px]">
					{sort === 'newest' ? 'Newest' : sort === 'oldest' ? 'Oldest' : 'Size'}
				</Select.Trigger>
				<Select.Content>
					<Select.Item value="newest">Newest</Select.Item>
					<Select.Item value="oldest">Oldest</Select.Item>
					<Select.Item value="size">Size</Select.Item>
				</Select.Content>
			</Select.Root>
		</div>

		{#if totalCount > 0}
			<span class="text-sm text-muted-foreground">{totalCount} files</span>
		{/if}
	</div>

	{#if mediaLoading}
		<div class="flex items-center justify-center py-12">
			<LoaderIcon class="size-8 animate-spin text-muted-foreground" />
		</div>
	{:else if mediaItems.length === 0}
		<div class="flex flex-col items-center justify-center py-12 text-center">
			<ImageIcon class="mb-4 size-12 text-muted-foreground/50" />
			<p class="text-lg font-medium text-muted-foreground">No media found</p>
			<p class="text-sm text-muted-foreground">
				{#if filter !== 'all' || sort !== 'newest'}
					Try changing your filters or
				{/if}
				upload some files to get started
			</p>
		</div>
	{:else}
		<div class="grid grid-cols-1 gap-4 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5">
			{#each mediaItems as media (media.id)}
				{@const MediaIcon = getMediaIcon(media.mime_type)}
				<Card class="group relative overflow-hidden">
					<div class="relative flex aspect-square items-center justify-center bg-muted/50">
						{#if isImage(media.mime_type)}
							<img src={media.url} alt={media.alt_text || 'Media'} class="size-full object-cover" />
						{:else}
							<MediaIcon class="size-12 text-muted-foreground" />
						{/if}

						<div
							class="absolute inset-0 flex items-center justify-center gap-2 bg-black/50 opacity-0 transition-opacity group-hover:opacity-100"
						>
							<button
								class="rounded-full bg-white/20 p-2 transition-colors hover:bg-white/30"
								onclick={() => showUsage(media)}
								title="View usage"
							>
								<ExternalLinkIcon class="size-4 text-white" />
							</button>
							<button
								class="rounded-full bg-white/20 p-2 transition-colors hover:bg-white/30"
								onclick={() => toggleFavorite(media.id)}
								title={media.is_favorite ? 'Remove from favorites' : 'Add to favorites'}
							>
								<HeartIcon class="size-4" fill={media.is_favorite ? 'currentColor' : 'none'} />
							</button>
							{#if media.usage_count === 0}
								<button
									class="rounded-full bg-white/20 p-2 transition-colors hover:bg-destructive/80"
									onclick={() => deleteMedia(media.id)}
									title="Delete"
								>
									<TrashIcon class="size-4 text-white" />
								</button>
							{/if}
						</div>

						{#if media.is_favorite}
							<div class="absolute top-2 right-2">
								<HeartIcon class="size-5 fill-red-500 text-red-500" />
							</div>
						{/if}
					</div>

					<CardContent class="p-3">
						<div class="flex items-start justify-between gap-2">
							<div class="min-w-0 flex-1">
								<p class="truncate text-sm font-medium">{media.mime_type}</p>
								<p class="text-xs text-muted-foreground">
									{formatSize(media.size)} · {formatDate(media.created_at)}
								</p>
							</div>
						</div>
						<div class="mt-2 flex items-center gap-2">
							{#if media.usage_count > 0}
								<span
									class="inline-flex items-center gap-1 rounded-full bg-primary/10 px-2 py-0.5 text-xs text-primary"
								>
									Used in {media.usage_count}
									{media.usage_count === 1 ? 'post' : 'posts'}
								</span>
							{:else}
								<span
									class="inline-flex items-center gap-1 rounded-full bg-muted px-2 py-0.5 text-xs text-muted-foreground"
								>
									Unused
								</span>
							{/if}
						</div>
					</CardContent>
				</Card>
			{/each}
		</div>
	{/if}
</div>

<Dialog.Root bind:open={uploadDialogOpen}>
	<Dialog.Content class="sm:max-w-md">
		<Dialog.Header>
			<Dialog.Title>Upload Media</Dialog.Title>
			<Dialog.Description>Upload an image or video to use in your posts.</Dialog.Description>
		</Dialog.Header>

		<div class="space-y-4 py-4">
			<div class="space-y-2">
				<label for="file-upload" class="text-sm font-medium">File</label>
				<Input id="file-upload" type="file" accept="image/*,video/*" />
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

		<div class="max-h-[400px] space-y-3 overflow-y-auto py-4">
			{#if usageLoading}
				<div class="flex items-center justify-center py-8">
					<LoaderIcon class="size-6 animate-spin text-muted-foreground" />
				</div>
			{:else if mediaUsage.length === 0}
				<p class="py-8 text-center text-sm text-muted-foreground">
					This media is not used in any posts
				</p>
			{:else}
				{#each mediaUsage as usage}
					<div class="rounded-md border p-3">
						<p class="line-clamp-2 text-sm">{usage.content}</p>
						<div class="mt-2 flex items-center gap-3 text-xs text-muted-foreground">
							<span class="rounded-full bg-muted px-2 py-0.5">{usage.status}</span>
							{#if usage.scheduled}
								<span>Scheduled: {new Date(usage.scheduled).toLocaleString()}</span>
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
