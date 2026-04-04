<script lang="ts">
	import { Textarea } from '$lib/components/ui/textarea';
	import MediaUpload from './media-upload.svelte';
	import XIcon from 'lucide-svelte/icons/x';

	interface Props {
		index: number;
		total: number;
		content?: string;
		mediaIds?: string[];
		workspaceId: string;
		disabled?: boolean;
		onRemove?: () => void;
	}

	let {
		index,
		total,
		content = $bindable(''),
		mediaIds = $bindable([]),
		workspaceId,
		disabled = false,
		onRemove
	}: Props = $props();

	const isFirst = $derived(index === 0);
	const isLast = $derived(index === total - 1);
</script>

<div class="relative">
	{#if !isFirst}
		<div
			class="absolute -top-3 left-4 flex h-6 w-0.5 items-center justify-center"
			style="background: linear-gradient(to bottom, transparent, hsl(var(--border)) 50%, transparent);"
		></div>
	{/if}

	<div class="rounded-lg border bg-card p-4 {isFirst ? '' : 'mt-3'}">
		<div class="mb-3 flex items-center justify-between">
			<div class="flex items-center gap-2">
				<div
					class="flex h-6 w-6 items-center justify-center rounded-full bg-primary text-xs font-bold text-primary-foreground"
				>
					{index + 1}
				</div>
				<span class="text-sm font-medium">Post {index + 1} of {total}</span>
			</div>
			{#if !isFirst && onRemove}
				<button
					type="button"
					class="rounded-md p-1 text-muted-foreground hover:bg-muted hover:text-foreground"
					onclick={onRemove}
				>
					<XIcon class="h-4 w-4" />
				</button>
			{/if}
		</div>

		<div class="space-y-3">
			<div class="space-y-2">
				<Textarea
					bind:value={content}
					rows={4}
					placeholder="What's in this post?"
					{disabled}
					class="resize-none"
				/>
				<div class="flex justify-end">
					<span class="text-xs text-muted-foreground">{content.length} characters</span>
				</div>
			</div>

			<MediaUpload {workspaceId} bind:mediaIds {disabled} />
		</div>
	</div>

	{#if !isLast}
		<div class="ml-4 h-3 w-0.5 bg-border"></div>
	{/if}
</div>
