<script lang="ts">
	import type { Snippet } from 'svelte';
	import { Skeleton } from '$lib/components/ui/skeleton/index.js';

	interface Props {
		/** Page title displayed in the header */
		title: string;
		/** Optional icon component to display before title */
		icon?: ConstructorOfATypedSvelteComponent;
		/** Optional description text below title - can be HTML string */
		description?: string;
		/** Optional header actions (buttons, etc.) */
		actions?: Snippet;
		/** Whether to show loading state */
		loading?: boolean;
		/** Optional loading message */
		loadingMessage?: string;
		/** Page content */
		children: Snippet;
	}

	let {
		title,
		icon: Icon,
		description,
		actions,
		loading = false,
		loadingMessage = 'Loading...',
		children
	}: Props = $props();
</script>

{#if loading}
	<div class="mx-auto flex w-full max-w-6xl flex-1 flex-col gap-4 px-4 py-6 lg:px-8">
		<Skeleton class="h-7 w-48 rounded" />
		<Skeleton class="h-4 w-64 rounded" />
		<div class="mt-4 flex flex-col gap-4">
			<Skeleton class="h-32 rounded-lg" />
			<Skeleton class="h-24 rounded-lg" />
			<Skeleton class="h-24 rounded-lg" />
		</div>
	</div>
{:else}
	<div class="mx-auto w-full max-w-6xl px-4 py-6 lg:px-8">
		<!-- Page Header -->
		<div class="mb-6 flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
			<div>
			<h1 class="flex items-center gap-2.5 text-xl font-semibold tracking-tight">
				{#if Icon}
					<Icon class="h-5 w-5 text-primary" />
				{/if}
				{title}
			</h1>
			{#if description}
				<p class="mt-1 text-sm text-muted-foreground">{description}</p>
			{/if}
			</div>
			{#if actions}
				<div class="flex shrink-0 items-center gap-2">
					{@render actions()}
				</div>
			{/if}
		</div>

		<!-- Page Content -->
		{@render children()}
	</div>
{/if}
