<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { page } from '$app/stores';
	import { client } from '$lib/api/client';
	import ComposePost from '$lib/components/compose-post.svelte';
	import { ui } from '$lib/stores/ui.svelte';
	import { Button } from '$lib/components/ui/button';
	import LoaderIcon from 'lucide-svelte/icons/loader-2';
	import TrashIcon from 'lucide-svelte/icons/trash-2';
	import PencilIcon from 'lucide-svelte/icons/pencil';

	interface PostMedia {
		media_id: string;
		display_order: number;
		file_path: string;
		mime_type: string;
	}

	interface PostDestination {
		social_account_id: string;
		platform: string;
		status: string;
	}

	interface PostDetail {
		id: string;
		workspace_id: string;
		created_by: string;
		content: string;
		status: string;
		scheduled_at: string;
		created_at: string;
		media: PostMedia[];
		destinations: PostDestination[];
	}

	let post = $state<PostDetail | null>(null);
	let loading = $state(true);
	let error = $state('');
	let deleting = $state(false);
	let showDeleteConfirm = $state(false);

	const postId = $derived($page.params.id);

	onMount(async () => {
		try {
			const { data, error: err } = await (client as any).GET('/posts/{id}', {
				params: { path: { id: postId } }
			});
			if (err) throw new Error((err as any)?.detail || 'Failed to load post');
			post = data;
		} catch (e) {
			error = (e as Error).message;
		} finally {
			loading = false;
		}
	});

	async function handleDelete() {
		if (!post) return;
		deleting = true;
		try {
			const { error: err } = await (client as any).DELETE('/posts/{id}', {
				params: { path: { id: post.id } }
			});
			if (err) throw new Error((err as any)?.detail || 'Failed to delete post');
			ui.triggerRefresh();
			goto('/');
		} catch (e) {
			error = (e as Error).message;
			deleting = false;
			showDeleteConfirm = false;
		}
	}

	async function handleSuccess() {
		ui.triggerRefresh();
		goto('/');
	}

	function handleCancel() {
		goto('/');
	}
</script>

<svelte:head>
	<title>{post ? 'Edit Post' : 'Loading...'} - OpenPost</title>
</svelte:head>

{#if loading}
	<div class="flex flex-1 items-center justify-center">
		<LoaderIcon class="h-8 w-8 animate-spin text-primary" />
	</div>
{:else if error && !post}
	<div class="mx-auto w-full max-w-6xl px-6 py-8 lg:px-12">
		<div class="rounded-lg border border-destructive/20 bg-destructive/10 p-6 text-center">
			<p class="mb-3 text-destructive">{error}</p>
			<Button variant="outline" onclick={() => goto('/')}>Back to Dashboard</Button>
		</div>
	</div>
{:else if post}
	<div class="mx-auto w-full max-w-6xl px-6 py-8 lg:px-12">
		<div class="mb-6 flex items-start justify-between">
			<div>
				<div class="flex items-center gap-2">
					<PencilIcon class="h-5 w-5 text-primary" />
					<h1 class="text-2xl font-bold tracking-tight">Edit Post</h1>
				</div>
				<p class="mt-1 text-sm text-muted-foreground">
					<span class="capitalize">{post.status}</span>
					{#if post.scheduled_at && post.scheduled_at !== '0001-01-01T00:00:00Z'}
						<span class="mx-1.5">·</span>
						Scheduled for {new Date(post.scheduled_at).toLocaleString()}
					{/if}
				</p>
			</div>
			{#if post.status === 'draft' || post.status === 'scheduled'}
				{#if showDeleteConfirm}
					<div class="flex items-center gap-2">
						<span class="text-sm text-destructive">Delete this post?</span>
						<Button
							variant="outline"
							size="sm"
							onclick={() => (showDeleteConfirm = false)}
							disabled={deleting}
						>
							Cancel
						</Button>
						<Button variant="destructive" size="sm" onclick={handleDelete} disabled={deleting}>
							{deleting ? 'Deleting...' : 'Confirm'}
						</Button>
					</div>
				{:else}
					<Button
						variant="ghost"
						size="sm"
						class="gap-1.5 text-muted-foreground hover:text-destructive"
						onclick={() => (showDeleteConfirm = true)}
						disabled={deleting}
					>
						<TrashIcon class="h-4 w-4" />
						Delete
					</Button>
				{/if}
			{/if}
		</div>

		{#if error}
			<div
				class="mb-4 rounded-lg border border-destructive/20 bg-destructive/10 p-3 text-sm text-destructive"
			>
				{error}
			</div>
		{/if}

		<div class="rounded-lg border bg-card p-6 pb-0">
			<ComposePost
				isPage={true}
				initialPost={post}
				onSuccess={handleSuccess}
				onCancel={handleCancel}
			/>
		</div>
	</div>
{/if}
