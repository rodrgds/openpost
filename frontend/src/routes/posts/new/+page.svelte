<script lang="ts">
	import { goto } from '$app/navigation';
	import ComposePost from '$lib/components/compose-post.svelte';
	import PageContainer from '$lib/components/page-container.svelte';
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

<PageContainer
	title="Create Post"
	description="Write your post and choose when to publish"
	icon={PlusIcon}
>
	<div class="rounded-lg border bg-card p-6 pb-0">
		<ComposePost isPage={true} onSuccess={handleSuccess} onCancel={handleCancel} />
	</div>
</PageContainer>
