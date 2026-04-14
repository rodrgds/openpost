<script lang="ts">
	import { goto } from '$app/navigation';
	import ComposePost from '$lib/components/compose-post.svelte';
	import { ui } from '$lib/stores/ui.svelte';
	import PlusIcon from 'lucide-svelte/icons/plus';

	async function handleSuccess() {
		ui.triggerRefresh();
		goto('/');
	}

	function handleCancel() {
		goto('/');
	}
</script>

<svelte:head>
	<title>New Post - OpenPost</title>
</svelte:head>

<div class="mx-auto w-full max-w-6xl px-6 py-8 lg:px-12">
	<div class="mb-6">
		<div class="flex items-center gap-2">
			<PlusIcon class="h-5 w-5 text-primary" />
			<h1 class="text-2xl font-bold tracking-tight">Create Post</h1>
		</div>
		<p class="mt-1 text-sm text-muted-foreground">Write your post and choose when to publish</p>
	</div>

	<div class="rounded-lg border bg-card p-6 pb-0">
		<ComposePost isPage={true} onSuccess={handleSuccess} onCancel={handleCancel} />
	</div>
</div>
