<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
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
	import { getStatusColor } from '$lib/utils';
	import { getPlatformKey } from '$lib/utils';
	import PlatformIcon from '$lib/components/platform-icon.svelte';
	import PlusIcon from 'lucide-svelte/icons/plus';
	import FileTextIcon from 'lucide-svelte/icons/file-text';
	import CalendarIcon from 'lucide-svelte/icons/calendar';
	import ClockIcon from 'lucide-svelte/icons/clock';
	import EditIcon from 'lucide-svelte/icons/pencil';
	import TrashIcon from 'lucide-svelte/icons/trash-2';
	import LoaderIcon from 'lucide-svelte/icons/loader-2';

	let workspaces = $state<Workspace[] | null>(null);
	let upcomingPosts = $state<Post[]>([]);
	let draftPosts = $state<Post[]>([]);
	let recentPosts = $state<Post[]>([]);
	let loading = $state(true);
	let error = $state('');
	let showCreateWorkspace = $state(false);
	let newWorkspaceName = $state('');
	let activeTab = $state<'upcoming' | 'drafts' | 'all'>('upcoming');

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

			const allUpcoming: Post[] = [];
			const allDrafts: Post[] = [];
			const allRecent: Post[] = [];

			for (const ws of workspaces ?? []) {
				const { data: upcomingData } = await client.GET('/posts', {
					params: { query: { workspace_id: ws.id, status: 'scheduled', limit: 10 } }
				});
				if (upcomingData) {
					allUpcoming.push(...upcomingData);
				}

				const { data: draftData } = await client.GET('/posts', {
					params: { query: { workspace_id: ws.id, status: 'draft', limit: 10 } }
				});
				if (draftData) {
					allDrafts.push(...draftData);
				}

				const { data: recentData } = await client.GET('/posts', {
					params: { query: { workspace_id: ws.id, limit: 5 } }
				});
				if (recentData) {
					allRecent.push(...recentData);
				}
			}

			upcomingPosts = allUpcoming
				.sort((a, b) => new Date(a.scheduled_at).getTime() - new Date(b.scheduled_at).getTime())
				.slice(0, 7);

			draftPosts = allDrafts.sort(
				(a, b) => new Date(b.created_at).getTime() - new Date(a.created_at).getTime()
			);

			recentPosts = allRecent
				.sort((a, b) => new Date(b.created_at).getTime() - new Date(a.created_at).getTime())
				.slice(0, 5);
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
			if (err) throw new Error((err as any).detail || 'Failed to create workspace');
			newWorkspaceName = '';
			showCreateWorkspace = false;
			await loadDashboard();
		} catch (e) {
			console.error('Failed to create workspace:', e);
			error = (e as Error).message;
		}
	}

	function getWorkspaceName(workspaceId: string): string {
		return workspaces?.find((w) => w.id === workspaceId)?.name || 'Unknown';
	}

	function formatScheduledAt(scheduledAt: string): string {
		if (!scheduledAt || scheduledAt === '0001-01-01T00:00:00Z') return '';
		const date = new Date(scheduledAt);
		const now = new Date();
		const diffMs = date.getTime() - now.getTime();
		const diffHours = Math.floor(diffMs / (1000 * 60 * 60));

		if (diffHours < 24 && date.getDate() === now.getDate()) {
			return `Today at ${date.toLocaleTimeString('en-US', { hour: 'numeric', minute: '2-digit' })}`;
		} else if (diffHours < 48) {
			return `Tomorrow at ${date.toLocaleTimeString('en-US', { hour: 'numeric', minute: '2-digit' })}`;
		} else {
			return (
				date.toLocaleDateString('en-US', { month: 'short', day: 'numeric' }) +
				' at ' +
				date.toLocaleTimeString('en-US', { hour: 'numeric', minute: '2-digit' })
			);
		}
	}

	async function deletePost(postId: string) {
		if (!confirm('Are you sure you want to delete this post?')) return;
		try {
			const { error: err } = await (client as any).DELETE('/posts/{id}', {
				params: { path: { id: postId } }
			});
			if (err) throw new Error((err as any)?.detail || 'Failed to delete post');
			ui.triggerRefresh();
		} catch (e) {
			console.error('Failed to delete post:', e);
		}
	}
</script>

<svelte:head>
	<title>Dashboard - OpenPost</title>
</svelte:head>

<div class="mx-auto w-full max-w-[1360px] px-4 py-6 lg:px-8">
	{#if loading}
		<div class="flex justify-center py-12">
			<LoaderIcon class="h-8 w-8 animate-spin text-primary" />
		</div>
	{:else if error && !isAuthenticated}
		<div class="py-12 text-center">
			<p class="text-muted-foreground">Please log in to view your dashboard.</p>
			<a href="/login" class="mt-2 inline-block font-medium text-primary hover:underline">
				Go to Login
			</a>
		</div>
	{:else}
		<div class="mb-8 flex items-center justify-between">
			<div>
				<h1 class="text-2xl font-bold">Dashboard</h1>
				<p class="text-sm text-muted-foreground">Manage your posts and schedule</p>
			</div>
			<div class="flex gap-2">
				<Button variant="outline" onclick={() => (showCreateWorkspace = true)}>
					Manage Workspaces
				</Button>
			</div>
		</div>

		{#if error}
			<div
				class="mb-4 rounded-md border border-destructive/20 bg-destructive/10 p-4 text-destructive"
			>
				{error}
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
		{/if}

		<div class="grid grid-cols-1 gap-6 lg:grid-cols-3">
			<div class="space-y-6 lg:col-span-2">
				<Card>
					<CardHeader class="pb-3">
						<div class="flex items-center justify-between">
							<div class="flex items-center gap-2">
								<PlusIcon class="h-5 w-5 text-primary" />
								<CardTitle class="text-lg">Create a Post</CardTitle>
							</div>
						</div>
						<CardDescription>Write and schedule content for your social accounts</CardDescription>
					</CardHeader>
					<CardContent>
						<Button onclick={() => goto('/posts/new')} class="w-full">
							<PlusIcon class="mr-2 h-4 w-4" />
							New Post
						</Button>
					</CardContent>
				</Card>

				<div class="space-y-4">
					<div class="flex items-center gap-4 border-b">
						<button
							class="flex items-center gap-2 border-b-2 px-1 py-3 text-sm font-medium transition-colors {activeTab ===
							'upcoming'
								? 'border-primary text-primary'
								: 'border-transparent text-muted-foreground hover:text-foreground'}"
							onclick={() => (activeTab = 'upcoming')}
						>
							<CalendarIcon class="h-4 w-4" />
							Upcoming
							{#if upcomingPosts.length > 0}
								<span class="ml-1 rounded-full bg-primary/10 px-2 py-0.5 text-xs text-primary">
									{upcomingPosts.length}
								</span>
							{/if}
						</button>
						<button
							class="flex items-center gap-2 border-b-2 px-1 py-3 text-sm font-medium transition-colors {activeTab ===
							'drafts'
								? 'border-primary text-primary'
								: 'border-transparent text-muted-foreground hover:text-foreground'}"
							onclick={() => (activeTab = 'drafts')}
						>
							<FileTextIcon class="h-4 w-4" />
							Drafts
							{#if draftPosts.length > 0}
								<span class="ml-1 rounded-full bg-primary/10 px-2 py-0.5 text-xs text-primary">
									{draftPosts.length}
								</span>
							{/if}
						</button>
						<button
							class="flex items-center gap-2 border-b-2 px-1 py-3 text-sm font-medium transition-colors {activeTab ===
							'all'
								? 'border-primary text-primary'
								: 'border-transparent text-muted-foreground hover:text-foreground'}"
							onclick={() => (activeTab = 'all')}
						>
							All Posts
						</button>
					</div>

					{#if activeTab === 'upcoming'}
						{#if upcomingPosts.length === 0}
							<Card>
								<CardContent class="flex flex-col items-center justify-center py-12">
									<CalendarIcon class="mb-3 h-12 w-12 text-muted-foreground/50" />
									<p class="mb-1 text-sm font-medium">No upcoming posts</p>
									<p class="mb-4 text-xs text-muted-foreground">Schedule a post to see it here</p>
									<Button variant="outline" size="sm" onclick={() => goto('/posts/new')}>
										Create Post
									</Button>
								</CardContent>
							</Card>
						{:else}
							<div class="space-y-3">
								{#each upcomingPosts as post}
									<Card class="transition-colors hover:bg-muted/30">
										<CardContent class="flex items-start justify-between gap-4 p-4">
											<div class="min-w-0 flex-1">
												<div class="mb-1 flex items-center gap-2">
													<span
														class="inline-flex rounded-full px-2 py-0.5 text-xs leading-5 font-semibold {getStatusColor(
															post.status
														)}"
													>
														{post.status}
													</span>
													<span class="text-xs text-muted-foreground">
														{getWorkspaceName(post.workspace_id)}
													</span>
												</div>
												<p class="mb-2 line-clamp-2 text-sm">{post.content}</p>
												<div class="flex items-center gap-1 text-xs text-muted-foreground">
													<ClockIcon class="h-3 w-3" />
													{formatScheduledAt(post.scheduled_at)}
												</div>
												{#if post.destinations && post.destinations.length > 0}
													<div class="mt-2 flex items-center gap-1">
														{#each post.destinations as dest}
															<div
																class="flex h-6 w-6 items-center justify-center rounded-full bg-muted"
																title={dest.platform}
															>
																<PlatformIcon
																	platform={getPlatformKey(dest.platform)}
																	class="h-3.5 w-3.5"
																/>
															</div>
														{/each}
													</div>
												{/if}
											</div>
											<div class="flex flex-col gap-1">
												<Button
													variant="ghost"
													size="sm"
													class="h-8 w-8 p-0"
													onclick={() => goto(`/posts/${post.id}`)}
												>
													<EditIcon class="h-4 w-4" />
												</Button>
											</div>
										</CardContent>
									</Card>
								{/each}
							</div>
						{/if}
					{:else if activeTab === 'drafts'}
						{#if draftPosts.length === 0}
							<Card>
								<CardContent class="flex flex-col items-center justify-center py-12">
									<FileTextIcon class="mb-3 h-12 w-12 text-muted-foreground/50" />
									<p class="mb-1 text-sm font-medium">No drafts</p>
									<p class="mb-4 text-xs text-muted-foreground">
										Save a post as draft to see it here
									</p>
									<Button variant="outline" size="sm" onclick={() => goto('/posts/new')}>
										Create Draft
									</Button>
								</CardContent>
							</Card>
						{:else}
							<div class="space-y-3">
								{#each draftPosts as post}
									<Card class="transition-colors hover:bg-muted/30">
										<CardContent class="flex items-start justify-between gap-4 p-4">
											<div class="min-w-0 flex-1">
												<div class="mb-1 flex items-center gap-2">
													<span
														class="inline-flex rounded-full bg-muted px-2 py-0.5 text-xs leading-5 font-semibold text-muted-foreground"
													>
														Draft
													</span>
													<span class="text-xs text-muted-foreground">
														{getWorkspaceName(post.workspace_id)}
													</span>
												</div>
												<p class="mb-2 line-clamp-2 text-sm">{post.content}</p>
												<div class="text-xs text-muted-foreground">
													Created {new Date(post.created_at).toLocaleDateString('en-US', {
														month: 'short',
														day: 'numeric'
													})}
												</div>
											</div>
											<div class="flex flex-col gap-1">
												<Button
													variant="ghost"
													size="sm"
													class="h-8 w-8 p-0"
													onclick={() => goto(`/posts/${post.id}`)}
												>
													<EditIcon class="h-4 w-4" />
												</Button>
												<Button
													variant="ghost"
													size="sm"
													class="h-8 w-8 p-0 text-destructive hover:text-destructive"
													onclick={() => deletePost(post.id)}
												>
													<TrashIcon class="h-4 w-4" />
												</Button>
											</div>
										</CardContent>
									</Card>
								{/each}
							</div>
						{/if}
					{:else if recentPosts.length === 0}
						<Card>
							<CardContent class="flex flex-col items-center justify-center py-12">
								<p class="mb-1 text-sm font-medium">No posts yet</p>
								<p class="mb-4 text-xs text-muted-foreground">
									Create your first post to get started
								</p>
								<Button variant="outline" size="sm" onclick={() => goto('/posts/new')}>
									Create Post
								</Button>
							</CardContent>
						</Card>
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
											<th
												class="px-6 py-3 text-right text-xs font-medium tracking-wider text-muted-foreground uppercase"
												>Actions</th
											>
										</tr>
									</thead>
									<tbody class="divide-y divide-border bg-card">
										{#each recentPosts as post}
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
												<td class="px-6 py-4 text-right whitespace-nowrap">
													<Button
														variant="ghost"
														size="sm"
														class="h-8 w-8 p-0"
														onclick={() => goto(`/posts/${post.id}`)}
													>
														<EditIcon class="h-4 w-4" />
													</Button>
												</td>
											</tr>
										{/each}
									</tbody>
								</table>
							</div>
						</Card>
					{/if}
				</div>
			</div>

			<div class="space-y-6">
				<Card>
					<CardHeader class="pb-3">
						<CardTitle class="text-lg">Quick Stats</CardTitle>
					</CardHeader>
					<CardContent class="space-y-4">
						<div class="flex items-center justify-between">
							<span class="text-sm text-muted-foreground">Upcoming Posts</span>
							<span class="text-lg font-semibold">{upcomingPosts.length}</span>
						</div>
						<div class="flex items-center justify-between">
							<span class="text-sm text-muted-foreground">Drafts</span>
							<span class="text-lg font-semibold">{draftPosts.length}</span>
						</div>
						<div class="flex items-center justify-between">
							<span class="text-sm text-muted-foreground">Workspaces</span>
							<span class="text-lg font-semibold">{workspaces?.length ?? 0}</span>
						</div>
					</CardContent>
				</Card>

				{#if workspaces && workspaces.length > 0}
					<Card>
						<CardHeader class="pb-3">
							<CardTitle class="text-lg">Workspaces</CardTitle>
						</CardHeader>
						<CardContent class="space-y-2">
							{#each workspaces ?? [] as ws}
								<div class="flex items-center justify-between rounded-md border p-2 text-sm">
									<span>{ws.name}</span>
								</div>
							{/each}
						</CardContent>
					</Card>
				{/if}
			</div>
		</div>
	{/if}
</div>

{#if showCreateWorkspace}
	<div
		class="fixed inset-0 z-50 flex items-center justify-center bg-black/50"
		role="dialog"
		aria-modal="true"
		tabindex="-1"
		onkeydown={(e) => e.key === 'Escape' && (showCreateWorkspace = false)}
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
