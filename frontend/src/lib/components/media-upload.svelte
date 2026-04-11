<script lang="ts">
	import { getToken } from '$lib/api/client';
	import { getApiBase } from '$lib/stores/instance.svelte';
	import { Button } from '$lib/components/ui/button';
	import { Textarea } from '$lib/components/ui/textarea';
	import {
		Tooltip,
		TooltipContent,
		TooltipProvider,
		TooltipTrigger
	} from '$lib/components/ui/tooltip';
	import PlusIcon from 'lucide-svelte/icons/plus';
	import XIcon from 'lucide-svelte/icons/x';
	import ImageIcon from 'lucide-svelte/icons/image';
	import VideoIcon from 'lucide-svelte/icons/video';
	import LoaderIcon from 'lucide-svelte/icons/loader-2';
	import PencilIcon from 'lucide-svelte/icons/pencil';
	import CheckIcon from 'lucide-svelte/icons/check';

	interface Props {
		workspaceId: string;
		mediaIds?: string[];
		disabled?: boolean;
	}

	let { workspaceId, mediaIds = $bindable([]), disabled = false }: Props = $props();

	interface MediaItem {
		clientId: string;
		id: string;
		file: File;
		status: 'uploading' | 'ready' | 'error';
		url: string;
		altText: string;
		mimeType: string;
	}

	let items = $state<MediaItem[]>([]);
	let _isDragging = $state(false);
	let editingAltId = $state<string | null>(null);
	let editingAltText = $state('');

	let inputElement: HTMLInputElement | null = null;
	let uploadSeq = 0;

	function resetFileInput() {
		if (inputElement) inputElement.value = '';
	}

	async function handleFiles(files: FileList | File[]) {
		for (const file of Array.from(files)) {
			if (!file.type.startsWith('image/') && !file.type.startsWith('video/')) continue;

			const item: MediaItem = {
				clientId: `media-${uploadSeq++}`,
				id: '',
				file,
				status: 'uploading',
				url: URL.createObjectURL(file),
				altText: '',
				mimeType: file.type
			};
			items = [...items, item];

			try {
				const formData = new FormData();
				formData.append('file', file);
				formData.append('workspace_id', workspaceId);
				if (item.altText) {
					formData.append('alt_text', item.altText);
				}

				const token = getToken();
				const resp = await fetch(`${getApiBase()}/media/upload`, {
					method: 'POST',
					headers: token ? { Authorization: `Bearer ${token}` } : {},
					body: formData
				});

				if (!resp.ok) {
					throw new Error(`Upload failed (${resp.status})`);
				}

				const data = await resp.json();
				const idx = items.findIndex((m) => m.clientId === item.clientId);
				if (idx !== -1) {
					items = items.map((m, i) => (i === idx ? { ...m, id: data.id, status: 'ready' } : m));
				}
			} catch {
				const idx = items.findIndex((m) => m.clientId === item.clientId);
				if (idx !== -1) {
					items = items.map((m, i) => (i === idx ? { ...m, status: 'error' } : m));
				}
			}
		}
		resetFileInput();
	}

	function removeItem(index: number) {
		const item = items[index];
		if (item.url.startsWith('blob:')) URL.revokeObjectURL(item.url);
		items = items.filter((_, i) => i !== index);
		syncMediaIds();
	}

	function syncMediaIds() {
		mediaIds = items.filter((i) => i.status === 'ready').map((i) => i.id);
	}

	function startEditAlt(item: MediaItem) {
		editingAltId = item.id;
		editingAltText = item.altText;
	}

	function saveAlt(item: MediaItem) {
		item.altText = editingAltText;
		editingAltId = null;
		editingAltText = '';
	}

	$effect(() => {
		mediaIds = items.filter((i) => i.status === 'ready').map((i) => i.id);
	});

	const readyCount = $derived(items.filter((i) => i.status === 'ready').length);
</script>

{#if items.length > 0}
	<div class="space-y-2">
		<div class="flex flex-wrap gap-2">
			{#each items as item, i (item.url + i)}
				<div class="group relative h-20 w-20 overflow-hidden rounded-md border">
					{#if item.mimeType.startsWith('image/')}
						<img src={item.url} alt={item.altText} class="h-full w-full object-cover" />
					{:else}
						<div class="flex h-full w-full items-center justify-center bg-muted">
							<VideoIcon class="h-6 w-6 text-muted-foreground" />
						</div>
					{/if}

					{#if item.status === 'uploading'}
						<div class="absolute inset-0 flex items-center justify-center bg-background/80">
							<LoaderIcon class="h-5 w-5 animate-spin text-primary" />
						</div>
					{/if}

					{#if item.status === 'error'}
						<div class="absolute inset-0 flex items-center justify-center bg-destructive/20">
							<span class="text-xs text-destructive">Failed</span>
						</div>
					{/if}

					{#if item.status === 'ready'}
						{#if editingAltId === item.id}
							<div class="absolute inset-0 flex flex-col bg-background/95 p-1">
								<Textarea
									bind:value={editingAltText}
									class="h-full min-h-0 resize-none text-xs"
									placeholder="Alt text"
									onkeydown={(e) => {
										if (e.key === 'Enter' && !e.shiftKey) {
											e.preventDefault();
											saveAlt(item);
										}
										if (e.key === 'Escape') {
											editingAltId = null;
										}
									}}
								/>
								<div class="flex justify-end gap-1">
									<Button
										variant="ghost"
										size="icon-xs"
										class="h-5 w-5"
										onclick={() => saveAlt(item)}
									>
										<CheckIcon class="h-3 w-3" />
									</Button>
								</div>
							</div>
						{:else}
							<div
								class="absolute inset-0 flex items-start justify-between bg-gradient-to-b from-black/40 to-transparent opacity-0 transition-opacity group-hover:opacity-100"
							>
								<TooltipProvider>
									<Tooltip>
										<TooltipTrigger>
											<button
												type="button"
												class="rounded-full bg-black/40 p-1 text-white hover:bg-black/60"
												onclick={() => startEditAlt(item)}
											>
												<PencilIcon class="h-3 w-3" />
											</button>
										</TooltipTrigger>
										<TooltipContent>
											<p class="text-xs">Add alt text</p>
										</TooltipContent>
									</Tooltip>
								</TooltipProvider>
								<button
									type="button"
									class="rounded-full bg-black/40 p-1 text-white hover:bg-black/60"
									onclick={() => removeItem(i)}
								>
									<XIcon class="h-3 w-3" />
								</button>
							</div>
						{/if}
					{:else}
						<button
							type="button"
							class="absolute top-1 right-1 rounded-full bg-black/40 p-1 text-white hover:bg-black/60"
							onclick={() => removeItem(i)}
						>
							<XIcon class="h-3 w-3" />
						</button>
					{/if}
				</div>
			{/each}
		</div>
		<div class="text-xs text-muted-foreground">
			{readyCount} media attached
			{#if readyCount >= 4}
				<span class="text-amber-500"> (max 4 per post on most platforms)</span>
			{/if}
		</div>
	</div>
{/if}

<button
	type="button"
	class="relative flex h-24 w-full cursor-pointer items-center justify-center rounded-lg border-2 border-dashed border-muted-foreground/25 transition-colors hover:border-muted-foreground/50 hover:bg-muted/30 {disabled
		? 'pointer-events-none opacity-50'
		: ''}"
	ondragover={(e) => {
		e.preventDefault();
		_isDragging = true;
	}}
	ondragleave={() => (_isDragging = false)}
	ondrop={(e) => {
		e.preventDefault();
		_isDragging = false;
		if (e.dataTransfer?.files) handleFiles(e.dataTransfer.files);
	}}
	onclick={() => {
		if (!disabled && items.length < 4) inputElement?.click();
	}}
>
	<input
		type="file"
		accept="image/*,video/*"
		multiple
		class="hidden"
		bind:this={inputElement}
		onchange={(e) => {
			const target = e.target as HTMLInputElement;
			if (target.files) handleFiles(target.files);
		}}
	/>
	<div class="flex flex-col items-center gap-1 text-muted-foreground">
		{#if items.length === 0}
			<ImageIcon class="h-6 w-6" />
			<span class="text-xs">Drop images or videos here, or click to upload</span>
		{:else if items.length < 4}
			<PlusIcon class="h-5 w-5" />
			<span class="text-xs">Add more media</span>
		{:else}
			<span class="text-xs text-amber-500">Maximum 4 media reached</span>
		{/if}
	</div>
</button>
