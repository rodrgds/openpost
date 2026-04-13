<script lang="ts">
	import { goto } from '$app/navigation';
	import ComposePost from '$lib/components/compose-post.svelte';
	import { ui } from '$lib/stores/ui.svelte';

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

<div class="mx-auto w-full max-w-4xl px-4 py-6 lg:px-8">
	<div class="mb-6">
		<h1 class="text-2xl font-bold">Create Post</h1>
		<p class="text-sm text-muted-foreground">Write your post and choose when to publish</p>
	</div>

	<div class="rounded-lg border bg-card">
		<ComposePost isPage={true} onSuccess={handleSuccess} onCancel={handleCancel} />
	</div>
</div>
