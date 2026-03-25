<script lang="ts">
	import { onMount } from 'svelte';
	import * as Sidebar from '$lib/components/ui/sidebar';
	import * as CalendarUi from '$lib/components/ui/calendar';
	import { Button } from '$lib/components/ui/button';
	import { client, type ScheduleOverview } from '$lib/api/client';
	import type { DateValue } from '@internationalized/date';
	import { getLocalTimeZone, today } from '@internationalized/date';
	import PlusIcon from 'lucide-svelte/icons/plus';
	import CircleDotIcon from 'lucide-svelte/icons/circle-dot';

	type DayMarkerArgs = {
		day: DateValue;
		outsideMonth: boolean;
	};

	let selectedDate = $state<DateValue>(today(getLocalTimeZone()));
	let selectedWorkspaceId = $state('');
	let selectedPlatform = $state('all');
	let overview = $state<ScheduleOverview | null>(null);
	let loading = $state(true);

	const monthString = $derived.by(() => {
		const jsDate = selectedDate.toDate(getLocalTimeZone());
		const year = jsDate.getFullYear();
		const month = String(jsDate.getMonth() + 1).padStart(2, '0');
		return `${year}-${month}`;
	});

	const dayCounts = $derived.by(() => {
		const map = new Map<string, number>();
		if (!overview) {
			return map;
		}

		for (const item of overview?.days ?? []) {
			map.set(item.date, item.count);
		}

		return map;
	});

	const dayPlatformCounts = $derived.by(() => {
		const map = new Map<string, { platform: string; count: number }[]>();
		if (!overview) {
			return map;
		}

		for (const item of overview?.days ?? []) {
			map.set(item.date, item.platforms || []);
		}

		return map;
	});

	const totalScheduled = $derived.by(() => {
		if (!overview) {
			return 0;
		}
		return (overview?.days ?? []).reduce((sum: number, day) => sum + day.count, 0);
	});

	const activeWorkspaceName = $derived.by(() => {
		if (!overview || !selectedWorkspaceId) {
			return 'All workspaces';
		}
		return (
			(overview?.workspaces ?? []).find((workspace) => workspace.id === selectedWorkspaceId)
				?.name || 'All workspaces'
		);
	});

	const platformLabel = (platform: string): string => {
		switch (platform) {
			case 'x':
				return 'X';
			case 'mastodon':
				return 'Mastodon';
			case 'threads':
				return 'Threads';
			case 'bluesky':
				return 'Bluesky';
			case 'linkedin':
				return 'LinkedIn';
			default:
				return platform;
		}
	};

	const platformDotClass = (platform: string): string => {
		switch (platform) {
			case 'x':
				return 'bg-zinc-800 dark:bg-zinc-100';
			case 'mastodon':
				return 'bg-indigo-500';
			case 'threads':
				return 'bg-amber-500';
			case 'bluesky':
				return 'bg-sky-500';
			case 'linkedin':
				return 'bg-blue-600';
			default:
				return 'bg-sidebar-primary';
		}
	};

	onMount(loadOverview);

	// Track previous month to detect actual changes
	let previousMonth = $state('');

	$effect(() => {
		const currentMonth = monthString;
		// Only reload if month actually changed and we've already loaded once
		if (previousMonth && previousMonth !== currentMonth) {
			loadOverview();
		}
		previousMonth = currentMonth;
	});

	async function loadOverview() {
		loading = true;
		try {
			const { data, error: err } = await client.GET('/posts/schedule-overview', {
				params: {
					query: {
						workspace_id: selectedWorkspaceId || undefined,
						platform: selectedPlatform === 'all' ? undefined : selectedPlatform,
						month: monthString
					}
				}
			});
			if (err || !data) throw new Error('Failed to load');
			overview = data;

			if (!selectedWorkspaceId) {
				selectedWorkspaceId = overview?.selected_workspace_id || '';
			}
		} catch {
			overview = null;
		} finally {
			loading = false;
		}
	}

	function selectWorkspace(workspaceId: string) {
		selectedWorkspaceId = workspaceId;
		loadOverview();
	}

	function selectPlatform(platform: string) {
		selectedPlatform = platform;
		loadOverview();
	}
</script>

{#snippet dayMarker({ day, outsideMonth }: DayMarkerArgs)}
	{@const key = day.toString()}
	{@const count = dayCounts.get(key) || 0}
	{@const platformCounts = dayPlatformCounts.get(key) || []}
	{@const dots = platformCounts.slice(0, 3)}
	<div class="relative flex size-(--cell-size) items-center justify-center">
		<CalendarUi.Day />
		{#if !outsideMonth && count > 0}
			<div class="pointer-events-none absolute bottom-0.5 flex items-center justify-center gap-0.5">
				{#if selectedPlatform === 'all'}
					{#each dots as marker (`${key}-${marker.platform}`)}
						<span class={`h-1.5 w-1.5 rounded-full ${platformDotClass(marker.platform)}`}></span>
					{/each}
				{:else}
					<span class={`h-1.5 w-1.5 rounded-full ${platformDotClass(selectedPlatform)}`}></span>
				{/if}
			</div>
		{/if}
	</div>
{/snippet}

<Sidebar.Root collapsible="none" class="sticky top-0 hidden h-svh border-s lg:flex">
	<Sidebar.Header class="h-16 border-b border-sidebar-border">
		<div class="flex h-full items-center px-4">
			<span class="font-semibold text-sidebar-foreground">Schedule</span>
		</div>
	</Sidebar.Header>
	<Sidebar.Content>
		<Sidebar.Group class="px-0 pt-2">
			<Sidebar.GroupContent>
				<CalendarUi.Calendar
					type="single"
					readonly
					bind:value={selectedDate}
					day={dayMarker}
					class="select-none [--cell-size:--spacing(9)] [&_[data-bits-calendar-head-cell]]:w-[33px] [&_[role=gridcell]]:w-[33px] [&_[role=gridcell]_[role=button][data-today]]:bg-sidebar-primary [&_[role=gridcell]_[role=button][data-today]]:text-sidebar-primary-foreground"
				/>
			</Sidebar.GroupContent>
		</Sidebar.Group>

		<Sidebar.Separator class="mx-0" />

		<Sidebar.Group class="py-0">
			<Sidebar.GroupLabel class="text-sidebar-foreground/70">Workspace</Sidebar.GroupLabel>
			<Sidebar.GroupContent>
				<Sidebar.Menu>
					<Sidebar.MenuItem>
						<Sidebar.MenuButton
							isActive={selectedWorkspaceId === ''}
							onclick={() => selectWorkspace('')}
						>
							All workspaces
						</Sidebar.MenuButton>
					</Sidebar.MenuItem>
					{#each overview?.workspaces || [] as workspace (workspace.id)}
						<Sidebar.MenuItem>
							<Sidebar.MenuButton
								isActive={selectedWorkspaceId === workspace.id}
								onclick={() => selectWorkspace(workspace.id)}
							>
								{workspace.name}
							</Sidebar.MenuButton>
						</Sidebar.MenuItem>
					{/each}
				</Sidebar.Menu>
			</Sidebar.GroupContent>
		</Sidebar.Group>

		<Sidebar.Separator class="mx-0" />

		<Sidebar.Group class="py-0">
			<Sidebar.GroupLabel class="text-sidebar-foreground/70">Social Media</Sidebar.GroupLabel>
			<Sidebar.GroupContent>
				<Sidebar.Menu>
					<Sidebar.MenuItem>
						<Sidebar.MenuButton
							isActive={selectedPlatform === 'all'}
							onclick={() => selectPlatform('all')}
						>
							All platforms
						</Sidebar.MenuButton>
					</Sidebar.MenuItem>
					{#each overview?.platforms || [] as platform (platform)}
						<Sidebar.MenuItem>
							<Sidebar.MenuButton
								isActive={selectedPlatform === platform}
								onclick={() => selectPlatform(platform)}
							>
								{platformLabel(platform)}
							</Sidebar.MenuButton>
						</Sidebar.MenuItem>
					{/each}
				</Sidebar.Menu>
			</Sidebar.GroupContent>
		</Sidebar.Group>

		<Sidebar.Separator class="mx-0" />

		<Sidebar.Group>
			<Sidebar.GroupLabel class="text-sidebar-foreground/70">Overview</Sidebar.GroupLabel>
			<Sidebar.GroupContent>
				<Sidebar.Menu>
					<Sidebar.MenuItem>
						<Sidebar.MenuButton class="text-sidebar-foreground/80">
							<CircleDotIcon class="size-3.5" />
							<span
								>{loading ? 'Loading schedule...' : `${totalScheduled} scheduled this month`}</span
							>
						</Sidebar.MenuButton>
					</Sidebar.MenuItem>
					<Sidebar.MenuItem>
						<Sidebar.MenuButton class="text-sidebar-foreground/80">
							<span class="truncate">{activeWorkspaceName}</span>
						</Sidebar.MenuButton>
					</Sidebar.MenuItem>
				</Sidebar.Menu>
			</Sidebar.GroupContent>
		</Sidebar.Group>
	</Sidebar.Content>
	<Sidebar.Footer>
		<Sidebar.Menu>
			<Sidebar.MenuItem>
				<Button
					class="w-full justify-start"
					href={selectedWorkspaceId ? `/workspace/${selectedWorkspaceId}/compose` : '/'}
				>
					<PlusIcon class="size-4" />
					<span>New Post</span>
				</Button>
			</Sidebar.MenuItem>
		</Sidebar.Menu>
	</Sidebar.Footer>
</Sidebar.Root>
