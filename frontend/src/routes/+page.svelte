<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { auth } from '$lib/stores/auth';
	import { client, type Workspace, type Post } from '$lib/api/client';
	import { Button } from '$lib/components/ui/button';
	import { Card, CardContent } from '$lib/components/ui/card';
	import { ui } from '$lib/stores/ui.svelte';
	import { getStatusColor, getPlatformKey, getPlatformColor, getPlatformName } from '$lib/utils';
	import PlatformIcon from '$lib/components/platform-icon.svelte';
	import PlusIcon from 'lucide-svelte/icons/plus';
	import FileTextIcon from 'lucide-svelte/icons/file-text';
	import ClockIcon from 'lucide-svelte/icons/clock';
	import EditIcon from 'lucide-svelte/icons/pencil';
	import TrashIcon from 'lucide-svelte/icons/trash-2';
	import LoaderIcon from 'lucide-svelte/icons/loader-2';
	import ArrowRightIcon from 'lucide-svelte/icons/arrow-right';
	import CalendarDaysIcon from 'lucide-svelte/icons/calendar-days';
	import LayoutDashboardIcon from 'lucide-svelte/icons/layout-dashboard';
	import SendIcon from 'lucide-svelte/icons/send';

	let workspaces = $state<Workspace[] | null>(null);
	let upcomingPosts = $state<Post[]>([]);
	let draftPosts = $state<Post[]>([]);
	let recentPosts = $state<Post[]>([]);
	let loading = $state(true);
	let activeTab = $state<'upcoming' | 'drafts' | 'all'>('upcoming');

	let isAuthenticated = $state(false);
	let userName = $state('');
	let authReady = $state(false);

	onMount(() => {
		const unsubscribe = auth.subscribe((state) => {
			isAuthenticated = state.isAuthenticated;
			userName = state.user?.email?.split('@')[0] || '';
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
			if (wsErr) {
				goto('/onboarding');
				return;
			}
			workspaces = wsData ?? [];

			if (workspaces.length === 0) {
				goto('/onboarding');
				return;
			}

			const allUpcoming: Post[] = [];
			const allDrafts: Post[] = [];
			const allRecent: Post[] = [];

			for (const ws of workspaces) {
				const { data: upcomingData } = await client.GET('/posts', {
					params: { query: { workspace_id: ws.id, status: 'scheduled', limit: 10 } }
				});
				if (upcomingData) allUpcoming.push(...upcomingData);

				const { data: draftData } = await client.GET('/posts', {
					params: { query: { workspace_id: ws.id, status: 'draft', limit: 10 } }
				});
				if (draftData) allDrafts.push(...draftData);

				const { data: recentData } = await client.GET('/posts', {
					params: { query: { workspace_id: ws.id, limit: 5 } }
				});
				if (recentData) allRecent.push(...recentData);
			}

			upcomingPosts = allUpcoming
				.sort((a, b) => new Date(a.scheduled_at).getTime() - new Date(b.scheduled_at).getTime())
				.slice(0, 7);

			draftPosts = allDrafts.sort(
				(a, b) => new Date(b.created_at).getTime() - new Date(a.created_at).getTime()
			);

			recentPosts = allRecent
				.sort((a, b) => new Date(b.created_at).getTime() - new Date(a.created_at).getTime())
				.slice(0, 10);
		} catch (e) {
			console.error('Failed to load dashboard:', e);
		} finally {
			loading = false;
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

	function truncateContent(text: string, max = 120): string {
		if (text.length <= max) return text;
		return text.slice(0, max).trim() + '...';
	}

	async function deletePost(postId: string) {
		if (!confirm('Delete this post? This cannot be undone.')) return;
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

{#if loading}
	<div class="flex flex-1 items-center justify-center">
		<LoaderIcon class="h-8 w-8 animate-spin text-primary" />
	</div>
{:else}
	<div class="mx-auto w-full max-w-[1360px] px-4 py-6 lg:px-8">
		<div class="mb-8 flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
			<div>
				<h1 class="flex items-center gap-2 text-2xl font-bold tracking-tight">
					<LayoutDashboardIcon class="h-6 w-6 text-primary" />
					Dashboard
				</h1>
				<p class="mt-1 text-sm text-muted-foreground">
					Welcome back, {userName || 'there'}. Here's your content overview.
				</p>
			</div>
			<Button onclick={() => goto('/posts/new')} class="gap-2">
				<PlusIcon class="h-4 w-4" />
				New Post
			</Button>
		</div>

		<!-- Stats Cards -->
		<div class="mb-8 grid grid-cols-1 gap-4 sm:grid-cols-3">
			<Card class="border-l-4 border-l-blue-500">
				<CardContent class="flex items-center gap-4 p-4">
					<div class="flex h-12 w-12 items-center justify-center rounded-xl bg-blue-500/10">
						<CalendarDaysIcon class="h-6 w-6 text-blue-500" />
					</div>
					<div>
						<p class="text-2xl font-bold">{upcomingPosts.length}</p>
						<p class="text-sm text-muted-foreground">Upcoming</p>
					</div>
				</CardContent>
			</Card>

			<Card class="border-l-4 border-l-amber-500">
				<CardContent class="flex items-center gap-4 p-4">
					<div class="flex h-12 w-12 items-center justify-center rounded-xl bg-amber-500/10">
						<FileTextIcon class="h-6 w-6 text-amber-500" />
					</div>
					<div>
						<p class="text-2xl font-bold">{draftPosts.length}</p>
						<p class="text-sm text-muted-foreground">Drafts</p>
					</div>
				</CardContent>
			</Card>

			<Card class="border-l-4 border-l-emerald-500">
				<CardContent class="flex items-center gap-4 p-4">
					<div class="flex h-12 w-12 items-center justify-center rounded-xl bg-emerald-500/10">
						<SendIcon class="h-6 w-6 text-emerald-500" />
					</div>
					<div>
						<p class="text-2xl font-bold">
							{recentPosts.filter((p) => p.status === 'published').length}
						</p>
						<p class="text-sm text-muted-foreground">Published</p>
					</div>
				</CardContent>
			</Card>
		</div>

		<!-- Tab Navigation -->
		<div class="mb-6 flex items-center gap-1 border-b">
			<button
				class="relative flex items-center gap-2 px-4 py-3 text-sm font-medium transition-colors {activeTab ===
				'upcoming'
					? 'border-b-2 border-primary text-primary'
					: 'text-muted-foreground hover:text-foreground'}"
				onclick={() => (activeTab = 'upcoming')}
			>
				<CalendarDaysIcon class="h-4 w-4" />
				Upcoming
				{#if upcomingPosts.length > 0}
					<span class="ml-1 rounded-full bg-primary/10 px-2 py-0.5 text-xs">
						{upcomingPosts.length}
					</span>
				{/if}
			</button>
			<button
				class="relative flex items-center gap-2 px-4 py-3 text-sm font-medium transition-colors {activeTab ===
				'drafts'
					? 'border-b-2 border-primary text-primary'
					: 'text-muted-foreground hover:text-foreground'}"
				onclick={() => (activeTab = 'drafts')}
			>
				<FileTextIcon class="h-4 w-4" />
				Drafts
				{#if draftPosts.length > 0}
					<span class="ml-1 rounded-full bg-primary/10 px-2 py-0.5 text-xs">
						{draftPosts.length}
					</span>
				{/if}
			</button>
			<button
				class="relative flex items-center gap-2 px-4 py-3 text-sm font-medium transition-colors {activeTab ===
				'all'
					? 'border-b-2 border-primary text-primary'
					: 'text-muted-foreground hover:text-foreground'}"
				onclick={() => (activeTab = 'all')}
			>
				All Posts
			</button>
		</div>

		<!-- Post Lists -->
		<div class="grid grid-cols-1 gap-6 lg:grid-cols-3">
			<div class="lg:col-span-2">
				{#if activeTab === 'upcoming'}
					{#if upcomingPosts.length === 0}
						<div
							class="flex flex-col items-center justify-center rounded-lg border border-dashed py-16"
						>
							<CalendarDaysIcon class="mb-4 h-12 w-12 text-muted-foreground/40" />
							<p class="mb-1 text-base font-medium">No upcoming posts</p>
							<p class="mb-4 text-sm text-muted-foreground">Schedule a post to see it here</p>
							<Button variant="outline" onclick={() => goto('/posts/new')}>Schedule a Post</Button>
						</div>
					{:else}
						<div class="space-y-3">
							{#each upcomingPosts as post (post.id)}
								<div class="group rounded-lg border bg-card p-4 transition-all hover:shadow-sm">
									<div class="flex items-start justify-between gap-3">
										<div class="min-w-0 flex-1">
											<div class="mb-2 flex flex-wrap items-center gap-2">
												<span
													class="inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-semibold {getStatusColor(
														post.status
													)}"
												>
													{post.status}
												</span>
												<span class="text-xs text-muted-foreground">
													{getWorkspaceName(post.workspace_id)}
												</span>
											</div>
											<p class="mb-2 text-sm leading-relaxed">{truncateContent(post.content)}</p>
											<div class="flex items-center gap-3 text-xs text-muted-foreground">
												<span class="flex items-center gap-1">
													<ClockIcon class="h-3 w-3" />
													{formatScheduledAt(post.scheduled_at)}
												</span>
												{#if post.destinations && post.destinations.length > 0}
													<div class="flex items-center gap-1">
														{#each post.destinations as dest (dest.social_account_id)}
															<div
																class="flex h-5 w-5 items-center justify-center rounded-full {getPlatformColor(
																	dest.platform
																)}"
																title={dest.platform}
															>
																<PlatformIcon
																	platform={getPlatformKey(dest.platform)}
																	class="h-3 w-3 text-white"
																/>
															</div>
														{/each}
													</div>
												{/if}
											</div>
										</div>
										<div
											class="flex shrink-0 gap-1 opacity-0 transition-opacity group-hover:opacity-100"
										>
											<Button
												variant="ghost"
												size="icon"
												class="h-8 w-8"
												onclick={() => goto(`/posts/${post.id}`)}
											>
												<EditIcon class="h-4 w-4" />
											</Button>
										</div>
									</div>
								</div>
							{/each}
						</div>
					{/if}
				{:else if activeTab === 'drafts'}
					{#if draftPosts.length === 0}
						<div
							class="flex flex-col items-center justify-center rounded-lg border border-dashed py-16"
						>
							<FileTextIcon class="mb-4 h-12 w-12 text-muted-foreground/40" />
							<p class="mb-1 text-base font-medium">No drafts</p>
							<p class="mb-4 text-sm text-muted-foreground">Save a post as draft to see it here</p>
							<Button variant="outline" onclick={() => goto('/posts/new')}>Create Draft</Button>
						</div>
					{:else}
						<div class="space-y-3">
							{#each draftPosts as post (post.id)}
								<div class="group rounded-lg border bg-card p-4 transition-all hover:shadow-sm">
									<div class="flex items-start justify-between gap-3">
										<div class="min-w-0 flex-1">
											<div class="mb-2 flex flex-wrap items-center gap-2">
												<span
													class="inline-flex items-center rounded-full bg-muted px-2.5 py-0.5 text-xs font-semibold text-muted-foreground"
												>
													Draft
												</span>
												<span class="text-xs text-muted-foreground">
													{getWorkspaceName(post.workspace_id)}
												</span>
											</div>
											<p class="mb-1 text-sm leading-relaxed">{truncateContent(post.content)}</p>
											<p class="text-xs text-muted-foreground">
												Created {new Date(post.created_at).toLocaleDateString('en-US', {
													month: 'short',
													day: 'numeric',
													year: 'numeric'
												})}
											</p>
										</div>
										<div
											class="flex shrink-0 gap-1 opacity-0 transition-opacity group-hover:opacity-100"
										>
											<Button
												variant="ghost"
												size="icon"
												class="h-8 w-8"
												onclick={() => goto(`/posts/${post.id}`)}
											>
												<EditIcon class="h-4 w-4" />
											</Button>
											<Button
												variant="ghost"
												size="icon"
												class="h-8 w-8 text-destructive hover:text-destructive"
												onclick={() => deletePost(post.id)}
											>
												<TrashIcon class="h-4 w-4" />
											</Button>
										</div>
									</div>
								</div>
							{/each}
						</div>
					{/if}
				{:else if recentPosts.length === 0}
					<div
						class="flex flex-col items-center justify-center rounded-lg border border-dashed py-16"
					>
						<SendIcon class="mb-4 h-12 w-12 text-muted-foreground/40" />
						<p class="mb-1 text-base font-medium">No posts yet</p>
						<p class="mb-4 text-sm text-muted-foreground">Create your first post to get started</p>
						<Button variant="outline" onclick={() => goto('/posts/new')}>Create Post</Button>
					</div>
				{:else}
					<div class="space-y-3">
						{#each recentPosts as post (post.id)}
							<div class="group rounded-lg border bg-card p-4 transition-all hover:shadow-sm">
								<div class="flex items-start justify-between gap-3">
									<div class="min-w-0 flex-1">
										<div class="mb-2 flex flex-wrap items-center gap-2">
											<span
												class="inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-semibold {getStatusColor(
													post.status
												)}"
											>
												{post.status}
											</span>
											<span class="text-xs text-muted-foreground">
												{getWorkspaceName(post.workspace_id)}
											</span>
										</div>
										<p class="mb-2 text-sm leading-relaxed">{truncateContent(post.content)}</p>
										<div class="flex items-center gap-3 text-xs text-muted-foreground">
											{#if post.scheduled_at && post.scheduled_at !== '0001-01-01T00:00:00Z'}
												<span class="flex items-center gap-1">
													<ClockIcon class="h-3 w-3" />
													{formatScheduledAt(post.scheduled_at)}
												</span>
											{/if}
											{#if post.destinations && post.destinations.length > 0}
												<div class="flex items-center gap-1">
													{#each post.destinations as dest (dest.social_account_id)}
														<div
															class="flex h-5 w-5 items-center justify-center rounded-full {getPlatformColor(
																dest.platform
															)}"
															title={dest.platform}
														>
															<PlatformIcon
																platform={getPlatformKey(dest.platform)}
																class="h-3 w-3 text-white"
															/>
														</div>
													{/each}
												</div>
											{/if}
										</div>
									</div>
									<div
										class="flex shrink-0 gap-1 opacity-0 transition-opacity group-hover:opacity-100"
									>
										{#if post.status !== 'published'}
											<Button
												variant="ghost"
												size="icon"
												class="h-8 w-8"
												onclick={() => goto(`/posts/${post.id}`)}
											>
												<EditIcon class="h-4 w-4" />
											</Button>
										{/if}
									</div>
								</div>
							</div>
						{/each}
					</div>
				{/if}
			</div>

			<!-- Sidebar: Quick Actions + Workspaces -->
			<div class="space-y-6">
				<!-- Quick Compose CTA -->
				<div class="rounded-lg border bg-gradient-to-br from-primary/5 to-primary/10 p-5">
					<p class="mb-1 font-medium">What's on your mind?</p>
					<p class="mb-4 text-sm text-muted-foreground">
						Write something and schedule it for later.
					</p>
					<Button onclick={() => goto('/posts/new')} class="w-full gap-2">
						<PlusIcon class="h-4 w-4" />
						Create Post
					</Button>
				</div>

				<!-- Workspaces -->
				<div class="rounded-lg border bg-card">
					<div class="flex items-center justify-between border-b px-4 py-3">
						<h3 class="text-sm font-medium">Workspaces</h3>
						<span class="text-xs text-muted-foreground">{workspaces?.length ?? 0}</span>
					</div>
					<div class="divide-y">
						{#each workspaces ?? [] as ws (ws.id)}
							<div class="flex items-center gap-3 px-4 py-3">
								<div
									class="flex h-8 w-8 items-center justify-center rounded-lg bg-primary/10 text-xs font-bold text-primary"
								>
									{ws.name.slice(0, 2).toUpperCase()}
								</div>
								<div class="min-w-0 flex-1">
									<p class="truncate text-sm font-medium">{ws.name}</p>
								</div>
							</div>
						{/each}
					</div>
				</div>
			</div>
		</div>
	</div>
{/if}
