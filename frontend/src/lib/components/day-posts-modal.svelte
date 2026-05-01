<script lang="ts">
	import * as Dialog from '$lib/components/ui/dialog';
	import { Button } from '$lib/components/ui/button';
	import { Card, CardContent } from '$lib/components/ui/card';
	import { client, type Post } from '$lib/api/client';
	import { ui } from '$lib/stores/ui.svelte';
	import { workspaceCtx } from '$lib/stores/workspace.svelte';
	import { getLocalTimeZone, today, type DateValue } from '@internationalized/date';
	import PlusIcon from 'lucide-svelte/icons/plus';
	import CalendarIcon from 'lucide-svelte/icons/calendar';
	import TrashIcon from 'lucide-svelte/icons/trash-2';
	import PencilIcon from 'lucide-svelte/icons/pencil';
	import { Skeleton } from '$lib/components/ui/skeleton/index.js';
	import { getStatusColor } from '$lib/utils';
	import PlatformIcon from '$lib/components/platform-icon.svelte';
	import { goto } from '$app/navigation';
	import { m } from '$lib/paraglide/messages';
	import { getLocaleTag } from '$lib/i18n';

	let posts = $state<Post[]>([]);
	let loading = $state(false);
	let error = $state('');
	let open = $state(false);

	let currentDate = $derived<DateValue | undefined>(ui.dayPostsDate);
	let dateStr = $derived(currentDate ? currentDate.toString() : '');
	let isFutureDay = $derived.by(() => {
		if (!currentDate) return false;
		const todayDate = today(getLocalTimeZone());
		return currentDate.compare(todayDate) >= 0;
	});
	let formattedDate = $derived.by(() => {
		if (!currentDate) return '';
		return currentDate.toDate(getLocalTimeZone()).toLocaleDateString(getLocaleTag(), {
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
			const workspaceId = workspaceCtx.currentWorkspace?.id;
			const { data, error: err } = await client.GET('/posts', {
				params: { query: { date, ...(workspaceId ? { workspace_id: workspaceId } : {}) } }
			});
			if (err) throw new Error(m.day_posts_load_failed());
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
		return d.toLocaleTimeString(getLocaleTag(), {
			hour: '2-digit',
			minute: '2-digit',
			hour12: false,
			timeZone: workspaceCtx.settings.timezone || 'UTC'
		});
	}

	function handleNewPost() {
		ui.closeDayPosts();
		goto('/');
	}

	async function handleDelete(postId: string) {
		if (!confirm(m.day_posts_delete_confirm())) return;
		try {
			const { error: err } = await (client as any).DELETE('/posts/{id}', {
				params: { path: { id: postId } }
			});
			if (err) throw new Error((err as any).detail || m.day_posts_delete_failed());
			loadPosts(dateStr);
			ui.triggerRefresh();
		} catch (e) {
			console.error('Failed to delete post:', e);
		}
	}

	async function handleReschedule(postId: string) {
		const newDate = prompt(m.day_posts_reschedule_date());
		if (!newDate) return;
		const newTime = prompt(m.day_posts_reschedule_time());
		if (!newTime) return;
		try {
			const scheduledAt = new Date(`${newDate}T${newTime}:00`).toISOString();
			await (client as any).PATCH('/posts/{id}', {
				params: { path: { id: postId } },
				body: { scheduled_at: scheduledAt }
			});
			loadPosts(dateStr);
			ui.triggerRefresh();
		} catch (e) {
			console.error('Failed to reschedule post:', e);
		}
	}
</script>

<Dialog.Root {open} onOpenChange={handleOpenChange}>
	<Dialog.Content
		class="max-h-[90dvh] min-h-0 w-[calc(100%-1rem)] touch-pan-y overflow-y-auto overscroll-contain p-0 sm:w-full sm:max-w-[640px]"
	>
		<div class="min-h-0 p-4 sm:p-6">
			<Dialog.Header>
				<Dialog.Title class="flex items-center gap-2 text-xl font-semibold">
					<CalendarIcon class="size-5" />
					{formattedDate}
				</Dialog.Title>
				<Dialog.Description>
					{m.day_posts_scheduled_count({ count: posts.length })}
				</Dialog.Description>
			</Dialog.Header>
			<div class="mt-4 space-y-4">
				{#if loading}
					<div class="space-y-4 py-4">
						{#each [1, 2, 3] as _}
							<div class="flex items-start gap-3">
								<Skeleton class="h-8 w-8 shrink-0 rounded-full" />
								<div class="flex flex-1 flex-col gap-2">
									<Skeleton class="h-3 w-full rounded" />
									<Skeleton class="h-3 w-2/3 rounded" />
								</div>
							</div>
						{/each}
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
						<p class="text-sm">{m.day_posts_empty()}</p>
					</div>
				{:else}
					<div class="grid max-h-[55dvh] gap-3 overflow-y-auto">
						{#each posts as post (post.id)}
							<Card class="group gap-0 p-0 shadow-none">
								<CardContent class="p-4">
									<div class="flex items-start justify-between gap-3">
										<div class="min-w-0 flex-1">
											<p class="line-clamp-2 text-sm">{post.content}</p>
										</div>
										<div class="flex shrink-0 items-center gap-1">
											<button
												type="button"
												class="rounded p-1 text-muted-foreground opacity-0 transition-opacity group-hover:opacity-100 hover:bg-muted hover:text-foreground"
												onclick={(e) => {
													e.stopPropagation();
													handleReschedule(post.id);
												}}
												title={m.day_posts_reschedule()}
											>
												<PencilIcon class="h-3.5 w-3.5" />
											</button>
											<button
												type="button"
												class="rounded p-1 text-muted-foreground opacity-0 transition-opacity group-hover:opacity-100 hover:bg-muted hover:text-destructive"
												onclick={(e) => {
													e.stopPropagation();
													handleDelete(post.id);
												}}
												title={m.common_delete()}
											>
												<TrashIcon class="h-3.5 w-3.5" />
											</button>
											<span
												class="rounded-full px-2 py-0.5 text-xs font-medium {getStatusColor(
													post.status
												)}"
											>
												{post.status}
											</span>
										</div>
									</div>
									<div class="mt-2 flex items-center justify-between">
										<div class="flex items-center gap-1.5">
											{#each post.destinations ?? [] as dest (dest.social_account_id)}
												<span
													class="flex size-6 items-center justify-center rounded-full bg-primary text-xs text-primary-foreground"
													title={dest.platform}
												>
													<PlatformIcon platform={dest.platform} class="size-4" />
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
						{m.day_posts_new_for_day()}
					</Button>
				{/if}
			</div>
		</div>
	</Dialog.Content>
</Dialog.Root>
