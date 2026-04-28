<script lang="ts">
	import { onMount } from 'svelte';
	import { workspaceCtx } from '$lib/stores/workspace.svelte';
	import { Button } from '$lib/components/ui/button';
	import * as Select from '$lib/components/ui/select';
	import PageContainer from '$lib/components/page-container.svelte';
	import LoaderIcon from 'lucide-svelte/icons/loader-2';
	import SettingsIcon from 'lucide-svelte/icons/settings';
	import SaveIcon from 'lucide-svelte/icons/save';
	import XIcon from 'lucide-svelte/icons/x';
	import ClockIcon from 'lucide-svelte/icons/clock';
	import ImageIcon from 'lucide-svelte/icons/image';
	import CalendarIcon from 'lucide-svelte/icons/calendar';
	import PlusIcon from 'lucide-svelte/icons/plus';
	import TrashIcon from 'lucide-svelte/icons/trash';
	import { Skeleton } from '$lib/components/ui/skeleton/index.js';
	import { client } from '$lib/api/client';

	const timezones = [
		{ group: 'Americas', value: 'America/New_York', label: 'New York (ET)' },
		{ group: 'Americas', value: 'America/Chicago', label: 'Chicago (CT)' },
		{ group: 'Americas', value: 'America/Denver', label: 'Denver (MT)' },
		{ group: 'Americas', value: 'America/Los_Angeles', label: 'Los Angeles (PT)' },
		{ group: 'Americas', value: 'America/Phoenix', label: 'Phoenix (AZ)' },
		{ group: 'Americas', value: 'America/Anchorage', label: 'Anchorage (AK)' },
		{ group: 'Americas', value: 'Pacific/Honolulu', label: 'Honolulu (HI)' },
		{ group: 'Americas', value: 'America/Toronto', label: 'Toronto (ET)' },
		{ group: 'Americas', value: 'America/Vancouver', label: 'Vancouver (PT)' },
		{ group: 'Americas', value: 'America/Mexico_City', label: 'Mexico City (CT)' },
		{ group: 'Americas', value: 'America/Bogota', label: 'Bogota' },
		{ group: 'Americas', value: 'America/Lima', label: 'Lima' },
		{ group: 'Americas', value: 'America/Santiago', label: 'Santiago' },
		{ group: 'Americas', value: 'America/Sao_Paulo', label: 'Sao Paulo' },
		{ group: 'Americas', value: 'America/Buenos_Aires', label: 'Buenos Aires' },
		{ group: 'Europe', value: 'UTC', label: 'UTC' },
		{ group: 'Europe', value: 'Europe/London', label: 'London (GMT/BST)' },
		{ group: 'Europe', value: 'Europe/Dublin', label: 'Dublin (GMT/IST)' },
		{ group: 'Europe', value: 'Europe/Lisbon', label: 'Lisbon (WET/WEST)' },
		{ group: 'Europe', value: 'Europe/Madrid', label: 'Madrid (CET/CEST)' },
		{ group: 'Europe', value: 'Europe/Paris', label: 'Paris (CET/CEST)' },
		{ group: 'Europe', value: 'Europe/Amsterdam', label: 'Amsterdam (CET/CEST)' },
		{ group: 'Europe', value: 'Europe/Brussels', label: 'Brussels (CET/CEST)' },
		{ group: 'Europe', value: 'Europe/Berlin', label: 'Berlin (CET/CEST)' },
		{ group: 'Europe', value: 'Europe/Vienna', label: 'Vienna (CET/CEST)' },
		{ group: 'Europe', value: 'Europe/Zurich', label: 'Zurich (CET/CEST)' },
		{ group: 'Europe', value: 'Europe/Rome', label: 'Rome (CET/CEST)' },
		{ group: 'Europe', value: 'Europe/Stockholm', label: 'Stockholm (CET/CEST)' },
		{ group: 'Europe', value: 'Europe/Oslo', label: 'Oslo (CET/CEST)' },
		{ group: 'Europe', value: 'Europe/Copenhagen', label: 'Copenhagen (CET/CEST)' },
		{ group: 'Europe', value: 'Europe/Helsinki', label: 'Helsinki (EET/EEST)' },
		{ group: 'Europe', value: 'Europe/Warsaw', label: 'Warsaw (CET/CEST)' },
		{ group: 'Europe', value: 'Europe/Prague', label: 'Prague (CET/CEST)' },
		{ group: 'Europe', value: 'Europe/Budapest', label: 'Budapest (CET/CEST)' },
		{ group: 'Europe', value: 'Europe/Athens', label: 'Athens (EET/EEST)' },
		{ group: 'Europe', value: 'Europe/Bucharest', label: 'Bucharest (EET/EEST)' },
		{ group: 'Europe', value: 'Europe/Kiev', label: 'Kiev (EET/EEST)' },
		{ group: 'Europe', value: 'Europe/Moscow', label: 'Moscow (MSK)' },
		{ group: 'Europe', value: 'Europe/Istanbul', label: 'Istanbul (TRT)' },
		{ group: 'Asia', value: 'Asia/Dubai', label: 'Dubai (GST)' },
		{ group: 'Asia', value: 'Asia/Riyadh', label: 'Riyadh (AST)' },
		{ group: 'Asia', value: 'Asia/Tehran', label: 'Tehran (IRST/IRDT)' },
		{ group: 'Asia', value: 'Asia/Kolkata', label: 'Mumbai/Delhi (IST)' },
		{ group: 'Asia', value: 'Asia/Bangkok', label: 'Bangkok (ICT)' },
		{ group: 'Asia', value: 'Asia/Jakarta', label: 'Jakarta (WIB)' },
		{ group: 'Asia', value: 'Asia/Singapore', label: 'Singapore (SGT)' },
		{ group: 'Asia', value: 'Asia/Hong_Kong', label: 'Hong Kong (HKT)' },
		{ group: 'Asia', value: 'Asia/Shanghai', label: 'Shanghai (CST)' },
		{ group: 'Asia', value: 'Asia/Tokyo', label: 'Tokyo (JST)' },
		{ group: 'Asia', value: 'Asia/Seoul', label: 'Seoul (KST)' },
		{ group: 'Asia', value: 'Asia/Manila', label: 'Manila (PHT)' },
		{ group: 'Asia', value: 'Asia/Kuala_Lumpur', label: 'Kuala Lumpur (MYT)' },
		{ group: 'Pacific', value: 'Australia/Perth', label: 'Perth (AWST)' },
		{ group: 'Pacific', value: 'Australia/Eucla', label: 'Eucla (AWST+)' },
		{ group: 'Pacific', value: 'Australia/Adelaide', label: 'Adelaide (ACST)' },
		{ group: 'Pacific', value: 'Australia/Brisbane', label: 'Brisbane (AEST)' },
		{ group: 'Pacific', value: 'Australia/Sydney', label: 'Sydney (AEST/AEDT)' },
		{ group: 'Pacific', value: 'Pacific/Auckland', label: 'Auckland (NZST/NZDT)' },
		{ group: 'Pacific', value: 'Pacific/Fiji', label: 'Fiji (FJT/FJST)' },
		{ group: 'Africa', value: 'Africa/Cairo', label: 'Cairo (EET)' },
		{ group: 'Africa', value: 'Africa/Johannesburg', label: 'Johannesburg (SAST)' },
		{ group: 'Africa', value: 'Africa/Lagos', label: 'Lagos (WAT)' },
		{ group: 'Africa', value: 'Africa/Nairobi', label: 'Nairobi (EAT)' }
	];

	const groupedTimezones = $derived(() => {
		const groups: Record<string, typeof timezones> = {};
		for (const tz of timezones) {
			if (!groups[tz.group]) groups[tz.group] = [];
			groups[tz.group].push(tz);
		}
		return groups;
	});

	function getTimezoneLabel(value: string): string {
		const tz = timezones.find((t) => t.value === value);
		return tz?.label ?? value;
	}

	let detectedTimezone = $state('');

	onMount(() => {
		detectedTimezone = Intl.DateTimeFormat().resolvedOptions().timeZone;
		if (!workspaceCtx.settings.timezone && detectedTimezone) {
			workspaceCtx.settings.timezone = detectedTimezone;
		}
	});

	const cleanupDaysOptions = [
		{ value: 0, label: 'Disabled' },
		{ value: 7, label: '7 days' },
		{ value: 14, label: '14 days' },
		{ value: 30, label: '30 days' },
		{ value: 60, label: '60 days' },
		{ value: 90, label: '90 days' },
		{ value: 180, label: '180 days' },
		{ value: 365, label: '1 year' }
	];

	let saving = $state(false);
	let toastMessage = $state('');

	async function saveSettings() {
		saving = true;
		try {
			await workspaceCtx.saveSettings({
				timezone: workspaceCtx.settings.timezone,
				week_start: workspaceCtx.settings.week_start,
				media_cleanup_days: workspaceCtx.settings.media_cleanup_days,
				random_delay_minutes: workspaceCtx.settings.random_delay_minutes,
				slot_start_hour: workspaceCtx.settings.slot_start_hour,
				slot_end_hour: workspaceCtx.settings.slot_end_hour,
				slot_interval_minutes: workspaceCtx.settings.slot_interval_minutes
			});
			toastMessage = 'Settings saved successfully';
		} catch (e) {
			toastMessage = (e as Error).message;
		} finally {
			saving = false;
		}
	}

	function parseDurationInput(input: string): number | null {
		input = input.trim().toLowerCase();
		// Try direct number first (assume minutes)
		const direct = parseInt(input, 10);
		if (!isNaN(direct) && direct > 0 && String(direct) === input) {
			return direct;
		}
		// Parse patterns like "15m", "30 min", "1h", "1h30m", "90 minutes", "2 hours"
		const hourMatch = input.match(/(\d+)\s*h/);
		const minMatch = input.match(/(\d+)\s*m/);
		let total = 0;
		if (hourMatch) total += parseInt(hourMatch[1], 10) * 60;
		if (minMatch) total += parseInt(minMatch[1], 10);
		if (total > 0) return total;
		return null;
	}

	let intervalInput = $state(String(workspaceCtx.settings.slot_interval_minutes));
	let intervalError = $state('');

	function handleIntervalChange(value: string) {
		intervalInput = value;
		const parsed = parseDurationInput(value);
		if (parsed !== null && parsed >= 1 && parsed <= 180) {
			intervalError = '';
			workspaceCtx.settings.slot_interval_minutes = parsed;
		} else if (value.trim() !== '') {
			intervalError = 'Enter a duration between 1 minute and 3 hours (e.g. 15m, 1h, 30)';
		}
	}

	// Posting schedules
	interface PostingSchedule {
		id: string;
		workspace_id: string;
		set_id: string;
		utc_hour: number;
		utc_minute: number;
		day_of_week: number;
		label: string;
		is_active: boolean;
		created_at: string;
	}

	let schedules = $state<PostingSchedule[]>([]);
	let loadingSchedules = $state(false);
	let showAddSchedule = $state(false);
	let showSuggestSchedule = $state(false);
	let suggestedPostsPerDay = $state(3);
	let generatingSchedule = $state(false);
	let newSchedule = $state({
		day_of_week: 1,
		utc_hour: 9,
		utc_minute: 0,
		label: ''
	});

	const dayNames = ['Sunday', 'Monday', 'Tuesday', 'Wednesday', 'Thursday', 'Friday', 'Saturday'];

	async function loadSchedules() {
		if (!workspaceCtx.currentWorkspace) return;
		loadingSchedules = true;
		try {
			const { data, error: err } = await (client as any).GET('/posting-schedules', {
				params: { query: { workspace_id: workspaceCtx.currentWorkspace.id } }
			});
			if (!err && data) {
				schedules = data;
			}
		} catch (e) {
			console.error('Failed to load schedules:', e);
		} finally {
			loadingSchedules = false;
		}
	}

	async function addSchedule() {
		if (!workspaceCtx.currentWorkspace) return;
		try {
			const { error: err } = await (client as any).POST('/posting-schedules', {
				body: {
					workspace_id: workspaceCtx.currentWorkspace.id,
					day_of_week: newSchedule.day_of_week,
					utc_hour: newSchedule.utc_hour,
					utc_minute: newSchedule.utc_minute,
					label: newSchedule.label
				}
			});
			if (err) throw err;
			showAddSchedule = false;
			newSchedule = { day_of_week: 1, utc_hour: 9, utc_minute: 0, label: '' };
			await loadSchedules();
			toastMessage = 'Schedule added successfully';
		} catch (e) {
			toastMessage = (e as Error).message || 'Failed to add schedule';
		}
	}

	async function deleteSchedule(id: string) {
		try {
			const { error: err } = await (client as any).DELETE('/posting-schedules/{id}', {
				params: { path: { id } }
			});
			if (err) throw err;
			await loadSchedules();
			toastMessage = 'Schedule deleted successfully';
		} catch (e) {
			toastMessage = (e as Error).message || 'Failed to delete schedule';
		}
	}

	async function generateSuggestedSchedule() {
		if (!workspaceCtx.currentWorkspace) return;
		generatingSchedule = true;
		try {
			const { error: err } = await (client as any).POST('/posting-schedules/suggest', {
				body: {
					workspace_id: workspaceCtx.currentWorkspace.id,
					posts_per_day: suggestedPostsPerDay
				}
			});
			if (err) throw err;
			showSuggestSchedule = false;
			await loadSchedules();
			toastMessage = `Generated suggested schedule with ${suggestedPostsPerDay} posts per day`;
		} catch (e) {
			toastMessage = (e as Error).message || 'Failed to generate schedule';
		} finally {
			generatingSchedule = false;
		}
	}

	function formatTime(hour: number, minute: number): string {
		const h = hour.toString().padStart(2, '0');
		const m = minute.toString().padStart(2, '0');
		return `${h}:${m}`;
	}

	function formatLocalTime(hour: number, minute: number): string {
		// Create a date in UTC with the given time
		const utcDate = new Date();
		utcDate.setUTCHours(hour, minute, 0, 0);
		// Format in local timezone
		return utcDate.toLocaleTimeString('en-US', {
			hour: '2-digit',
			minute: '2-digit',
			hour12: false
		});
	}

	$effect(() => {
		if (workspaceCtx.currentWorkspace) {
			loadSchedules();
		}
	});

	function handleTimezoneChange(value: string) {
		workspaceCtx.settings.timezone = value;
	}

	function handleWeekStartChange(value: number) {
		workspaceCtx.settings.week_start = value;
	}

	function handleCleanupDaysChange(value: number) {
		workspaceCtx.settings.media_cleanup_days = value;
	}
</script>

<svelte:head>
	<title>Settings - OpenPost</title>
</svelte:head>

{#if toastMessage}
	<div
		class="pointer-events-auto fixed right-4 bottom-4 z-50 mb-4 flex items-center gap-2 rounded-lg border bg-background px-4 py-3 shadow-lg"
	>
		<span class="text-sm">{toastMessage}</span>
		<button onclick={() => (toastMessage = '')}>
			<XIcon class="size-4" />
		</button>
	</div>
{/if}

<PageContainer
	title="Settings"
	description="Manage your workspace preferences"
	icon={SettingsIcon}
	loading={!workspaceCtx.currentWorkspace}
	loadingMessage="Loading workspace..."
>
	<div class="space-y-8">
		<!-- Workspace Info -->
		<section class="space-y-4">
			<h2 class="mb-4 text-lg font-semibold">Workspace</h2>
			<div class="space-y-4">
				<div class="flex items-center gap-4">
					<span class="text-sm font-medium">Current Workspace</span>
					<span class="text-sm text-muted-foreground">{workspaceCtx.currentWorkspace?.name}</span>
				</div>
			</div>
		</section>

		<!-- Date & Time Settings -->
		<section class="space-y-4">
			<h2 class="mb-4 flex items-center gap-2 text-lg font-semibold">
				<ClockIcon class="h-5 w-5 text-muted-foreground" />
				Date & Time
			</h2>
			<div class="space-y-4">
				<div class="grid gap-4 sm:grid-cols-2">
					<div class="space-y-2">
						<label class="text-sm font-medium" for="timezone-select">Timezone</label>
						<Select.Root
							type="single"
							value={workspaceCtx.settings.timezone}
							onValueChange={handleTimezoneChange}
						>
							<Select.Trigger id="timezone-select" class="w-full">
								{getTimezoneLabel(workspaceCtx.settings.timezone)}
							</Select.Trigger>
							<Select.Content class="max-h-80 overflow-y-auto">
								{#each Object.entries(groupedTimezones()) as [group, tzs]}
									<Select.Group>
										<Select.GroupHeading class="text-xs">{group}</Select.GroupHeading>
										{#each tzs as tz}
											<Select.Item value={tz.value}>{tz.label}</Select.Item>
										{/each}
									</Select.Group>
								{/each}
							</Select.Content>
						</Select.Root>
						<p class="text-sm text-muted-foreground">Used for displaying scheduled post times</p>
					</div>

					<div class="space-y-2">
						<label class="text-sm font-medium" for="week-start-select">Week Starts On</label>
						<Select.Root
							type="single"
							value={String(workspaceCtx.settings.week_start)}
							onValueChange={(v) => handleWeekStartChange(Number(v))}
						>
							<Select.Trigger id="week-start-select" class="w-full">
								{workspaceCtx.settings.week_start === 0 ? 'Sunday' : 'Monday'}
							</Select.Trigger>
							<Select.Content>
								<Select.Item value="0">Sunday</Select.Item>
								<Select.Item value="1">Monday</Select.Item>
							</Select.Content>
						</Select.Root>
						<p class="text-sm text-muted-foreground">Affects the calendar display in the sidebar</p>
					</div>
				</div>
			</div>
		</section>

		<!-- Media Cleanup Settings -->
		<section class="space-y-4">
			<h2 class="mb-4 flex items-center gap-2 text-lg font-semibold">
				<ImageIcon class="h-5 w-5 text-muted-foreground" />
				Media Cleanup
			</h2>
			<div class="space-y-4">
				<div class="space-y-2">
					<label class="text-sm font-medium" for="cleanup-select">Auto-delete unused media</label>
					<Select.Root
						type="single"
						value={String(workspaceCtx.settings.media_cleanup_days)}
						onValueChange={(v) => handleCleanupDaysChange(Number(v))}
					>
						<Select.Trigger id="cleanup-select" class="w-full">
							{cleanupDaysOptions.find((o) => o.value === workspaceCtx.settings.media_cleanup_days)
								?.label || 'Disabled'}
						</Select.Trigger>
						<Select.Content>
							{#each cleanupDaysOptions as option}
								<Select.Item value={String(option.value)}>{option.label}</Select.Item>
							{/each}
						</Select.Content>
					</Select.Root>
					<p class="text-sm text-muted-foreground">
						Automatically delete unused, non-favorited media after this period. Favorited media is
						always kept.
					</p>
				</div>
			</div>
		</section>

		<!-- Posting Schedule Settings -->
		<section class="rounded-lg border p-6">
			<div class="mb-4 flex items-center justify-between">
				<h2 class="flex items-center gap-2 text-lg font-semibold">
					<CalendarIcon class="h-5 w-5 text-muted-foreground" />
					Posting Schedule
				</h2>
				<Button onclick={() => (showAddSchedule = true)} variant="outline" size="sm">
					<PlusIcon class="mr-2 h-4 w-4" />
					Add Time Slot
				</Button>
			</div>
			<p class="mb-4 text-sm text-muted-foreground">
				Define your preferred posting times. The "Suggest Time" button in the compose page will use
				these slots.
			</p>

			{#if loadingSchedules}
				<div class="space-y-2">
					<Skeleton class="h-14 rounded-md" />
					<Skeleton class="h-14 rounded-md" />
					<Skeleton class="h-14 rounded-md" />
				</div>
			{:else if schedules.length === 0}
				<div class="rounded-md border border-dashed p-8 text-center text-muted-foreground">
					<p class="text-sm">No posting schedules configured.</p>
					<p class="mt-1 text-xs">Add time slots to enable the "Suggest Time" feature.</p>
					{#if !showSuggestSchedule}
						<Button
							onclick={() => (showSuggestSchedule = true)}
							variant="outline"
							size="sm"
							class="mt-4"
						>
							Use suggested schedule
						</Button>
					{:else}
						<div class="mt-4 flex flex-col items-center gap-3">
							<div class="flex items-center gap-2">
								<label class="text-sm" for="posts-per-day">Posts per day</label>
								<Select.Root
									type="single"
									value={String(suggestedPostsPerDay)}
									onValueChange={(v) => (suggestedPostsPerDay = Number(v))}
								>
									<Select.Trigger id="posts-per-day" class="w-24">
										{suggestedPostsPerDay}
									</Select.Trigger>
									<Select.Content class="max-h-60 overflow-y-auto">
										{#each Array.from({ length: 10 }, (_, i) => i + 1) as n}
											<Select.Item value={String(n)}>{n}</Select.Item>
										{/each}
									</Select.Content>
								</Select.Root>
							</div>
							<div class="flex gap-2">
								<Button onclick={() => (showSuggestSchedule = false)} variant="outline" size="sm">
									Cancel
								</Button>
								<Button onclick={generateSuggestedSchedule} size="sm" disabled={generatingSchedule}>
									{#if generatingSchedule}
										<LoaderIcon class="mr-2 h-4 w-4 animate-spin" />
									{/if}
									Generate Schedule
								</Button>
							</div>
						</div>
					{/if}
				</div>
			{:else}
				<div class="space-y-2">
					{#each dayNames as dayName, dayIndex}
						{@const daySchedules = schedules.filter((s) => s.day_of_week === dayIndex)}
						{#if daySchedules.length > 0}
							<div class="rounded-md border p-3">
								<div class="mb-2 text-sm font-medium">{dayName}</div>
								<div class="flex flex-wrap gap-2">
									{#each daySchedules as schedule}
										<div class="flex items-center gap-2 rounded-md bg-muted px-3 py-1.5 text-sm">
											<span class="font-medium">
												{formatLocalTime(schedule.utc_hour, schedule.utc_minute)}
											</span>
											{#if schedule.label}
												<span class="text-xs text-muted-foreground">({schedule.label})</span>
											{/if}
											<button
												onclick={() => deleteSchedule(schedule.id)}
												class="ml-1 text-muted-foreground hover:text-destructive"
											>
												<TrashIcon class="h-3.5 w-3.5" />
											</button>
										</div>
									{/each}
								</div>
							</div>
						{/if}
					{/each}
				</div>
			{/if}

			{#if showAddSchedule}
				<div class="mt-4 rounded-md border bg-muted/30 p-4">
					<h3 class="mb-3 text-sm font-medium">Add New Time Slot</h3>
					<div class="grid gap-4 sm:grid-cols-4">
						<div class="space-y-2">
							<label class="text-xs" for="schedule-day">Day</label>
							<Select.Root
								type="single"
								value={String(newSchedule.day_of_week)}
								onValueChange={(v) => (newSchedule.day_of_week = Number(v))}
							>
								<Select.Trigger id="schedule-day" class="w-full">
									{dayNames[newSchedule.day_of_week]}
								</Select.Trigger>
								<Select.Content>
									{#each dayNames as name, idx}
										<Select.Item value={String(idx)}>{name}</Select.Item>
									{/each}
								</Select.Content>
							</Select.Root>
						</div>
						<div class="space-y-2">
							<label class="text-xs" for="schedule-hour">Hour (UTC)</label>
							<Select.Root
								type="single"
								value={String(newSchedule.utc_hour)}
								onValueChange={(v) => (newSchedule.utc_hour = Number(v))}
							>
								<Select.Trigger id="schedule-hour" class="w-full">
									{newSchedule.utc_hour.toString().padStart(2, '0')}:00
								</Select.Trigger>
								<Select.Content class="max-h-60 overflow-y-auto">
									{#each Array.from({ length: 24 }, (_, i) => i) as hour}
										<Select.Item value={String(hour)}>
											{hour.toString().padStart(2, '0')}:00
										</Select.Item>
									{/each}
								</Select.Content>
							</Select.Root>
						</div>
						<div class="space-y-2">
							<label class="text-xs" for="schedule-minute">Minute</label>
							<Select.Root
								type="single"
								value={String(newSchedule.utc_minute)}
								onValueChange={(v) => (newSchedule.utc_minute = Number(v))}
							>
								<Select.Trigger id="schedule-minute" class="w-full">
									{newSchedule.utc_minute.toString().padStart(2, '0')}
								</Select.Trigger>
								<Select.Content>
									{#each [0, 15, 30, 45] as minute}
										<Select.Item value={String(minute)}>
											{minute.toString().padStart(2, '0')}
										</Select.Item>
									{/each}
								</Select.Content>
							</Select.Root>
						</div>
						<div class="space-y-2">
							<label class="text-xs" for="schedule-label">Label (optional)</label>
							<input
								id="schedule-label"
								type="text"
								bind:value={newSchedule.label}
								placeholder="e.g., Morning"
								class="h-9 w-full rounded-md border border-input bg-transparent px-3 text-sm"
							/>
						</div>
					</div>
					<div class="mt-4 flex justify-end gap-2">
						<Button onclick={() => (showAddSchedule = false)} variant="outline" size="sm">
							Cancel
						</Button>
						<Button onclick={addSchedule} size="sm">Add Slot</Button>
					</div>
				</div>
			{/if}
		</section>

		<!-- Natural Posting Settings -->
		<section class="space-y-4">
			<h2 class="mb-4 flex items-center gap-2 text-lg font-semibold">
				<ClockIcon class="h-5 w-5 text-muted-foreground" />
				Natural Posting
			</h2>
			<div class="space-y-4">
				<p class="text-sm text-muted-foreground">
					Add a small random delay to scheduled posts so they don't all go out at exactly the same
					minute. This makes your posting pattern look more natural.
				</p>
				<div class="space-y-2">
					<label class="text-sm font-medium" for="random-delay">Random delay range</label>
					<Select.Root
						type="single"
						value={String(workspaceCtx.settings.random_delay_minutes)}
						onValueChange={(v) => (workspaceCtx.settings.random_delay_minutes = Number(v))}
					>
						<Select.Trigger id="random-delay" class="w-full sm:w-64">
							{#if workspaceCtx.settings.random_delay_minutes === 0}
								No delay (exact time)
							{:else}
								±{workspaceCtx.settings.random_delay_minutes} minutes
							{/if}
						</Select.Trigger>
						<Select.Content>
							<Select.Item value="0">No delay (exact time)</Select.Item>
							<Select.Item value="5">±5 minutes</Select.Item>
							<Select.Item value="10">±10 minutes</Select.Item>
							<Select.Item value="15">±15 minutes</Select.Item>
							<Select.Item value="30">±30 minutes</Select.Item>
							<Select.Item value="45">±45 minutes</Select.Item>
							<Select.Item value="60">±1 hour</Select.Item>
						</Select.Content>
					</Select.Root>
				</div>
			</div>
		</section>

		<!-- Time Slot Configuration -->
		<section class="space-y-4">
			<h2 class="mb-4 flex items-center gap-2 text-lg font-semibold">
				<ClockIcon class="h-5 w-5 text-muted-foreground" />
				Time Slot Defaults
			</h2>
			<div class="space-y-4">
				<p class="text-sm text-muted-foreground">
					Configure the default time range and interval shown in the compose page scheduler.
				</p>
				<div class="grid gap-4 sm:grid-cols-3">
					<div class="space-y-2">
						<label class="text-sm font-medium" for="start-time">Start time</label>
						<Select.Root
							type="single"
							value={String(workspaceCtx.settings.slot_start_hour)}
							onValueChange={(v) => (workspaceCtx.settings.slot_start_hour = Number(v))}
						>
							<Select.Trigger id="start-time" class="w-full">
								{workspaceCtx.settings.slot_start_hour.toString().padStart(2, '0')}:00
							</Select.Trigger>
							<Select.Content class="max-h-60 overflow-y-auto">
								{#each Array.from({ length: 24 }, (_, i) => i) as hour}
									<Select.Item value={String(hour)}>
										{hour.toString().padStart(2, '0')}:00
									</Select.Item>
								{/each}
							</Select.Content>
						</Select.Root>
					</div>
					<div class="space-y-2">
						<label class="text-sm font-medium" for="end-time">End time</label>
						<Select.Root
							type="single"
							value={String(workspaceCtx.settings.slot_end_hour)}
							onValueChange={(v) => (workspaceCtx.settings.slot_end_hour = Number(v))}
						>
							<Select.Trigger id="end-time" class="w-full">
								{workspaceCtx.settings.slot_end_hour.toString().padStart(2, '0')}:00
							</Select.Trigger>
							<Select.Content class="max-h-60 overflow-y-auto">
								{#each Array.from({ length: 24 }, (_, i) => i) as hour}
									<Select.Item value={String(hour)}>
										{hour.toString().padStart(2, '0')}:00
									</Select.Item>
								{/each}
							</Select.Content>
						</Select.Root>
					</div>
					<div class="space-y-2">
						<label class="text-sm font-medium" for="interval">Interval</label>
						<input
							id="interval"
							type="text"
							value={intervalInput}
							oninput={(e) => handleIntervalChange((e.target as HTMLInputElement).value)}
							placeholder="e.g. 15m, 30 min, 1h"
							class="h-9 w-full rounded-md border border-input bg-transparent px-3 text-sm {intervalError
								? 'border-destructive'
								: ''}"
						/>
						{#if intervalError}
							<p class="text-xs text-destructive">{intervalError}</p>
						{:else}
							<p class="text-xs text-muted-foreground">
								Current: {workspaceCtx.settings.slot_interval_minutes} minutes
							</p>
						{/if}
					</div>
				</div>
			</div>
		</section>

		<!-- Save Button -->
		<div class="flex justify-end">
			<Button onclick={saveSettings} disabled={saving}>
				{#if saving}
					<LoaderIcon class="mr-2 h-4 w-4 animate-spin" />
				{:else}
					<SaveIcon class="mr-2 h-4 w-4" />
				{/if}
				Save Changes
			</Button>
		</div>
	</div>
</PageContainer>
