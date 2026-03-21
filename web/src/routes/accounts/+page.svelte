<script lang="ts">
	import { onMount } from 'svelte';
	import { auth } from '$lib/stores/auth';
	import { api } from '$lib/api/client';
	import { Button } from '$lib/components/ui/button';
	import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '$lib/components/ui/card';
	import { Input } from '$lib/components/ui/input';
	import { Label } from '$lib/components/ui/label';
	import * as DropdownMenu from '$lib/components/ui/dropdown-menu';
	import type { Workspace, SocialAccount } from '$lib/types';
	import { goto } from '$app/navigation';
	import FolderOpenIcon from 'lucide-svelte/icons/folder-open';
	import ChevronDownIcon from 'lucide-svelte/icons/chevron-down';
	
	let workspaces = $state<Workspace[] | null>(null);
	let selectedWorkspaceId = $state('');
	let loading = $state(true);
	let error = $state('');
	
	let accounts = $state<SocialAccount[]>([]);
	let accountsLoading = $state(false);
	
	let mastodonInstance = $state('mastodon.social');
	let showMastodonModal = $state(false);
	let selectedWorkspaceName = $derived(
		workspaces?.find((workspace) => workspace.id === selectedWorkspaceId)?.name || 'Select workspace'
	);
	
	async function loadAccounts() {
		if (!selectedWorkspaceId) return;
		accountsLoading = true;
		try {
			accounts = await api.listAccounts(selectedWorkspaceId);
		} catch (e) {
			console.error('Failed to load accounts:', e);
			accounts = [];
		} finally {
			accountsLoading = false;
		}
	}
	
	onMount(() => {
		auth.subscribe(async state => {
			if (!state.isLoading && !state.isAuthenticated) {
				goto('/login');
			} else if (!state.isLoading && state.isAuthenticated) {
				try {
					workspaces = await api.listWorkspaces();
					if (workspaces && workspaces.length > 0) {
						selectedWorkspaceId = workspaces[0].id;
						await loadAccounts();
					}
				} catch (e) {
					console.error('Failed to load workspaces:', e);
				} finally {
					loading = false;
				}
			}
		});
	});
	
	$effect(() => {
		if (selectedWorkspaceId) {
			loadAccounts();
		}
	});
	
	async function connectTwitter() {
		if (!selectedWorkspaceId) {
			alert('Please create a workspace first');
			return;
		}
		try {
			const response = await api.getTwitterAuthUrl(selectedWorkspaceId);
			window.location.href = response.url;
		} catch (e) {
			error = (e as Error).message;
		}
	}
	
	async function connectMastodon() {
		if (!selectedWorkspaceId) {
			alert('Please create a workspace first');
			return;
		}
		if (!mastodonInstance.trim()) {
			alert('Please enter a Mastodon instance URL');
			return;
		}
		
		try {
			let instance = mastodonInstance.trim();
			if (!instance.startsWith('http')) {
				instance = 'https://' + instance;
			}
			
			localStorage.setItem('oauth_workspace_id', selectedWorkspaceId);
			localStorage.setItem('oauth_mastodon_instance', instance);
			
			const response = await api.getMastodonAuthUrl(selectedWorkspaceId, instance);
			window.location.href = response.url;
		} catch (e) {
			error = (e as Error).message;
		}
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
	
	function getPlatformColor(platform: string): string {
		switch (platform) {
			case 'x': return 'bg-black';
			case 'mastodon': return 'bg-indigo-500';
			case 'threads': return 'bg-orange-500';
			case 'bluesky': return 'bg-sky-500';
			case 'linkedin': return 'bg-blue-600';
			default: return 'bg-gray-500';
		}
	}
</script>

<svelte:head>
	<title>Connected Accounts - OpenPost</title>
</svelte:head>

{#if loading}
	<div class="flex justify-center py-12">
		<div class="animate-spin rounded-full h-8 w-8 border-b-2 border-primary"></div>
	</div>
{:else if !workspaces || workspaces.length === 0}
	<div class="max-w-4xl mx-auto px-4 py-8">
		<Card class="text-center">
			<CardContent class="pt-6">
				<CardTitle class="mb-2">No Workspaces Found</CardTitle>
				<CardDescription class="mb-4">You need to create a workspace before connecting social accounts.</CardDescription>
				<Button href="/">Create Workspace</Button>
			</CardContent>
		</Card>
	</div>
{:else}
	<div class="max-w-4xl mx-auto px-4 py-8">
		<h1 class="text-2xl font-bold mb-6">Connected Accounts</h1>
		
		{#if error}
			<div class="mb-4 p-3 bg-destructive/10 border border-destructive/20 rounded-md text-destructive">
				{error}
				<Button variant="ghost" size="sm" onclick={() => error = ''}>Dismiss</Button>
			</div>
		{/if}
		
		<div class="mb-6">
			<Label class="mb-2">Workspace</Label>
			<DropdownMenu.Root>
				<DropdownMenu.Trigger>
					{#snippet child({ props })}
						<Button {...props} variant="outline" class="w-full max-w-sm justify-between">
							<span class="flex items-center gap-2 truncate">
								<span class="inline-flex size-6 items-center justify-center rounded-md border border-border bg-muted/50">
									<FolderOpenIcon class="size-3.5" />
								</span>
								<span class="truncate">{selectedWorkspaceName}</span>
							</span>
							<ChevronDownIcon class="size-4 opacity-70" />
						</Button>
					{/snippet}
				</DropdownMenu.Trigger>
				<DropdownMenu.Content class="w-72 rounded-lg" align="start" side="bottom" sideOffset={6}>
					<DropdownMenu.Label class="text-xs text-muted-foreground">Workspaces</DropdownMenu.Label>
					{#each workspaces as workspace (workspace.id)}
						<DropdownMenu.Item onSelect={() => (selectedWorkspaceId = workspace.id)} class="gap-2 p-2">
							<span class="inline-flex size-6 items-center justify-center rounded-md border border-border text-[0.625rem] font-semibold">
								{workspace.name.slice(0, 1).toUpperCase()}
							</span>
							<span class="truncate">{workspace.name}</span>
						</DropdownMenu.Item>
					{/each}
				</DropdownMenu.Content>
			</DropdownMenu.Root>
		</div>
		
		<div class="mb-8">
			<h2 class="text-lg font-medium mb-4">Connected Accounts</h2>
			
			{#if accountsLoading}
				<div class="flex justify-center py-4">
					<div class="animate-spin rounded-full h-6 w-6 border-b-2 border-primary"></div>
				</div>
			{:else if !accounts || accounts.length === 0}
				<div class="bg-muted/50 border rounded-md p-4 text-center text-muted-foreground">
					No accounts connected yet. Connect a platform below to get started.
				</div>
			{:else}
				<div class="space-y-3">
					{#each accounts as account}
						<Card>
							<CardContent class="flex items-center justify-between p-4">
								<div class="flex items-center gap-3">
									<div class="w-10 h-10 {getPlatformColor(account.platform)} rounded-full flex items-center justify-center">
										<span class="text-white font-bold text-lg">{getPlatformIcon(account.platform)}</span>
									</div>
									<div>
										<h3 class="font-medium capitalize">{account.platform}</h3>
										<p class="text-sm text-muted-foreground">
											{#if account.account_username}
												@{account.account_username}
											{:else if account.instance_url}
												{account.instance_url.replace('https://', '')}
											{:else}
												Account ID: {account.account_id}
											{/if}
										</p>
									</div>
								</div>
								<span class="px-2 py-1 bg-green-100 text-green-700 text-xs font-medium rounded">Connected</span>
							</CardContent>
						</Card>
					{/each}
				</div>
			{/if}
		</div>
		
		<div class="space-y-4">
			<h2 class="text-lg font-medium">Connect a Platform</h2>
			
			<Card>
				<CardContent class="flex items-center justify-between p-4">
					<div class="flex items-center gap-3">
						<div class="w-10 h-10 bg-black rounded-full flex items-center justify-center">
							<span class="text-white font-bold">X</span>
						</div>
						<div>
							<h3 class="font-medium">X (Twitter)</h3>
							<p class="text-sm text-muted-foreground">Connect your X account to post tweets</p>
						</div>
					</div>
					<Button onclick={connectTwitter}>Connect</Button>
				</CardContent>
			</Card>
			
			<Card>
				<CardContent class="flex items-center justify-between p-4">
					<div class="flex items-center gap-3">
						<div class="w-10 h-10 bg-indigo-500 rounded-full flex items-center justify-center">
							<span class="text-white font-bold text-lg">🐘</span>
						</div>
						<div>
							<h3 class="font-medium">Mastodon</h3>
							<p class="text-sm text-muted-foreground">Connect your Mastodon account from any instance</p>
						</div>
					</div>
					<div class="flex gap-2">
						<Button href="/accounts/mastodon/callback" variant="outline">Enter Code</Button>
						<Button onclick={() => showMastodonModal = true}>Connect</Button>
					</div>
				</CardContent>
			</Card>
			
			<Card class="opacity-60">
				<CardContent class="flex items-center justify-between p-4">
					<div class="flex items-center gap-3">
						<div class="w-10 h-10 bg-orange-500 rounded-full flex items-center justify-center">
							<span class="text-white font-bold text-lg">📸</span>
						</div>
						<div>
							<h3 class="font-medium">Threads</h3>
							<p class="text-sm text-muted-foreground">Coming soon</p>
						</div>
					</div>
					<Button disabled>Coming Soon</Button>
				</CardContent>
			</Card>
			
			<Card class="opacity-60">
				<CardContent class="flex items-center justify-between p-4">
					<div class="flex items-center gap-3">
						<div class="w-10 h-10 bg-sky-500 rounded-full flex items-center justify-center">
							<span class="text-white font-bold text-lg">🦋</span>
						</div>
						<div>
							<h3 class="font-medium">Bluesky</h3>
							<p class="text-sm text-muted-foreground">Coming soon</p>
						</div>
					</div>
					<Button disabled>Coming Soon</Button>
				</CardContent>
			</Card>
			
			<Card class="opacity-60">
				<CardContent class="flex items-center justify-between p-4">
					<div class="flex items-center gap-3">
						<div class="w-10 h-10 bg-blue-600 rounded-full flex items-center justify-center">
							<span class="text-white font-bold text-lg">💼</span>
						</div>
						<div>
							<h3 class="font-medium">LinkedIn</h3>
							<p class="text-sm text-muted-foreground">Coming soon</p>
						</div>
					</div>
					<Button disabled>Coming Soon</Button>
				</CardContent>
			</Card>
		</div>
	</div>
{/if}

{#if showMastodonModal}
	<div class="fixed inset-0 bg-black/50 flex items-center justify-center z-50" onclick={() => showMastodonModal = false}>
		<Card class="w-full max-w-md mx-4" onclick={(e: MouseEvent) => e.stopPropagation()}>
			<CardHeader>
				<CardTitle>Connect Mastodon</CardTitle>
			</CardHeader>
			<CardContent>
				<form onsubmit={(e) => { e.preventDefault(); connectMastodon(); }} class="space-y-4">
					<div class="space-y-2">
						<Label for="instance">Mastodon Instance</Label>
						<Input
							type="text"
							id="instance"
							bind:value={mastodonInstance}
							placeholder="mastodon.social"
							required
						/>
						<p class="text-xs text-muted-foreground">Enter your Mastodon instance URL (e.g., mastodon.social, fosstodon.org)</p>
					</div>
					<div class="flex gap-3 justify-end">
						<Button type="button" variant="outline" onclick={() => showMastodonModal = false}>Cancel</Button>
						<Button type="submit">Connect</Button>
					</div>
				</form>
			</CardContent>
		</Card>
	</div>
{/if}
