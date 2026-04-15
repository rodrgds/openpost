<script lang="ts">
	import type { Snippet } from 'svelte';

	interface Props {
		/** Page title displayed in the header */
		title: string;
		/** Optional icon component to display before title */
		icon?: any;
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
	<div class="flex flex-1 flex-col items-center justify-center gap-4 py-16">
		<div class="h-8 w-8 animate-spin rounded-full border-b-2 border-primary"></div>
		{#if loadingMessage}
			<p class="text-sm text-muted-foreground">{loadingMessage}</p>
		{/if}
	</div>
{:else}
	<div class="mx-auto w-full max-w-6xl px-4 py-6 lg:px-8">
		<!-- Page Header -->
		<div class="mb-6 flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
			<div>
				<h1 class="flex items-center gap-2 text-2xl font-bold tracking-tight">
					{#if Icon}
						<Icon class="h-6 w-6 text-primary" />
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
