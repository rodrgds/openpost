<script lang="ts">
	import { mode } from 'mode-watcher';

	interface Props {
		class?: string;
		width?: number;
		height?: number;
		showText?: boolean;
	}

	let { class: className = '', width = 100, height = 28, showText = false }: Props = $props();

	let isDark = $derived(mode.current === 'dark');
</script>

{#if showText}
	<div class="flex items-center gap-2 {className}">
		<img
			src={isDark ? '/logo-white.png' : '/logo-black.png'}
			alt="OpenPost"
			{width}
			{height}
			class="h-auto max-w-full"
		/>
		<span class="font-bold text-sidebar-foreground whitespace-nowrap">OpenPost</span>
	</div>
{:else}
	<img
		src={isDark ? '/logo-white.png' : '/logo-black.png'}
		alt="OpenPost"
		{width}
		{height}
		class="h-auto max-w-full {className}"
	/>
{/if}
