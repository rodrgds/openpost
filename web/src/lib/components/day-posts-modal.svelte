<script lang="ts">
	import * as Dialog from '$lib/components/ui/dialog';
	import * as Drawer from '$lib/components/ui/drawer';
	import { Button } from '$lib/components/ui/button';
	import { Card, CardContent } from '$lib/components/ui/card';
	import { IsMobile } from '$lib/hooks/is-mobile.svelte';
	import { client, type Post } from '$lib/api/client';
	import { ui } from '$lib/stores/ui.svelte';
	import { getLocalTimeZone, today, type DateValue } from '@internationalized/date';
	import PlusIcon from 'lucide-svelte/icons/plus';
	import LoaderIcon from 'lucide-svelte/icons/loader-2';
	import CalendarIcon from 'lucide-svelte/icons/calendar';

	const isMobile = new IsMobile();

	let posts = $state<Post[]>([]);
	let loading = $state(false);
	let error = $state('');
	let open = $state(false);

	let currentDate = $derived<DateValue | undefined>(ui.dayPostsDate);
	let dateStr = $derived(
		currentDate ? currentDate.toDate(getLocalTimeZone()).toISOString().split('T')[0] : ''
	);
	let isFutureDay = $derived.by(() => {
		if (!currentDate) return false;
		const todayDate = today(getLocalTimeZone());
		return currentDate.compare(todayDate) >= 0;
	});
	let formattedDate = $derived.by(() => {
		if (!currentDate) return '';
		return currentDate.toDate(getLocalTimeZone()).toLocaleDateString('en-US', {
			weekday: 'long',
			month: 'long',
			day: 'numeric',
			year: 'numeric'
		});
	});

	$effect(() => {
		open = ui.isDayPostsOpen;
		if (open && dateStr) {
			loadPosts(dateStr);
		}
	});

	function handleOpenChange(isOpen: boolean) {
		open = isOpen;
		if (!isOpen) {
			ui.closeDayPosts();
		}
	}

	async function loadPosts(date: string) {
		loading = true;
		error = '';
		try {
			const { data, error: err } = await client.GET('/posts', {
				params: { query: { date } }
			});
			if (err) throw new Error('Failed to load posts');
			posts = data ?? [];
		} catch (e) {
			error = (e as Error).message;
			posts = [];
		} finally {
			loading = false;
		}
	}

	function getTime(iso: string): string {
		const d = new Date(iso);
		return d.toLocaleTimeString('en-US', { hour: '2-digit', minute: '2-digit', hour12: false });
	}

	function getStatusColor(status: string): string {
		switch (status) {
			case 'scheduled':
				return 'bg-blue-500/10 text-blue-600 dark:text-blue-400';
			case 'publishing':
				return 'bg-yellow-500/10 text-yellow-600 dark:text-yellow-400';
			case 'published':
				return 'bg-green-500/10 text-green-600 dark:text-green-400';
			case 'failed':
				return 'bg-red-500/10 text-red-600 dark:text-red-400';
			default:
				return 'bg-muted text-muted-foreground';
		}
	}

	function getPlatformIcon(platform: string): string {
		switch (platform) {
			case 'x':
				return '\u{1D54F}';
			case 'mastodon':
				return '\u{1F418}';
			case 'threads':
				return '\u{1F4F8}';
			case 'bluesky':
				return '\u{1F98B}';
			case 'linkedin':
				return '\u{1F4BC}';
			default:
				return '?';
		}
	}

	function handleNewPost() {
		if (currentDate) {
			ui.openComposeForDay(currentDate);
		}
	}
</script>

{#if !isMobile.current}
	<Dialog.Root {open} onOpenChange={handleOpenChange}>
		<Dialog.Content class="p-6 sm:max-w-[640px]">
			<Dialog.Header>
				<Dialog.Title class="flex items-center gap-2 text-xl font-bold">
					<CalendarIcon class="size-5" />
					{formattedDate}
				</Dialog.Title>
				<Dialog.Description>
					{posts.length} scheduled post{posts.length !== 1 ? 's' : ''}
				</Dialog.Description>
			</Dialog.Header>
			<div class="mt-4 space-y-4">
				{#if loading}
					<div class="flex justify-center py-8">
						<LoaderIcon class="size-6 animate-spin text-muted-foreground" />
					</div>
				{:else if error}
					<div
						class="rounded-md border border-destructive/20 bg-destructive/10 p-3 text-sm text-destructive"
					>
						{error}
					</div>
				{:else if posts.length === 0}
					<div class="flex flex-col items-center gap-3 py-8 text-center text-muted-foreground">
						<CalendarIcon class="size-10 opacity-40" />
						<p class="text-sm">No posts scheduled for this day.</p>
					</div>
				{:else}
					<div class="grid max-h-[50vh] gap-3 overflow-y-auto">
						{#each posts as post (post.id)}
							<Card class="gap-0 p-0 shadow-none">
								<CardContent class="p-4">
									<div class="flex items-start justify-between gap-3">
										<div class="min-w-0 flex-1">
											<p class="line-clamp-2 text-sm">{post.content}</p>
										</div>
										<span
											class="shrink-0 rounded-full px-2 py-0.5 text-xs font-medium {getStatusColor(
												post.status
											)}"
										>
											{post.status}
										</span>
									</div>
									<div class="mt-2 flex items-center justify-between">
										<div class="flex items-center gap-1.5">
											{#each post.destinations ?? [] as dest (dest.social_account_id)}
												<span
													class="flex size-6 items-center justify-center rounded-full bg-primary text-xs text-primary-foreground"
													title={dest.platform}
												>
													{getPlatformIcon(dest.platform)}
												</span>
											{/each}
										</div>
										<span class="text-xs text-muted-foreground">
											{getTime(post.scheduled_at)}
										</span>
									</div>
								</CardContent>
							</Card>
						{/each}
					</div>
				{/if}

				{#if isFutureDay}
					<Button class="w-full gap-2" onclick={handleNewPost}>
						<PlusIcon class="size-4" />
						New Post for This Day
					</Button>
				{/if}
			</div>
		</Dialog.Content>
	</Dialog.Root>
{:else}
	<Drawer.Root {open} onOpenChange={handleOpenChange}>
		<Drawer.Content class="max-h-[95vh]">
			<div class="scrollbar-hide mx-auto w-full max-w-4xl overflow-auto p-6">
				<Drawer.Header class="px-0">
					<Drawer.Title class="flex items-center gap-2 text-xl font-bold">
						<CalendarIcon class="size-5" />
						{formattedDate}
					</Drawer.Title>
					<Drawer.Description>
						{posts.length} scheduled post{posts.length !== 1 ? 's' : ''}
					</Drawer.Description>
				</Drawer.Header>
				<div class="mt-4 space-y-4">
					{#if loading}
						<div class="flex justify-center py-8">
							<LoaderIcon class="size-6 animate-spin text-muted-foreground" />
						</div>
					{:else if error}
						<div
							class="rounded-md border border-destructive/20 bg-destructive/10 p-3 text-sm text-destructive"
						>
							{error}
						</div>
					{:else if posts.length === 0}
						<div class="flex flex-col items-center gap-3 py-8 text-center text-muted-foreground">
							<CalendarIcon class="size-10 opacity-40" />
							<p class="text-sm">No posts scheduled for this day.</p>
						</div>
					{:else}
						<div class="grid max-h-[50vh] gap-3 overflow-y-auto">
							{#each posts as post (post.id)}
								<Card class="gap-0 p-0 shadow-none">
									<CardContent class="p-4">
										<div class="flex items-start justify-between gap-3">
											<div class="min-w-0 flex-1">
												<p class="line-clamp-2 text-sm">{post.content}</p>
											</div>
											<span
												class="shrink-0 rounded-full px-2 py-0.5 text-xs font-medium {getStatusColor(
													post.status
												)}"
											>
												{post.status}
											</span>
										</div>
										<div class="mt-2 flex items-center justify-between">
											<div class="flex items-center gap-1.5">
												{#each post.destinations ?? [] as dest (dest.social_account_id)}
													<span
														class="flex size-6 items-center justify-center rounded-full bg-primary text-xs text-primary-foreground"
														title={dest.platform}
													>
														{getPlatformIcon(dest.platform)}
													</span>
												{/each}
											</div>
											<span class="text-xs text-muted-foreground">
												{getTime(post.scheduled_at)}
											</span>
										</div>
									</CardContent>
								</Card>
							{/each}
						</div>
					{/if}

					{#if isFutureDay}
						<Button class="w-full gap-2" onclick={handleNewPost}>
							<PlusIcon class="size-4" />
							New Post for This Day
						</Button>
					{/if}
				</div>
			</div>
		</Drawer.Content>
	</Drawer.Root>
{/if}
