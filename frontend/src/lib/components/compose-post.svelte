<script lang="ts">
	import { onMount } from 'svelte';
	import { client, type SocialAccount, type Workspace, getToken } from '$lib/api/client';
	import { getApiBase } from '$lib/stores/instance.svelte';
	import { Button } from '$lib/components/ui/button';
	import { Card, CardContent } from '$lib/components/ui/card';
	import { Checkbox } from '$lib/components/ui/checkbox';
	import { Label } from '$lib/components/ui/label';
	import { Textarea } from '$lib/components/ui/textarea';
	import { Calendar } from '$lib/components/ui/calendar';
	import * as Select from '$lib/components/ui/select';
	import {
		CalendarDate,
		getLocalTimeZone,
		today,
		isEqualDay,
		type DateValue
	} from '@internationalized/date';
	import LoaderIcon from 'lucide-svelte/icons/loader-2';
	import LayersIcon from 'lucide-svelte/icons/layers';
	import PlusIcon from 'lucide-svelte/icons/plus';
	import XIcon from 'lucide-svelte/icons/x';
	import { getPlatformKey, getPlatformName } from '$lib/utils';
	import PlatformIcon from '$lib/components/platform-icon.svelte';
	import * as Collapsible from '$lib/components/ui/collapsible';

	type SocialAccountWithThreadSupport = SocialAccount & {
		thread_replies_supported?: boolean;
	};

	interface Props {
		initialDate?: DateValue;
		onSuccess?: () => void;
		onCancel?: () => void;
	}

	let { initialDate, onSuccess, onCancel }: Props = $props();

	let content = $state('');
	let mediaIds = $state<string[]>([]);
	let isThreadMode = $state(false);
	let threadPosts = $state<Array<{ content: string; mediaIds: string[] }>>([
		{ content: '', mediaIds: [] }
	]);
	let isSubmitting = $state(false);
	let error = $state('');
	let workspaces = $state<Workspace[]>([]);
	let selectedWorkspaceId = $state<string>('');
	let accounts = $state<SocialAccountWithThreadSupport[]>([]);
	let selectedAccountIds = $state<string[]>([]);
	let loadingWorkspaces = $state(true);
	let loadingAccounts = $state(false);
	let accountsPanelOpen = $state(false);

	let selectedDate = $state<CalendarDate | undefined>(undefined);
	let selectedTime = $state<string | null>(null);

	const allTimeSlots = Array.from({ length: 37 }, (_, i) => {
		const totalMinutes = i * 15;
		const hour = Math.floor(totalMinutes / 60) + 9;
		const minute = totalMinutes % 60;
		return `${hour.toString().padStart(2, '0')}:${minute.toString().padStart(2, '0')}`;
	});

	const isToday = $derived(
		selectedDate ? isEqualDay(selectedDate, today(getLocalTimeZone())) : false
	);

	const timeSlots = $derived.by(() => {
		if (!isToday) return allTimeSlots;
		const now = new Date();
		const currentMinutes = now.getHours() * 60 + now.getMinutes();
		return allTimeSlots.filter((slot) => {
			const [h, m] = slot.split(':').map(Number);
			return h * 60 + m > currentMinutes;
		});
	});

	$effect(() => {
		if (isToday && selectedTime && !timeSlots.includes(selectedTime)) {
			selectedTime = timeSlots.length > 0 ? timeSlots[0] : null;
		}
	});

	onMount(async () => {
		if (initialDate) {
			selectedDate = new CalendarDate(initialDate.year, initialDate.month, initialDate.day);
		} else {
			const tomorrow = today(getLocalTimeZone()).add({ days: 1 });
			selectedDate = new CalendarDate(tomorrow.year, tomorrow.month, tomorrow.day);
		}
		selectedTime = '10:00';

		try {
			const { data, error: err } = await client.GET('/workspaces');
			if (err || !data) throw new Error('Failed to load workspaces');
			workspaces = data;
			if (workspaces.length > 0) {
				selectedWorkspaceId = workspaces[0].id;
				await loadAccounts(selectedWorkspaceId);
			}
		} catch (e) {
			console.error('Failed to load workspaces:', e);
			error = (e as Error).message;
		} finally {
			loadingWorkspaces = false;
		}
	});

	async function loadAccounts(workspaceId: string) {
		if (!workspaceId) return;
		loadingAccounts = true;
		try {
			const { data, error: err } = await client.GET('/accounts', {
				params: { query: { workspace_id: workspaceId } }
			});
			accounts = data ?? [];
			const allowedIds = accounts
				.filter((a) => !isThreadMode || a.thread_replies_supported !== false)
				.map((a) => a.id);
			selectedAccountIds = allowedIds;
			accountsPanelOpen = false;
		} catch (e) {
			console.error('Failed to load accounts:', e);
			accounts = [];
			selectedAccountIds = [];
		} finally {
			loadingAccounts = false;
		}
	}

	function handleWorkspaceChange(value: string) {
		selectedWorkspaceId = value;
		loadAccounts(value);
	}

	function toggleAccount(id: string) {
		const account = accounts.find((a) => a.id === id);
		if (account && isThreadMode && account.thread_replies_supported === false) {
			return;
		}

		if (selectedAccountIds.includes(id)) {
			selectedAccountIds = selectedAccountIds.filter((a) => a !== id);
		} else {
			selectedAccountIds = [...selectedAccountIds, id];
		}
	}

	function selectAllAccounts() {
		selectedAccountIds = accounts
			.filter((a) => !isThreadMode || a.thread_replies_supported !== false)
			.map((a) => a.id);
	}

	function clearAllAccounts() {
		selectedAccountIds = [];
	}

	function isThreadDisabledAccount(account: SocialAccountWithThreadSupport): boolean {
		return isThreadMode && account.thread_replies_supported === false;
	}

	function getScheduledAt(): string | undefined {
		if (!selectedDate || !selectedTime) return undefined;
		const [hours, minutes] = selectedTime.split(':').map(Number);
		const date = selectedDate.toDate(getLocalTimeZone());
		date.setHours(hours, minutes, 0, 0);
		return date.toISOString();
	}

	function hasValidContent(): boolean {
		if (isThreadMode) {
			return threadPosts.some((p) => p.content.trim().length > 0);
		}
		return content.trim().length > 0;
	}

	let isDraggingFile = $state(false);
	let isUploading = $state(false);

	async function handleFileUpload(files: FileList | File[]) {
		if (!selectedWorkspaceId || isSubmitting) return;

		isUploading = true;
		try {
			for (const file of Array.from(files)) {
				if (!file.type.startsWith('image/') && !file.type.startsWith('video/')) continue;
				if (mediaIds.length >= 4) break;

				const formData = new FormData();
				formData.append('file', file);
				formData.append('workspace_id', selectedWorkspaceId);

				const token = getToken();
				const resp = await fetch(`${getApiBase()}/media/upload`, {
					method: 'POST',
					headers: token ? { Authorization: `Bearer ${token}` } : {},
					body: formData
				});

				if (resp.ok) {
					const data = await resp.json();
					mediaIds = [...mediaIds, data.id];
				}
			}
		} catch (e) {
			console.error('Failed to upload media:', e);
		} finally {
			isUploading = false;
		}
	}

	function handlePaste(e: ClipboardEvent) {
		const items = e.clipboardData?.items;
		if (!items) return;

		const files: File[] = [];
		for (const item of Array.from(items)) {
			if (item.kind === 'file') {
				const file = item.getAsFile();
				if (file) files.push(file);
			}
		}
		if (files.length > 0) {
			e.preventDefault();
			handleFileUpload(files);
		}
	}

	function handleDragOver(e: DragEvent) {
		e.preventDefault();
		isDraggingFile = true;
	}

	function handleDragLeave(e: DragEvent) {
		e.preventDefault();
		isDraggingFile = false;
	}

	function handleDrop(e: DragEvent) {
		e.preventDefault();
		isDraggingFile = false;
		const files = e.dataTransfer?.files;
		if (files && files.length > 0) {
			handleFileUpload(files);
		}
	}

	async function createPost(publishNow: boolean = false) {
		error = '';

		if (!selectedWorkspaceId) {
			error = 'Please select a workspace';
			return;
		}

		if (!hasValidContent()) {
			error = 'Please enter some content';
			return;
		}

		if (selectedAccountIds.length === 0) {
			error = 'Please select at least one account to publish to';
			return;
		}

		let scheduledAt: string | undefined;
		if (publishNow) {
			scheduledAt = new Date().toISOString();
		} else {
			scheduledAt = getScheduledAt();
		}

		isSubmitting = true;

		try {
			if (isThreadMode) {
				const validPosts = threadPosts.filter((p) => p.content.trim().length > 0);
				if (validPosts.length < 2) {
					error = 'A thread must have at least 2 posts with content';
					isSubmitting = false;
					return;
				}

				const { error: err } = await client.POST('/posts/thread' as any, {
					body: {
						workspace_id: selectedWorkspaceId,
						social_account_ids: selectedAccountIds,
						scheduled_at: scheduledAt,
						posts: validPosts.map((p) => ({
							content: p.content,
							media_ids: p.mediaIds
						}))
					}
				});
				if (err) throw new Error((err as any).detail || 'Failed to create thread');
			} else {
				const { error: err } = await client.POST('/posts', {
					body: {
						workspace_id: selectedWorkspaceId,
						content,
						social_account_ids: selectedAccountIds,
						scheduled_at: scheduledAt,
						media_ids: mediaIds
					}
				});
				if (err) throw new Error(err.detail || 'Failed to create post');
			}
			if (onSuccess) onSuccess();
		} catch (e) {
			error = (e as Error).message || 'Failed to create post';
		} finally {
			isSubmitting = false;
		}
	}

	async function handleSubmit(e: Event) {
		e.preventDefault();
		await createPost(false);
	}

	async function handlePostNow() {
		await createPost(true);
	}

	function toggleThreadMode() {
		if (!isThreadMode) {
			threadPosts = [
				{ content, mediaIds },
				{ content: '', mediaIds: [] }
			];
			const threadSafeSelection = selectedAccountIds.filter((id) => {
				const account = accounts.find((a) => a.id === id);
				return account?.thread_replies_supported !== false;
			});
			selectedAccountIds =
				threadSafeSelection.length > 0
					? threadSafeSelection
					: accounts.filter((a) => a.thread_replies_supported !== false).map((a) => a.id);
		} else {
			content = threadPosts[0]?.content ?? '';
			mediaIds = threadPosts[0]?.mediaIds ?? [];
			threadPosts = [{ content, mediaIds }];
		}
		isThreadMode = !isThreadMode;
	}

	const selectableAccountsCount = $derived(
		accounts.filter((a) => !isThreadMode || a.thread_replies_supported !== false).length
	);

	const selectedSelectableAccountsCount = $derived(
		accounts.filter(
			(a) =>
				selectedAccountIds.includes(a.id) && (!isThreadMode || a.thread_replies_supported !== false)
		).length
	);

	const accountSelectionSummary = $derived.by(() => {
		if (accounts.length === 0) return 'No accounts connected';
		if (selectedSelectableAccountsCount === 0) return 'No accounts selected';
		if (selectedSelectableAccountsCount === selectableAccountsCount) {
			return isThreadMode
				? `All ${selectableAccountsCount} supported accounts`
				: `All ${accounts.length} accounts`;
		}
		return `${selectedSelectableAccountsCount} of ${selectableAccountsCount} selected`;
	});

	function addThreadPost() {
		threadPosts = [...threadPosts, { content: '', mediaIds: [] }];
	}

	function removeThreadPost(index: number) {
		threadPosts = threadPosts.filter((_, i) => i !== index);
		if (threadPosts.length <= 1) {
			content = threadPosts[0]?.content ?? '';
			mediaIds = threadPosts[0]?.mediaIds ?? [];
			isThreadMode = false;
		}
	}
</script>

<div class="space-y-6">
	{#if error}
		<div
			class="rounded-md border border-destructive/20 bg-destructive/10 p-3 text-sm text-destructive"
		>
			{error}
		</div>
	{/if}

	<form onsubmit={handleSubmit} class="space-y-6 pb-6">
		<div class="grid grid-cols-1 gap-6 sm:grid-cols-2">
			<div class="space-y-6">
				<div class="space-y-2">
					<Label for="workspace">Workspace</Label>
					<Select.Root
						type="single"
						bind:value={selectedWorkspaceId}
						onValueChange={handleWorkspaceChange}
					>
						<Select.Trigger class="w-full">
							{workspaces.find((w) => w.id === selectedWorkspaceId)?.name ||
								(loadingWorkspaces ? 'Loading...' : 'Select a workspace')}
						</Select.Trigger>
						<Select.Content>
							{#each workspaces as workspace}
								<Select.Item value={workspace.id}>{workspace.name}</Select.Item>
							{/each}
						</Select.Content>
					</Select.Root>
				</div>

				{#if isThreadMode}
					<div class="space-y-3">
						<div class="flex items-center justify-between">
							<div class="flex items-center gap-2">
								<Label>Thread</Label>
								<span class="text-xs text-muted-foreground">
									{threadPosts.length} posts
								</span>
							</div>
							<Button
								type="button"
								variant="ghost"
								size="sm"
								onclick={addThreadPost}
								class="gap-1 text-xs text-muted-foreground"
							>
								<PlusIcon class="h-3.5 w-3.5" />
								Add
							</Button>
						</div>
						{#each threadPosts as post, i (i)}
							<div class="flex items-start gap-2">
								<div
									class="mt-2 flex h-5 w-5 shrink-0 items-center justify-center rounded-full bg-primary text-xs font-bold text-primary-foreground"
								>
									{i + 1}
								</div>
								<div class="flex-1 space-y-1">
									<Textarea
										bind:value={post.content}
										rows={3}
										placeholder="What's in this post?"
										disabled={isSubmitting}
										class="resize-none"
									/>
									<div class="flex items-center justify-between">
										<span class="text-xs text-muted-foreground">
											{post.content.length}
										</span>
										{#if threadPosts.length > 1}
											<button
												type="button"
												class="text-xs text-muted-foreground hover:text-destructive"
												onclick={() => removeThreadPost(i)}
											>
												Remove
											</button>
										{/if}
									</div>
								</div>
							</div>
						{/each}
					</div>
				{:else}
					<div class="space-y-2">
						<div class="flex items-center justify-between">
							<Label for="content">Post Content</Label>
							<Button
								type="button"
								variant="ghost"
								size="sm"
								onclick={toggleThreadMode}
								class="gap-1 text-xs text-muted-foreground"
							>
								<LayersIcon class="h-3.5 w-3.5" />
								Thread
							</Button>
						</div>
						<div
							class="relative rounded-md border transition-colors {isDraggingFile
								? 'border-primary bg-primary/5'
								: 'border-border'} {isSubmitting ? 'pointer-events-none opacity-50' : ''}"
							role="region"
							aria-label="Drop area for media files"
							ondragover={handleDragOver}
							ondragleave={handleDragLeave}
							ondrop={handleDrop}
						>
							{#if mediaIds.length > 0}
								<div class="mb-2 flex flex-wrap gap-2">
									{#each mediaIds as mediaId, idx}
										<div class="relative h-16 w-16 overflow-hidden rounded-md border">
											<img
												src="{getApiBase()}/media/{mediaId}"
												alt="Attached media"
												class="h-full w-full object-cover"
											/>
											<button
												type="button"
												class="absolute top-1 right-1 rounded-full bg-black/40 p-1 text-white hover:bg-black/60"
												onclick={() => {
													mediaIds = mediaIds.filter((_, i) => i !== idx);
												}}
											>
												<XIcon class="h-3 w-3" />
											</button>
										</div>
									{/each}
								</div>
							{/if}
							<Textarea
								id="content"
								bind:value={content}
								rows={mediaIds.length > 0 ? 4 : 6}
								placeholder="What's on your mind? Drop images, paste from clipboard, or type..."
								required
								class="resize-none border-0 bg-transparent focus:ring-0 focus:outline-none"
								onpaste={handlePaste}
							/>
							{#if isUploading}
								<div class="absolute inset-0 flex items-center justify-center bg-background/80">
									<LoaderIcon class="h-5 w-5 animate-spin text-primary" />
								</div>
							{/if}
						</div>
						<div class="flex items-center justify-between">
							<span class="text-xs text-muted-foreground">{content.length} characters</span>
							{#if mediaIds.length > 0}
								<span class="text-xs text-muted-foreground">
									{mediaIds.length} media attached
								</span>
							{/if}
						</div>
					</div>
				{/if}

				<div class="space-y-2">
					<Label>Publish to</Label>
					{#if loadingAccounts}
						<div class="flex justify-center py-4">
							<LoaderIcon class="h-6 w-6 animate-spin text-primary" />
						</div>
					{:else if !accounts || accounts.length === 0}
						<div class="rounded-md border border-border bg-muted p-4 text-sm text-muted-foreground">
							{#if selectedWorkspaceId}No accounts connected. <a
									href="/accounts"
									class="font-medium text-primary underline"
									onclick={onCancel}>Connect</a
								>{:else}Select a workspace first.{/if}
						</div>
					{:else}
						<Collapsible.Root bind:open={accountsPanelOpen}>
							<Collapsible.Trigger
								class="flex w-full items-center justify-between rounded-md border border-border bg-muted/30 px-3 py-2 text-sm"
							>
								<span class="flex items-center gap-2">
									<PlatformIcon platform={getPlatformKey(accounts[0].platform)} class="h-4 w-4" />
									<span>{accountSelectionSummary}</span>
								</span>
								<span class="text-xs text-muted-foreground">
									{accountsPanelOpen ? 'Hide' : 'Customize'}
								</span>
							</Collapsible.Trigger>
							<Collapsible.Content>
								<div class="mt-2 space-y-2">
									<div class="flex items-center justify-end gap-2">
										<button
											type="button"
											class="text-xs text-muted-foreground hover:text-foreground"
											onclick={clearAllAccounts}
										>
											Clear
										</button>
										<button
											type="button"
											class="text-xs text-primary hover:underline"
											onclick={selectAllAccounts}
										>
											Select all
										</button>
									</div>
									<div class="grid grid-cols-1 gap-2 sm:grid-cols-2">
										{#each accounts as account}
											{@const threadDisabled = isThreadDisabledAccount(account)}
											<label
												class="flex items-center gap-3 rounded-md border p-3 transition-colors {threadDisabled
													? 'cursor-not-allowed border-muted-foreground/20 bg-muted/40 opacity-60'
													: 'cursor-pointer hover:bg-muted/50'} {selectedAccountIds.includes(
													account.id
												)
													? 'border-primary bg-primary/5'
													: 'border-border'}"
											>
												<Checkbox
													checked={selectedAccountIds.includes(account.id)}
													disabled={threadDisabled}
													onCheckedChange={() => toggleAccount(account.id)}
												/>
												<div
													class="flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-primary text-primary-foreground"
												>
													<PlatformIcon
														platform={getPlatformKey(account.platform)}
														class="h-4 w-4"
													/>
												</div>
												<div class="min-w-0">
													<div class="truncate font-medium capitalize">
														{getPlatformName(account.platform)}
													</div>
													<div class="truncate text-xs text-muted-foreground">
														{#if account.account_username}
															@{account.account_username}
														{:else if account.instance_url}
															{account.instance_url.replace('https://', '')}
														{:else}
															Connected
														{/if}
													</div>
													{#if threadDisabled}
														<div class="truncate text-[11px] text-muted-foreground">
															Thread disabled
														</div>
													{/if}
												</div>
											</label>
										{/each}
									</div>
								</div>
							</Collapsible.Content>
						</Collapsible.Root>
					{/if}
				</div>
			</div>

			<div class="space-y-2">
				<Label>Schedule Date & Time</Label>
				<Card class="gap-0 overflow-hidden border p-0 shadow-none">
					<CardContent class="relative p-0 sm:pe-40">
						<div class="p-4">
							<Calendar
								type="single"
								bind:value={selectedDate}
								minValue={today(getLocalTimeZone())}
								class="bg-transparent p-0 [--cell-size:--spacing(10)]"
								weekdayFormat="short"
							/>
						</div>
						<div
							class="inset-y-0 end-0 no-scrollbar flex max-h-72 w-full scroll-pb-6 flex-col gap-4 overflow-y-auto border-t p-4 sm:absolute sm:max-h-none sm:w-40 sm:border-s sm:border-t-0"
						>
							<div class="grid gap-2">
								{#each timeSlots as time (time)}
									<Button
										variant={selectedTime === time ? 'default' : 'outline'}
										onclick={() => (selectedTime = time)}
										class="h-8 w-full py-1 text-xs shadow-none"
									>
										{time}
									</Button>
								{/each}
							</div>
						</div>
					</CardContent>
					<div class="flex flex-col gap-4 border-t bg-muted/30 px-4 py-3 md:flex-row">
						<div class="text-xs text-muted-foreground">
							{#if selectedDate && selectedTime}
								Scheduled for <span class="font-medium text-foreground">
									{selectedDate.toDate(getLocalTimeZone()).toLocaleDateString('en-US', {
										day: 'numeric',
										month: 'short'
									})}
								</span>
								at <span class="font-medium text-foreground">{selectedTime}</span>.
							{:else}
								Select a date and time.
							{/if}
						</div>
					</div>
				</Card>
			</div>
		</div>

		<div class="flex justify-end gap-3 border-t pt-4">
			{#if onCancel}
				<Button type="button" variant="outline" onclick={onCancel}>Cancel</Button>
			{/if}
			<Button
				type="button"
				variant="secondary"
				onclick={handlePostNow}
				disabled={isSubmitting || !hasValidContent() || selectedAccountIds.length === 0}
			>
				{isSubmitting ? 'Posting...' : 'Post Now'}
			</Button>
			<Button type="submit" disabled={isSubmitting || !selectedDate || !selectedTime}>
				{isSubmitting ? 'Creating...' : 'Schedule Post'}
			</Button>
		</div>
	</form>
</div>
