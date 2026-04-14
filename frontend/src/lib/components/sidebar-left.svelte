<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/stores';
	import { client, type ScheduleOverview } from '$lib/api/client';
	import * as Sidebar from '$lib/components/ui/sidebar';
	import * as DropdownMenu from '$lib/components/ui/dropdown-menu';
	import * as Avatar from '$lib/components/ui/avatar';
	import * as CalendarUi from '$lib/components/ui/calendar';
	import { Button } from '$lib/components/ui/button';
	import Logo from './Logo.svelte';
	import ComposeModal from './compose-modal.svelte';
	import DayPostsModal from './day-posts-modal.svelte';
	import HouseIcon from 'lucide-svelte/icons/home';
	import UsersIcon from 'lucide-svelte/icons/users';
	import ImageIcon from 'lucide-svelte/icons/image';
	import SettingsIcon from 'lucide-svelte/icons/settings';
	import PlusIcon from 'lucide-svelte/icons/plus';
	import LogOutIcon from 'lucide-svelte/icons/log-out';
	import ChevronsUpDownIcon from 'lucide-svelte/icons/chevrons-up-down';
	import CircleDotIcon from 'lucide-svelte/icons/circle-dot';
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

	let authState = $derived($auth);
	const sidebar = Sidebar.useSidebar();
	let pathname = $derived($page.url.pathname);

	// Calendar state
	let selectedDate = $state<DateValue | undefined>(undefined);
	let calendarPlaceholder = $state<DateValue>(today(getLocalTimeZone()));
	let overview = $state<ScheduleOverview | null>(null);
	let loadingSchedule = $state(true);

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

	const navItems = [
		{ title: 'Dashboard', url: '/', icon: HouseIcon, isActive: () => pathname === '/' },
		{
			title: 'Accounts',
			url: '/accounts',
			icon: UsersIcon,
			isActive: () => pathname.startsWith('/accounts')
		},
		{
			title: 'Media',
			url: '/media',
			icon: ImageIcon,
			isActive: () => pathname.startsWith('/media')
		},
		{
			title: 'Settings',
			url: '/settings',
			icon: SettingsIcon,
			isActive: () => pathname.startsWith('/settings')
		}
	];

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
	});

	// Track previous month to detect actual changes
	let previousMonth = $state('');

	$effect(() => {
		const currentMonth = monthString;
		if (previousMonth && previousMonth !== currentMonth) {
			loadOverview();
		}
		previousMonth = currentMonth;
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

	async function loadOverview() {
		loadingSchedule = true;
		try {
			const { data, error: err } = await client.GET('/posts/schedule-overview', {
				params: {
					query: {
						month: monthString
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

		<!-- New Post Button -->
		<div class="px-2 pb-2">
			<Button class="w-full justify-start gap-2" onclick={() => goto('/posts/new')}>
				<PlusIcon class="size-4" />
				<span>New Post</span>
			</Button>
		</div>

		<Sidebar.Separator />

		<!-- Main Navigation -->
		<Sidebar.Menu>
			{#each navItems as item (item.title)}
				<Sidebar.MenuItem>
					<Sidebar.MenuButton isActive={item.isActive()}>
						{#snippet child({ props })}
							<a href={item.url} {...props}>
								<item.icon class="text-sidebar-foreground" />
								<span class="text-sidebar-foreground">{item.title}</span>
							</a>
						{/snippet}
					</Sidebar.MenuButton>
				</Sidebar.MenuItem>
			{/each}
		</Sidebar.Menu>
	</Sidebar.Header>

	<Sidebar.Content>
		<Sidebar.Separator />
		<!-- Calendar Section -->
		<Sidebar.Group class="px-0 pt-4">
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
										? 'Loading...'
										: `${overview.days.reduce((s: number, d: { count: number }) => s + d.count, 0)} scheduled posts`}</span
								>
							</Sidebar.MenuButton>
						</Sidebar.MenuItem>
					</Sidebar.Menu>
				</Sidebar.GroupContent>
			</Sidebar.Group>
		{/if}
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

<ComposeModal onSuccess={loadOverview} />
<DayPostsModal />
