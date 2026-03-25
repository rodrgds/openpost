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
		<object
			data="/logo.svg"
			type="image/svg+xml"
			aria-label="OpenPost"
			{width}
			{height}
			class="h-auto max-w-full"
		></object>
		<span class="font-bold whitespace-nowrap text-sidebar-foreground">OpenPost</span>
	</div>
{:else}
	<object
		data="/logo.svg"
		type="image/svg+xml"
		aria-label="OpenPost"
		{width}
		{height}
		class="h-auto max-w-full {className}"
	></object>
{/if}
