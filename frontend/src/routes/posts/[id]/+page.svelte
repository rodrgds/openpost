<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { page } from '$app/stores';
	import { client } from '$lib/api/client';
	import ComposeSimple from '$lib/components/compose-simple.svelte';
	import { ui } from '$lib/stores/ui.svelte';
	import { Button } from '$lib/components/ui/button';
	import { Skeleton } from '$lib/components/ui/skeleton/index.js';
	import TrashIcon from 'lucide-svelte/icons/trash-2';

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
	let hasLoaded = $state(false);
	let error = $state('');
	let deleting = $state(false);
	let showDeleteConfirm = $state(false);

	const postId = $derived($page.params.id);

	async function loadPost(id: string) {
		error = '';
		try {
			const { data, error: err } = await (client as any).GET('/posts/{id}', {
				params: { path: { id } }
			});
			if (err) throw new Error((err as any)?.detail || 'Failed to load post');
			post = data;
		} catch (e) {
			error = (e as Error).message;
			if (!hasLoaded) post = null;
		} finally {
			hasLoaded = true;
		}
	}

	onMount(() => {
		if (postId) loadPost(postId);
	});

	$effect(() => {
		if (postId) {
			loadPost(postId);
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

{#if !hasLoaded}
	<div class="mx-auto w-full max-w-2xl space-y-4 p-6">
		<Skeleton class="h-9 w-full rounded-lg" />
		<Skeleton class="h-64 w-full rounded-lg" />
	</div>
{:else if error && !post}
	<div class="mx-auto w-full max-w-6xl px-4 py-6 lg:px-8">
		<div class="rounded-lg border border-destructive/20 bg-destructive/10 p-6 text-center">
			<p class="mb-3 text-destructive">{error}</p>
			<Button variant="outline" onclick={() => goto('/')}>Back</Button>
		</div>
	</div>
{:else if post}
	<div class="flex flex-1 flex-col overflow-hidden">
		<!-- Edit header with delete -->
		<div class="flex flex-wrap items-center justify-between gap-2 border-b px-3 py-2 md:px-4">
			<span class="text-xs text-muted-foreground">
				Editing {post.status} post
			</span>
			{#if post.status === 'draft' || post.status === 'scheduled'}
				{#if showDeleteConfirm}
					<div class="flex items-center gap-2">
						<span class="text-xs text-destructive">Delete?</span>
						<Button
							variant="ghost"
							size="xs"
							onclick={() => (showDeleteConfirm = false)}
							disabled={deleting}
							class="h-6 text-xs"
						>
							Cancel
						</Button>
						<Button
							variant="destructive"
							size="xs"
							onclick={handleDelete}
							disabled={deleting}
							class="h-6 text-xs"
						>
							{deleting ? '...' : 'Confirm'}
						</Button>
					</div>
				{:else}
					<Button
						variant="ghost"
						size="xs"
						class="h-6 gap-1 text-xs text-muted-foreground hover:text-destructive"
						onclick={() => (showDeleteConfirm = true)}
						disabled={deleting}
					>
						<TrashIcon class="h-3 w-3" />
						Delete
					</Button>
				{/if}
			{/if}
		</div>

		{#if error}
			<div
				class="mx-4 mt-3 rounded-md border border-destructive/20 bg-destructive/10 px-3 py-2 text-sm text-destructive"
			>
				{error}
			</div>
		{/if}

		<ComposeSimple initialPost={post} onSuccess={handleSuccess} onCancel={handleCancel} />
	</div>
{/if}
