<script lang="ts">
	import { onMount } from 'svelte';
	import { client, type Post } from '$lib/api/client';
	import { Button } from '$lib/components/ui/button';
	import {
		Card,
		CardContent,
		CardHeader,
		CardTitle,
		CardDescription
	} from '$lib/components/ui/card';
	import { goto } from '$app/navigation';
	import { page } from '$app/stores';

	let posts = $state<Post[] | null>(null);
	let isLoading = $state(true);
	let error = $state('');
	let workspaceId = $derived($page.params.id);

	onMount(async () => {
		if (!workspaceId) {
			goto('/');
			return;
		}

		try {
			const { data, error: err } = await client.GET('/posts', {
				params: { query: { workspace_id: workspaceId } }
			});
			if (err || !data) throw new Error('Failed to load posts');
			posts = data;
		} catch (e) {
			console.error('Failed to load posts:', e);
			error = (e as Error).message;
		} finally {
			isLoading = false;
		}
	});

	function formatDate(dateStr: string | null | undefined): string {
		if (!dateStr || dateStr === '0001-01-01T00:00:00Z') return '-';
		return new Date(dateStr).toLocaleString();
	}

	function getStatusColor(status: string): string {
		const colors: Record<string, string> = {
			draft: 'bg-muted text-muted-foreground',
			scheduled: 'bg-blue-100 text-blue-700 dark:bg-blue-950 dark:text-blue-300',
			publishing: 'bg-yellow-100 text-yellow-700 dark:bg-yellow-950 dark:text-yellow-300',
			published: 'bg-green-100 text-green-700 dark:bg-green-950 dark:text-green-300',
			failed: 'bg-destructive/10 text-destructive'
		};
		return colors[status] || 'bg-muted text-muted-foreground';
	}
</script>

<svelte:head>
	<title>Workspace - OpenPost</title>
</svelte:head>

<div class="mx-auto w-full max-w-[1360px] px-4 py-6 lg:px-8">
	<div class="mb-8 flex items-center justify-between">
		<div>
			<h1 class="text-2xl font-bold">Posts</h1>
			<p class="text-sm text-muted-foreground">Workspace: {workspaceId}</p>
		</div>
		<Button href="/workspace/{workspaceId}/compose">New Post</Button>
	</div>

	{#if isLoading}
		<div class="flex justify-center py-12">
			<div class="h-8 w-8 animate-spin rounded-full border-b-2 border-primary"></div>
		</div>
	{:else if error}
		<div class="rounded-md border border-destructive/20 bg-destructive/10 p-4 text-destructive">
			<p>Error: {error}</p>
			<Button
				variant="ghost"
				size="sm"
				onclick={() => {
					error = '';
					isLoading = true;
				}}>Retry</Button
			>
		</div>
	{:else if !posts || posts.length === 0}
		<div class="py-12 text-center">
			<Card class="mx-auto max-w-md">
				<CardHeader>
					<CardTitle>No posts yet</CardTitle>
					<CardDescription>Create your first post to start scheduling.</CardDescription>
				</CardHeader>
				<CardContent>
					<Button href="/workspace/{workspaceId}/compose">Create Post</Button>
				</CardContent>
			</Card>
		</div>
	{:else}
		<Card>
			<div class="overflow-x-auto">
				<table class="min-w-full divide-y divide-border">
					<thead class="bg-muted/50">
						<tr>
							<th
								class="px-6 py-3 text-left text-xs font-medium tracking-wider text-muted-foreground uppercase"
								>Content</th
							>
							<th
								class="px-6 py-3 text-left text-xs font-medium tracking-wider text-muted-foreground uppercase"
								>Status</th
							>
							<th
								class="px-6 py-3 text-left text-xs font-medium tracking-wider text-muted-foreground uppercase"
								>Scheduled</th
							>
							<th
								class="px-6 py-3 text-left text-xs font-medium tracking-wider text-muted-foreground uppercase"
								>Created</th
							>
						</tr>
					</thead>
					<tbody class="divide-y divide-border bg-background">
						{#each posts as post}
							<tr class="hover:bg-muted/50">
								<td class="px-6 py-4">
									<div class="max-w-md truncate text-sm">{post.content}</div>
								</td>
								<td class="px-6 py-4">
									<span
										class="rounded-full px-2 py-1 text-xs font-medium {getStatusColor(post.status)}"
									>
										{post.status}
									</span>
								</td>
								<td class="px-6 py-4 text-sm text-muted-foreground">
									{formatDate(post.scheduled_at)}
								</td>
								<td class="px-6 py-4 text-sm text-muted-foreground">
									{formatDate(post.created_at)}
								</td>
							</tr>
						{/each}
					</tbody>
				</table>
			</div>
		</Card>
	{/if}
</div>
