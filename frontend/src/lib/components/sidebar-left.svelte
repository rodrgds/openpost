<script lang="ts">
	import { onMount } from 'svelte';
	import { client, type ScheduleOverview, type Post } from '$lib/api/client';
	import * as Sidebar from '$lib/components/ui/sidebar';
	import * as DropdownMenu from '$lib/components/ui/dropdown-menu';
	import * as Avatar from '$lib/components/ui/avatar';
	import * as CalendarUi from '$lib/components/ui/calendar';
	import Logo from './Logo.svelte';
	import DayPostsModal from './day-posts-modal.svelte';
	import FileTextIcon from 'lucide-svelte/icons/file-text';
	import LogOutIcon from 'lucide-svelte/icons/log-out';
	import ChevronsUpDownIcon from 'lucide-svelte/icons/chevrons-up-down';
	import CircleDotIcon from 'lucide-svelte/icons/circle-dot';
	import LightbulbIcon from 'lucide-svelte/icons/lightbulb';
	import UsersIcon from 'lucide-svelte/icons/users';
	import ImageIcon from 'lucide-svelte/icons/image';
	import SettingsIcon from 'lucide-svelte/icons/settings';
	import TrashIcon from 'lucide-svelte/icons/trash-2';
	import ScrollTextIcon from 'lucide-svelte/icons/scroll-text';
	import { auth } from '$lib/stores/auth';
	import { goto } from '$app/navigation';
	import { toggleMode } from 'mode-watcher';
	import SunIcon from 'lucide-svelte/icons/sun';
	import MoonIcon from 'lucide-svelte/icons/moon';
	import ServerIcon from 'lucide-svelte/icons/server';
	import type { DateValue } from '@internationalized/date';
	import { IS_CAPACITOR } from '$lib/env';
	import { instanceStore } from '$lib/stores/instance.svelte';
	import { recreateClient } from '$lib/api/client';
	import { getLocalTimeZone, today } from '@internationalized/date';
	import { ui } from '$lib/stores/ui.svelte';
	import { workspaceCtx } from '$lib/stores/workspace.svelte';
	import { Skeleton } from '$lib/components/ui/skeleton/index.js';

	let authState = $derived($auth);
	const sidebar = Sidebar.useSidebar();

	// Calendar state
	let selectedDate = $state<DateValue | undefined>(undefined);
	let calendarPlaceholder = $state<DateValue>(today(getLocalTimeZone()));
	let overview = $state<ScheduleOverview | null>(null);
	let loadingSchedule = $state(true);

	// Drafts state
	let drafts = $state<Post[]>([]);
	let loadingDrafts = $state(false);

	const monthString = $derived.by(() => {
		const jsDate = calendarPlaceholder.toDate(getLocalTimeZone());
		const year = jsDate.getFullYear();
		const month = String(jsDate.getMonth() + 1).padStart(2, '0');
		return `${year}-${month}`;
	});

	const dayCounts = $derived.by(() => {
		const map = new Map<string, number>();
		if (!overview) return map;
		for (const item of overview?.days ?? []) {
			map.set(item.date, item.count);
		}
		return map;
	});

	const dayWorkspaceCounts = $derived.by(() => {
		const map = new Map<string, { workspace_id: string; count: number }[]>();
		if (!overview) return map;
		for (const item of overview?.days ?? []) {
			// @ts-ignore - added via backend change
			map.set(item.date, item.workspaces || []);
		}
		return map;
	});

	const workspaceColors = [
		'bg-blue-500',
		'bg-emerald-500',
		'bg-violet-500',
		'bg-orange-500',
		'bg-rose-500',
		'bg-amber-500',
		'bg-cyan-500',
		'bg-indigo-500'
	];

	function getWorkspaceColor(workspaceId: string) {
		let hash = 0;
		for (let i = 0; i < workspaceId.length; i++) {
			hash = workspaceId.charCodeAt(i) + ((hash << 5) - hash);
		}
		return workspaceColors[Math.abs(hash) % workspaceColors.length];
	}

	onMount(async () => {
		loadOverview();
		await workspaceCtx.initialize();
		loadDrafts();
	});

	// Track previous month to detect actual changes
	let previousMonth = $state('');
	let previousWorkspaceId = $state('');

	$effect(() => {
		const currentMonth = monthString;
		const currentWorkspaceId = workspaceCtx.currentWorkspace?.id ?? '';
		if (
			(previousMonth && previousMonth !== currentMonth) ||
			(previousWorkspaceId && previousWorkspaceId !== currentWorkspaceId)
		) {
			loadOverview();
		}
		previousMonth = currentMonth;
		previousWorkspaceId = currentWorkspaceId;
	});

	// Trigger day-posts modal on date selection
	$effect(() => {
		if (selectedDate) {
			ui.openDayPosts(selectedDate);
			// Reset selectedDate so clicking it again triggers the effect
			setTimeout(() => {
				selectedDate = undefined;
			}, 100);
		}
	});

	// Refresh drafts when ui.refreshCounter changes
	$effect(() => {
		if (ui.refreshCounter > 0) {
			loadDrafts();
			loadOverview();
		}
	});

	async function loadOverview() {
		loadingSchedule = true;
		try {
			const workspaceId = workspaceCtx.currentWorkspace?.id;
			const { data, error: err } = await client.GET('/posts/schedule-overview', {
				params: {
					query: {
						month: monthString,
						...(workspaceId ? { workspace_id: workspaceId } : {})
					}
				}
			});
			if (err || !data) throw new Error('Failed to load');
			overview = data;
		} catch {
			overview = null;
		} finally {
			loadingSchedule = false;
		}
	}

	async function loadDrafts() {
		loadingDrafts = true;
		try {
			const workspaceId = workspaceCtx.currentWorkspace?.id;
			if (!workspaceId) {
				drafts = [];
				return;
			}
			const { data, error: err } = await client.GET('/posts', {
				params: {
					query: {
						workspace_id: workspaceId,
						status: 'draft',
						limit: 20
					}
				}
			});
			if (err || !data) throw new Error('Failed to load drafts');
			drafts = data;
		} catch {
			drafts = [];
		} finally {
			loadingDrafts = false;
		}
	}

	async function deleteDraft(postId: string) {
		if (!confirm('Delete this draft?')) return;
		try {
			const { error: err } = await (client as any).DELETE('/posts/{id}', {
				params: { path: { id: postId } }
			});
			if (err) throw new Error((err as any)?.detail || 'Failed to delete');
			loadDrafts();
		} catch (e) {
			console.error('Failed to delete draft:', e);
		}
	}

	function handleLogout() {
		auth.logout();
		goto('/login');
	}

	function handleSwitchServer() {
		auth.logout();
		instanceStore().clearInstanceUrl();
		recreateClient();
		goto('/connect');
	}

	type DayMarkerArgs = {
		day: DateValue;
		outsideMonth: boolean;
	};

	function truncate(text: string, max: number = 40): string {
		if (text.startsWith('__openpost_thread__:')) {
			try {
				const data = JSON.parse(text.slice('__openpost_thread__:'.length));
				const firstPost = Array.isArray(data) && data.length > 0 ? data[0] : null;
				const content = firstPost?.c ?? '';
				const suffix = data.length > 1 ? ` (thread: ${data.length} posts)` : '';
				if (content.length + suffix.length <= max) return content + suffix;
				return content.slice(0, max - suffix.length - 3).trim() + '...' + suffix;
			} catch {
				return 'Thread draft';
			}
		}
		if (text.length <= max) return text;
		return text.slice(0, max).trim() + '...';
	}

	function draftHasMedia(draft: Post): boolean {
		// Check explicit media_ids first (populated by ListPosts)
		if (draft.media_ids && draft.media_ids.length > 0) return true;
		// Fallback: parse thread JSON for legacy/thread drafts
		if (draft.content.startsWith('__openpost_thread__:')) {
			try {
				const data = JSON.parse(draft.content.slice('__openpost_thread__:'.length));
				if (!Array.isArray(data)) return false;
				return data.some((item: any) => (item.m ?? []).length > 0);
			} catch {
				return false;
			}
		}
		return false;
	}
</script>

{#snippet dayMarker({ day, outsideMonth }: DayMarkerArgs)}
	{@const key = day.toString()}
	{@const count = dayCounts.get(key) || 0}
	{@const wsCounts = dayWorkspaceCounts.get(key) || []}
	{@const dots = wsCounts.slice(0, 3)}
	<div class="relative flex size-(--cell-size) items-center justify-center">
		<CalendarUi.Day />
		{#if !outsideMonth && count > 0}
			<div class="pointer-events-none absolute bottom-0.5 flex items-center justify-center gap-0.5">
				{#each dots as marker (`${key}-${marker.workspace_id}`)}
					<span class={`h-1 w-1 rounded-full ${getWorkspaceColor(marker.workspace_id)}`}></span>
				{/each}
			</div>
		{/if}
	</div>
{/snippet}

<Sidebar.Root>
	<Sidebar.Header>
		<!-- Logo -->
		<div class="flex items-center justify-center px-2 py-4">
			<a href="/" class="transition-opacity hover:opacity-90">
				<Logo width={28} height={28} showText={true} />
			</a>
		</div>

		<Sidebar.Separator />
	</Sidebar.Header>

	<Sidebar.Content>
		<!-- Calendar Section -->
		<Sidebar.Group class="px-0 pt-2">
			<Sidebar.GroupLabel
				class="px-4 text-xs font-semibold tracking-wider text-sidebar-foreground/50 uppercase"
				>Schedule</Sidebar.GroupLabel
			>
			<Sidebar.GroupContent>
				<CalendarUi.Calendar
					type="single"
					bind:value={selectedDate}
					bind:placeholder={calendarPlaceholder}
					day={dayMarker}
					weekStartsOn={workspaceCtx.settings.week_start as 0 | 1 | 2 | 3 | 4 | 5 | 6}
					class="mx-auto bg-transparent p-2 select-none [--cell-size:--spacing(8)] [&_[role=gridcell]_[role=button][data-today]]:bg-sidebar-primary [&_[role=gridcell]_[role=button][data-today]]:text-sidebar-primary-foreground [&_tr]:justify-center"
				/>
			</Sidebar.GroupContent>
		</Sidebar.Group>

		{#if overview && overview.days && overview.days.some((d: { count: number }) => d.count > 0)}
			<Sidebar.Group>
				<Sidebar.GroupLabel
					class="text-xs font-semibold tracking-wider text-sidebar-foreground/50 uppercase"
					>Upcoming</Sidebar.GroupLabel
				>
				<Sidebar.GroupContent>
					<Sidebar.Menu>
						<Sidebar.MenuItem>
							<Sidebar.MenuButton class="text-sidebar-foreground/80">
								<CircleDotIcon class="size-3.5" />
								<span
									>{loadingSchedule
										? ''
										: `${overview.days.reduce((s: number, d: { count: number }) => s + d.count, 0)} scheduled posts`}</span
								>
							</Sidebar.MenuButton>
						</Sidebar.MenuItem>
					</Sidebar.Menu>
				</Sidebar.GroupContent>
			</Sidebar.Group>
		{/if}

		<Sidebar.Separator />

		<!-- Drafts Section -->
		<Sidebar.Group class="flex-1 overflow-hidden">
			<Sidebar.GroupLabel
				class="px-4 text-xs font-semibold tracking-wider text-sidebar-foreground/50 uppercase"
			>
				Drafts
				{#if drafts.length > 0}
					<span class="ml-1 text-sidebar-foreground/40">({drafts.length})</span>
				{/if}
			</Sidebar.GroupLabel>
			<Sidebar.GroupContent class="max-h-64 overflow-y-auto">
				{#if loadingDrafts}
					<div class="space-y-2 px-2 py-2">
						{#each Array(4) as _}
							<div class="flex items-center gap-2 px-2 py-1.5">
								<Skeleton class="h-3.5 w-3.5 rounded-sm" />
								<Skeleton class="h-3.5 w-full" />
							</div>
						{/each}
					</div>
				{:else if drafts.length === 0}
					<div class="px-4 py-3 text-sm text-sidebar-foreground/40">
						No drafts yet. Start writing and your draft will appear here.
					</div>
				{:else}
					<Sidebar.Menu>
						{#each drafts as draft (draft.id)}
							<Sidebar.MenuItem>
								<Sidebar.MenuButton
									class="group relative text-sidebar-foreground/80"
									onclick={() => goto(`/posts/${draft.id}`)}
								>
									<FileTextIcon class="size-3.5 shrink-0" />
									<span class="truncate text-sm">{truncate(draft.content)}</span>
									{#if draftHasMedia(draft)}
										<ImageIcon class="size-3 shrink-0 text-sidebar-foreground/40" />
									{/if}
								</Sidebar.MenuButton>
								<Sidebar.MenuAction
									showOnHover
									onclick={(e) => {
										e.stopPropagation();
										deleteDraft(draft.id);
									}}
									class="text-sidebar-foreground/40 hover:text-destructive"
								>
									<TrashIcon class="size-3" />
								</Sidebar.MenuAction>
							</Sidebar.MenuItem>
						{/each}
					</Sidebar.Menu>
				{/if}
			</Sidebar.GroupContent>
		</Sidebar.Group>
	</Sidebar.Content>

	<Sidebar.Footer>
		<Sidebar.Separator />
		<!-- User Menu -->
		<Sidebar.Menu>
			<Sidebar.MenuItem>
				<DropdownMenu.Root>
					<DropdownMenu.Trigger>
						{#snippet child({ props })}
							<Sidebar.MenuButton
								{...props}
								size="lg"
								class="data-[state=open]:bg-sidebar-accent data-[state=open]:text-sidebar-accent-foreground"
							>
								<Avatar.Root class="size-8 rounded-lg">
									<Avatar.Fallback
										class="rounded-lg bg-sidebar-primary text-sidebar-primary-foreground"
									>
										{authState.user?.email?.charAt(0).toUpperCase() || 'U'}
									</Avatar.Fallback>
								</Avatar.Root>
								<div class="grid flex-1 text-start text-sm leading-tight">
									<span class="truncate font-medium text-sidebar-foreground"
										>{authState.user?.email?.split('@')[0] || 'User'}</span
									>
									<span class="truncate text-xs text-sidebar-foreground/70"
										>{authState.user?.email}</span
									>
								</div>
								<ChevronsUpDownIcon class="ms-auto size-4 text-sidebar-foreground" />
							</Sidebar.MenuButton>
						{/snippet}
					</DropdownMenu.Trigger>
					<DropdownMenu.Content
						class="w-56 rounded-lg"
						side={sidebar.isMobile ? 'bottom' : 'right'}
						align="start"
						sideOffset={4}
					>
						<DropdownMenu.Label class="p-0 font-normal">
							<div class="flex items-center gap-2 px-1 py-1.5 text-start text-sm">
								<Avatar.Root class="size-8 rounded-lg">
									<Avatar.Fallback class="rounded-lg bg-primary text-primary-foreground">
										{authState.user?.email?.charAt(0).toUpperCase() || 'U'}
									</Avatar.Fallback>
								</Avatar.Root>
								<div class="grid flex-1 text-start text-sm leading-tight">
									<span class="truncate font-medium"
										>{authState.user?.email?.split('@')[0] || 'User'}</span
									>
									<span class="truncate text-xs text-muted-foreground">{authState.user?.email}</span
									>
								</div>
							</div>
						</DropdownMenu.Label>
						<DropdownMenu.Separator />

						<!-- Navigation items moved here -->
						<DropdownMenu.Group>
							<DropdownMenu.Item onclick={() => goto('/accounts')}>
								<UsersIcon class="mr-2 size-4 text-muted-foreground" />
								<span>Accounts</span>
							</DropdownMenu.Item>
							<DropdownMenu.Item onclick={() => goto('/media')}>
								<ImageIcon class="mr-2 size-4 text-muted-foreground" />
								<span>Media</span>
							</DropdownMenu.Item>
							<DropdownMenu.Item onclick={() => goto('/prompts')}>
								<LightbulbIcon class="mr-2 size-4 text-muted-foreground" />
								<span>Prompts</span>
							</DropdownMenu.Item>
							<DropdownMenu.Item onclick={() => goto('/settings')}>
								<SettingsIcon class="mr-2 size-4 text-muted-foreground" />
								<span>Settings</span>
							</DropdownMenu.Item>
							<DropdownMenu.Item onclick={() => goto('/activity')}>
								<ScrollTextIcon class="mr-2 size-4 text-muted-foreground" />
								<span>Logs</span>
							</DropdownMenu.Item>
						</DropdownMenu.Group>

						<DropdownMenu.Separator />

						<DropdownMenu.Group>
							<DropdownMenu.Item onclick={toggleMode}>
								<SunIcon
									class="mr-2 size-4 scale-100 rotate-0 transition-all dark:scale-0 dark:-rotate-90"
								/>
								<MoonIcon
									class="absolute mr-2 size-4 scale-0 rotate-90 transition-all dark:scale-100 dark:rotate-0"
								/>
								<span>Toggle theme</span>
							</DropdownMenu.Item>
						</DropdownMenu.Group>

						<DropdownMenu.Separator />

						{#if IS_CAPACITOR}
							<DropdownMenu.Item onclick={handleSwitchServer}>
								<ServerIcon class="mr-2 text-muted-foreground" />
								<span>Change server</span>
							</DropdownMenu.Item>
							<DropdownMenu.Separator />
						{/if}

						<DropdownMenu.Item onclick={handleLogout}>
							<LogOutIcon class="mr-2 text-muted-foreground" />
							<span>Log out</span>
						</DropdownMenu.Item>
					</DropdownMenu.Content>
				</DropdownMenu.Root>
			</Sidebar.MenuItem>
		</Sidebar.Menu>
	</Sidebar.Footer>
	<Sidebar.Rail />
</Sidebar.Root>

<DayPostsModal />
