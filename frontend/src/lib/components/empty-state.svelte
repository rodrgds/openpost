<script lang="ts">
	import { Button } from '$lib/components/ui/button';

	interface Props {
		/** Icon component to display */
		icon: ConstructorOfATypedSvelteComponent;
		/** Main title text */
		title: string;
		/** Description text */
		description?: string;
		/** Primary action button text */
		actionLabel?: string;
		/** Primary action button callback */
		onAction?: () => void;
		/** Optional href for the action button (instead of onAction) */
		actionHref?: string;
		/** Variant for the container style */
		variant?: 'default' | 'dashed' | 'muted';
		/** Additional padding size */
		size?: 'sm' | 'md' | 'lg';
	}

	let {
		icon: Icon,
		title,
		description,
		actionLabel,
		onAction,
		actionHref,
		variant = 'default',
		size = 'md'
	}: Props = $props();

	const variantClasses = {
		default: 'border bg-card',
		dashed: 'border border-dashed',
		muted: 'border border-dashed bg-muted/30'
	};

	const sizeClasses = {
		sm: 'py-8',
		md: 'py-12',
		lg: 'py-16'
	};
</script>

<div
	class="flex flex-col items-center justify-center rounded-lg text-center {variantClasses[
		variant
	]} {sizeClasses[size]}"
>
	<Icon class="mb-3 h-10 w-10 text-muted-foreground/40" />
	<p class="mb-1 text-sm font-medium">{title}</p>
	{#if description}
		<p class="mb-4 text-xs text-muted-foreground">{description}</p>
	{/if}
	{#if actionLabel}
		{#if actionHref}
			<Button href={actionHref} size="sm" variant="outline">{actionLabel}</Button>
		{:else if onAction}
			<Button onclick={onAction} size="sm" variant="outline">{actionLabel}</Button>
		{/if}
	{/if}
</div>
