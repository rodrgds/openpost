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
				media_cleanup_days: workspaceCtx.settings.media_cleanup_days
			});
			toastMessage = 'Settings saved successfully';
		} catch (e) {
			toastMessage = (e as Error).message;
		} finally {
			saving = false;
		}
	}

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
		<section class="rounded-lg border p-6">
			<h2 class="mb-4 text-lg font-semibold">Workspace</h2>
			<div class="space-y-4">
				<div class="flex items-center gap-4">
					<span class="text-sm font-medium">Current Workspace</span>
					<span class="text-sm text-muted-foreground">{workspaceCtx.currentWorkspace?.name}</span>
				</div>
			</div>
		</section>

		<!-- Date & Time Settings -->
		<section class="rounded-lg border p-6">
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
						<p class="text-xs text-muted-foreground">Used for displaying scheduled post times</p>
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
						<p class="text-xs text-muted-foreground">Affects the calendar display in the sidebar</p>
					</div>
				</div>
			</div>
		</section>

		<!-- Media Cleanup Settings -->
		<section class="rounded-lg border p-6">
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
					<p class="text-xs text-muted-foreground">
						Automatically delete unused, non-favorited media after this period. Favorited media is
						always kept.
					</p>
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
