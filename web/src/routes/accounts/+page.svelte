<script lang="ts">
	import { onMount } from 'svelte';
	import { auth } from '$lib/stores/auth';
	import { client, type Workspace, type SocialAccount } from '$lib/api/client';
	import { Button } from '$lib/components/ui/button';
	import {
		Card,
		CardContent,
		CardHeader,
		CardTitle,
		CardDescription
	} from '$lib/components/ui/card';
	import { Input } from '$lib/components/ui/input';
	import { Label } from '$lib/components/ui/label';
	import * as DropdownMenu from '$lib/components/ui/dropdown-menu';
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
		workspaces?.find((workspace) => workspace.id === selectedWorkspaceId)?.name ||
			'Select workspace'
	);

	async function loadAccounts() {
		if (!selectedWorkspaceId) return;
		accountsLoading = true;
		try {
			const { data, error: err } = await client.GET('/accounts', {
				params: { query: { workspace_id: selectedWorkspaceId } }
			});
			accounts = data ?? [];
		} catch (e) {
			console.error('Failed to load accounts:', e);
			accounts = [];
		} finally {
			accountsLoading = false;
		}
	}

	onMount(() => {
		auth.subscribe(async (state) => {
			if (!state.isLoading && !state.isAuthenticated) {
				goto('/login');
			} else if (!state.isLoading && state.isAuthenticated) {
				try {
					const { data, error: err } = await client.GET('/workspaces');
					workspaces = data ?? [];
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
			const { data, error: err } = await client.GET('/accounts/{platform}/auth-url', {
				params: { path: { platform: 'x' }, query: { workspace_id: selectedWorkspaceId } }
			});
			if (data?.url) window.location.href = data.url;
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

			const { data, error: err } = await client.GET('/accounts/{platform}/auth-url', {
				params: {
					path: { platform: 'mastodon' },
					query: { workspace_id: selectedWorkspaceId, instance }
				}
			});
			if (data?.url) window.location.href = data.url;
		} catch (e) {
			error = (e as Error).message;
		}
	}

	function getPlatformIcon(platform: string): string {
		switch (platform) {
			case 'x':
				return '𝕏';
			case 'mastodon':
				return '🐘';
			case 'threads':
				return '📸';
			case 'bluesky':
				return '🦋';
			case 'linkedin':
				return '💼';
			default:
				return '?';
		}
	}

	function getPlatformColor(platform: string): string {
		switch (platform) {
			case 'x':
				return 'bg-black';
			case 'mastodon':
				return 'bg-indigo-500';
			case 'threads':
				return 'bg-orange-500';
			case 'bluesky':
				return 'bg-sky-500';
			case 'linkedin':
				return 'bg-blue-600';
			default:
				return 'bg-gray-500';
		}
	}
</script>

<svelte:head>
	<title>Connected Accounts - OpenPost</title>
</svelte:head>

{#if loading}
	<div class="flex justify-center py-12">
		<div class="h-8 w-8 animate-spin rounded-full border-b-2 border-primary"></div>
	</div>
{:else if !workspaces || workspaces.length === 0}
	<div class="mx-auto max-w-4xl px-4 py-8">
		<Card class="text-center">
			<CardContent class="pt-6">
				<CardTitle class="mb-2">No Workspaces Found</CardTitle>
				<CardDescription class="mb-4"
					>You need to create a workspace before connecting social accounts.</CardDescription
				>
				<Button href="/">Create Workspace</Button>
			</CardContent>
		</Card>
	</div>
{:else}
	<div class="mx-auto max-w-4xl px-4 py-8">
		<h1 class="mb-6 text-2xl font-bold">Connected Accounts</h1>

		{#if error}
			<div
				class="mb-4 rounded-md border border-destructive/20 bg-destructive/10 p-3 text-destructive"
			>
				{error}
				<Button variant="ghost" size="sm" onclick={() => (error = '')}>Dismiss</Button>
			</div>
		{/if}

		<div class="mb-6">
			<Label class="mb-2">Workspace</Label>
			<DropdownMenu.Root>
				<DropdownMenu.Trigger>
					{#snippet child({ props })}
						<Button {...props} variant="outline" class="w-full max-w-sm justify-between">
							<span class="flex items-center gap-2 truncate">
								<span
									class="inline-flex size-6 items-center justify-center rounded-md border border-border bg-muted/50"
								>
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
						<DropdownMenu.Item
							onSelect={() => (selectedWorkspaceId = workspace.id)}
							class="gap-2 p-2"
						>
							<span
								class="inline-flex size-6 items-center justify-center rounded-md border border-border text-[0.625rem] font-semibold"
							>
								{workspace.name.slice(0, 1).toUpperCase()}
							</span>
							<span class="truncate">{workspace.name}</span>
						</DropdownMenu.Item>
					{/each}
				</DropdownMenu.Content>
			</DropdownMenu.Root>
		</div>

		<div class="mb-8">
			<h2 class="mb-4 text-lg font-medium">Connected Accounts</h2>

			{#if accountsLoading}
				<div class="flex justify-center py-4">
					<div class="h-6 w-6 animate-spin rounded-full border-b-2 border-primary"></div>
				</div>
			{:else if !accounts || accounts.length === 0}
				<div class="rounded-md border bg-muted/50 p-4 text-center text-muted-foreground">
					No accounts connected yet. Connect a platform below to get started.
				</div>
			{:else}
				<div class="space-y-3">
					{#each accounts as account}
						<Card>
							<CardContent class="flex items-center justify-between p-4">
								<div class="flex items-center gap-3">
									<div
										class="h-10 w-10 {getPlatformColor(
											account.platform
										)} flex items-center justify-center rounded-full"
									>
										<span class="text-lg font-bold text-white"
											>{getPlatformIcon(account.platform)}</span
										>
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
								<span class="rounded bg-green-100 px-2 py-1 text-xs font-medium text-green-700"
									>Connected</span
								>
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
						<div class="flex h-10 w-10 items-center justify-center rounded-full bg-black">
							<span class="font-bold text-white">X</span>
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
						<div class="flex h-10 w-10 items-center justify-center rounded-full bg-indigo-500">
							<span class="text-lg font-bold text-white">🐘</span>
						</div>
						<div>
							<h3 class="font-medium">Mastodon</h3>
							<p class="text-sm text-muted-foreground">
								Connect your Mastodon account from any instance
							</p>
						</div>
					</div>
					<div class="flex gap-2">
						<Button href="/accounts/mastodon/callback" variant="outline">Enter Code</Button>
						<Button onclick={() => (showMastodonModal = true)}>Connect</Button>
					</div>
				</CardContent>
			</Card>

			<Card class="opacity-60">
				<CardContent class="flex items-center justify-between p-4">
					<div class="flex items-center gap-3">
						<div class="flex h-10 w-10 items-center justify-center rounded-full bg-orange-500">
							<span class="text-lg font-bold text-white">📸</span>
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
						<div class="flex h-10 w-10 items-center justify-center rounded-full bg-sky-500">
							<span class="text-lg font-bold text-white">🦋</span>
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
						<div class="flex h-10 w-10 items-center justify-center rounded-full bg-blue-600">
							<span class="text-lg font-bold text-white">💼</span>
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
	<div
		class="fixed inset-0 z-50 flex items-center justify-center bg-black/50"
		onclick={() => (showMastodonModal = false)}
	>
		<Card class="mx-4 w-full max-w-md" onclick={(e: MouseEvent) => e.stopPropagation()}>
			<CardHeader>
				<CardTitle>Connect Mastodon</CardTitle>
			</CardHeader>
			<CardContent>
				<form
					onsubmit={(e) => {
						e.preventDefault();
						connectMastodon();
					}}
					class="space-y-4"
				>
					<div class="space-y-2">
						<Label for="instance">Mastodon Instance</Label>
						<Input
							type="text"
							id="instance"
							bind:value={mastodonInstance}
							placeholder="mastodon.social"
							required
						/>
						<p class="text-xs text-muted-foreground">
							Enter your Mastodon instance URL (e.g., mastodon.social, fosstodon.org)
						</p>
					</div>
					<div class="flex justify-end gap-3">
						<Button type="button" variant="outline" onclick={() => (showMastodonModal = false)}
							>Cancel</Button
						>
						<Button type="submit">Connect</Button>
					</div>
				</form>
			</CardContent>
		</Card>
	</div>
{/if}
