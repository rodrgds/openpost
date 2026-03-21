<script lang="ts">
	import { onMount } from 'svelte';
	import { api } from '$lib/api/client';
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import { Button } from '$lib/components/ui/button';
	import { Card, CardContent, CardHeader, CardTitle, CardFooter } from '$lib/components/ui/card';
	import { Checkbox } from '$lib/components/ui/checkbox';
	import { Input } from '$lib/components/ui/input';
	import { Label } from '$lib/components/ui/label';
	import { Textarea } from '$lib/components/ui/textarea';
	import { Calendar } from '$lib/components/ui/calendar';
	import { CalendarDate, getLocalTimeZone, today } from '@internationalized/date';
	import type { SocialAccount } from '$lib/types';
	
	let workspaceId = $derived($page.params.id);
	let content = $state('');
	let isSubmitting = $state(false);
	let error = $state('');
	let accounts = $state<SocialAccount[]>([]);
	let selectedAccountIds = $state<string[]>([]);
	let loading = $state(true);
	
	// Calendar state
	let selectedDate = $state<CalendarDate | undefined>(undefined);
	let selectedTime = $state<string | null>(null);
	
	const timeSlots = Array.from({ length: 37 }, (_, i) => {
		const totalMinutes = i * 15;
		const hour = Math.floor(totalMinutes / 60) + 9;
		const minute = totalMinutes % 60;
		return `${hour.toString().padStart(2, '0')}:${minute.toString().padStart(2, '0')}`;
	});
	
	onMount(async () => {
		// Set default to tomorrow at 10:00
		const tomorrow = today(getLocalTimeZone()).add({ days: 1 });
		selectedDate = new CalendarDate(tomorrow.year, tomorrow.month, tomorrow.day);
		selectedTime = '10:00';
		
		if (!workspaceId) return;
		
		try {
			accounts = await api.listAccounts(workspaceId);
		} catch (e) {
			console.error('Failed to load accounts:', e);
			accounts = [];
		} finally {
			loading = false;
		}
	});
	
	function toggleAccount(id: string) {
		if (selectedAccountIds.includes(id)) {
			selectedAccountIds = selectedAccountIds.filter(a => a !== id);
		} else {
			selectedAccountIds = [...selectedAccountIds, id];
		}
	}
	
	function getScheduledAt(): string | undefined {
		if (!selectedDate || !selectedTime) return undefined;
		const [hours, minutes] = selectedTime.split(':').map(Number);
		const date = selectedDate.toDate(getLocalTimeZone());
		date.setHours(hours, minutes, 0, 0);
		return date.toISOString();
	}
	
	async function createPost(publishNow: boolean = false) {
		error = '';
		
		if (!workspaceId) {
			error = 'Workspace not found';
			return;
		}
		
		if (!content.trim()) {
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
			await api.createPost(workspaceId, content, selectedAccountIds, scheduledAt);
			goto(`/workspace/${workspaceId}`);
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
	
	function getPlatformIcon(platform: string): string {
		switch (platform) {
			case 'x': return '𝕏';
			case 'mastodon': return '🐘';
			case 'threads': return '📸';
			case 'bluesky': return '🦋';
			case 'linkedin': return '💼';
			default: return '?';
		}
	}
</script>

<svelte:head>
	<title>Compose Post - OpenPost</title>
</svelte:head>

<div class="mx-auto w-full max-w-[1100px] px-4 py-6 lg:px-8">
	<div class="mb-8">
		<a href="/workspace/{workspaceId}" class="text-primary hover:underline text-sm font-medium">
			← Back to Posts
		</a>
	</div>
	
	<Card>
		<CardHeader>
			<CardTitle>Compose Post</CardTitle>
		</CardHeader>
		<CardContent>
			{#if error}
				<div class="mb-4 p-3 bg-destructive/10 border border-destructive/20 rounded-md text-destructive text-sm">
					{error}
				</div>
			{/if}
			
			<form onsubmit={handleSubmit} class="space-y-6">
				<div class="space-y-2">
					<Label for="content">Post Content</Label>
					<Textarea
						id="content"
						bind:value={content}
						rows={6}
						placeholder="What's on your mind?"
						required
					/>
					<div class="flex justify-end">
						<span class="text-xs text-muted-foreground">{content.length} characters</span>
					</div>
				</div>
				
				<div class="space-y-2">
					<Label>Publish to</Label>
					{#if loading}
						<div class="flex justify-center py-4">
							<div class="animate-spin rounded-full h-6 w-6 border-b-2 border-primary"></div>
						</div>
					{:else if !accounts || accounts.length === 0}
						<div class="bg-muted border border-border rounded-md p-4 text-sm text-muted-foreground">
							No accounts connected. <a href="/accounts" class="underline font-medium text-primary">Connect an account</a> to publish posts.
						</div>
					{:else}
						<div class="space-y-2">
							{#each accounts as account}
								<label class="flex items-center gap-3 p-3 border rounded-md cursor-pointer hover:bg-muted/50 transition-colors {selectedAccountIds.includes(account.id) ? 'border-primary bg-primary/5' : 'border-border'}">
									<Checkbox
										checked={selectedAccountIds.includes(account.id)}
										onCheckedChange={() => toggleAccount(account.id)}
									/>
									<div class="flex h-8 w-8 items-center justify-center rounded-full bg-primary text-primary-foreground">
										<span class="font-bold text-sm">{getPlatformIcon(account.platform)}</span>
									</div>
									<div>
										<div class="font-medium capitalize">{account.platform}</div>
										<div class="text-xs text-muted-foreground">
											{#if account.account_username}
												@{account.account_username}
											{:else if account.instance_url}
												{account.instance_url.replace('https://', '')}
											{:else}
												Connected
											{/if}
										</div>
									</div>
								</label>
							{/each}
						</div>
					{/if}
				</div>
				
				<div class="space-y-2">
					<Label>Schedule Date & Time</Label>
					<Card class="gap-0 p-0">
						<CardContent class="relative p-0 md:pe-48">
							<div class="p-6">
								<Calendar
									type="single"
									bind:value={selectedDate}
									class="bg-transparent p-0 [--cell-size:--spacing(10)] md:[--cell-size:--spacing(12)] [&_[data-outside-month]]:hidden"
									weekdayFormat="short"
								/>
							</div>
							<div class="no-scrollbar inset-y-0 end-0 flex max-h-72 w-full scroll-pb-6 flex-col gap-4 overflow-y-auto border-t p-6 md:absolute md:max-h-none md:w-48 md:border-s md:border-t-0">
								<div class="grid gap-2">
									{#each timeSlots as time (time)}
										<Button
											variant={selectedTime === time ? 'default' : 'outline'}
											onclick={() => (selectedTime = time)}
											class="w-full shadow-none"
										>
											{time}
										</Button>
									{/each}
								</div>
							</div>
						</CardContent>
						<CardFooter class="flex flex-col gap-4 border-t px-6 py-5 md:flex-row">
							<div class="text-sm text-muted-foreground">
								{#if selectedDate && selectedTime}
									Your post is scheduled for
									<span class="font-medium text-foreground">
										{selectedDate.toDate(getLocalTimeZone()).toLocaleDateString('en-US', {
											weekday: 'long',
											day: 'numeric',
											month: 'short',
										})}
									</span>
									at <span class="font-medium text-foreground">{selectedTime}</span>.
								{:else}
									Select a date and time for your post.
								{/if}
							</div>
						</CardFooter>
					</Card>
				</div>
				
				<div class="flex gap-3 justify-end">
					<Button type="button" variant="outline" onclick={() => goto(`/workspace/${workspaceId}`)}>Cancel</Button>
					<Button
						type="button"
						variant="secondary"
						onclick={handlePostNow}
						disabled={isSubmitting || !content.trim() || selectedAccountIds.length === 0}
					>
						{isSubmitting ? 'Posting...' : 'Post Now'}
					</Button>
					<Button
						type="submit"
						disabled={isSubmitting || !selectedDate || !selectedTime}
					>
						{isSubmitting ? 'Creating...' : 'Schedule Post'}
					</Button>
				</div>
			</form>
		</CardContent>
</Card>
</div>
