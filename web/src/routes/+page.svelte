<script lang="ts">
	import { onMount } from 'svelte';
	import { auth } from '$lib/stores/auth';
	import { client, type Workspace, type Post } from '$lib/api/client';
	import { Button } from '$lib/components/ui/button';
	import {
		Card,
		CardContent,
		CardHeader,
		CardTitle,
		CardDescription
	} from '$lib/components/ui/card';
	import { Input } from '$lib/components/ui/input';
	import { Label } from '$lib/components/ui/label';
	import { ui } from '$lib/stores/ui.svelte';

	let workspaces = $state<Workspace[] | null>(null);
	let posts = $state<Post[]>([]);
	let loading = $state(true);
	let error = $state('');
	let showCreateWorkspace = $state(false);
	let newWorkspaceName = $state('');

	let authReady = $state(false);
	let isAuthenticated = $state(false);

	onMount(() => {
		const unsubscribe = auth.subscribe((state) => {
			isAuthenticated = state.isAuthenticated;

			if (!state.isLoading && !authReady) {
				authReady = true;
				if (state.isAuthenticated) {
					loadDashboard();
				} else {
					loading = false;
				}
			}
		});

		return unsubscribe;
	});

	$effect(() => {
		if (ui.refreshCounter > 0 && isAuthenticated) {
			loadDashboard();
		}
	});

	async function loadDashboard() {
		loading = true;
		try {
			const { data: wsData, error: wsErr } = await client.GET('/workspaces');
			if (wsErr || !wsData) throw new Error('Failed to load workspaces');
			workspaces = wsData;

			const allPosts: Post[] = [];
			for (const ws of workspaces) {
				const { data: pData } = await client.GET('/posts', {
					params: { query: { workspace_id: ws.id } }
				});
				if (pData) {
					allPosts.push(...pData);
				}
			}
			posts = allPosts.sort(
				(a, b) => new Date(b.created_at).getTime() - new Date(a.created_at).getTime()
			);
		} catch (e) {
			console.error('Failed to load dashboard:', e);
			error = (e as Error).message;
		} finally {
			loading = false;
		}
	}

	async function createWorkspace(e: Event) {
		e.preventDefault();
		if (!newWorkspaceName.trim()) return;

		try {
			const { error: err } = await client.POST('/workspaces', {
				body: { name: newWorkspaceName }
			});
			if (err) throw new Error(err.detail || 'Failed to create workspace');
			newWorkspaceName = '';
			showCreateWorkspace = false;
			await loadDashboard();
		} catch (e) {
			console.error('Failed to create workspace:', e);
			error = (e as Error).message;
		}
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

	function getWorkspaceName(workspaceId: string): string {
		return workspaces?.find((w) => w.id === workspaceId)?.name || 'Unknown Workspace';
	}
</script>

<svelte:head>
	<title>Dashboard - OpenPost</title>
</svelte:head>

<div class="mx-auto w-full max-w-[1360px] px-4 py-6 lg:px-8">
	<div class="mb-8 flex items-center justify-between">
		<div>
			<h1 class="text-2xl font-bold">Dashboard</h1>
			<p class="text-sm text-muted-foreground">All your posts across all workspaces</p>
		</div>
		<div class="flex gap-2">
			<Button variant="outline" onclick={() => (showCreateWorkspace = true)}
				>Manage Workspaces</Button
			>
		</div>
	</div>

	{#if loading}
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
					loadDashboard();
				}}
				class="mt-2"
			>
				Retry
			</Button>
		</div>
	{:else if !isAuthenticated}
		<div class="py-12 text-center">
			<p class="text-muted-foreground">Please log in to view your dashboard.</p>
			<a href="/login" class="mt-2 inline-block font-medium text-primary hover:underline">
				Go to Login
			</a>
		</div>
	{:else if posts.length === 0}
		<div class="py-12 text-center">
			<Card class="mx-auto max-w-md border-dashed">
				<CardHeader>
					<CardTitle>No posts found</CardTitle>
					<CardDescription
						>You haven't created any posts yet. Create your first post to see it here!</CardDescription
					>
				</CardHeader>
				<CardContent>
					<Button onclick={() => ui.openCompose()}>Create your first post</Button>
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
								>Workspace</th
							>
							<th
								class="px-6 py-3 text-left text-xs font-medium tracking-wider text-muted-foreground uppercase"
								>Status</th
							>
							<th
								class="px-6 py-3 text-left text-xs font-medium tracking-wider text-muted-foreground uppercase"
								>Scheduled</th
							>
						</tr>
					</thead>
					<tbody class="divide-y divide-border bg-card">
						{#each posts as post}
							<tr class="transition-colors hover:bg-muted/30">
								<td class="px-6 py-4 whitespace-nowrap">
									<div class="max-w-xs truncate text-sm font-medium">{post.content}</div>
								</td>
								<td class="px-6 py-4 whitespace-nowrap">
									<div class="text-sm text-muted-foreground">
										{getWorkspaceName(post.workspace_id)}
									</div>
								</td>
								<td class="px-6 py-4 whitespace-nowrap">
									<span
										class="inline-flex rounded-full px-2 text-xs leading-5 font-semibold {getStatusColor(
											post.status
										)}"
									>
										{post.status}
									</span>
								</td>
								<td class="px-6 py-4 text-sm whitespace-nowrap text-muted-foreground">
									{post.scheduled_at && post.scheduled_at !== '0001-01-01T00:00:00Z'
										? new Date(post.scheduled_at).toLocaleString()
										: 'Draft'}
								</td>
							</tr>
						{/each}
					</tbody>
				</table>
			</div>
		</Card>
	{/if}
</div>

{#if showCreateWorkspace}
	<!-- svelte-ignore a11y_click_events_have_key_events a11y_no_static_element_interactions -->
	<div
		class="fixed inset-0 z-50 flex items-center justify-center bg-black/50"
		onclick={() => (showCreateWorkspace = false)}
	>
		<Card class="mx-4 w-full max-w-md" onclick={(e: MouseEvent) => e.stopPropagation()}>
			<CardHeader>
				<CardTitle>Workspaces</CardTitle>
				<CardDescription>Manage your workspaces</CardDescription>
			</CardHeader>
			<CardContent class="space-y-4">
				<div class="space-y-2">
					<Label>Existing Workspaces</Label>
					<div class="max-h-40 space-y-1 overflow-y-auto rounded-md border p-2">
						{#each workspaces || [] as ws}
							<div
								class="flex items-center justify-between rounded-sm px-2 py-1 text-sm hover:bg-muted"
							>
								<span>{ws.name}</span>
							</div>
						{/each}
					</div>
				</div>

				<form onsubmit={createWorkspace} class="space-y-4">
					<div class="space-y-2">
						<Label for="workspace-name">New Workspace Name</Label>
						<Input
							type="text"
							id="workspace-name"
							bind:value={newWorkspaceName}
							placeholder="My Workspace"
							required
						/>
					</div>
					<div class="flex justify-end gap-3">
						<Button type="button" variant="outline" onclick={() => (showCreateWorkspace = false)}
							>Close</Button
						>
						<Button type="submit">Create</Button>
					</div>
				</form>
			</CardContent>
		</Card>
	</div>
{/if}
