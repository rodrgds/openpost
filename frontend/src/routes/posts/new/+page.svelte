<script lang="ts">
	import { goto } from '$app/navigation';
	import { onMount } from 'svelte';
	import { Skeleton } from '$lib/components/ui/skeleton/index.js';

	onMount(() => {
		goto('/');
	});
</script>

<div class="mx-auto w-full max-w-2xl space-y-4 p-6">
	<Skeleton class="h-9 w-full rounded-lg" />
	<Skeleton class="h-64 w-full rounded-lg" />
</div>
