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
	import * as Dialog from '$lib/components/ui/dialog';
	import { Input } from '$lib/components/ui/input';
	import { Label } from '$lib/components/ui/label';
	import * as DropdownMenu from '$lib/components/ui/dropdown-menu';
	import { Checkbox } from '$lib/components/ui/checkbox';
	import { goto } from '$app/navigation';
	import FolderOpenIcon from 'lucide-svelte/icons/folder-open';
	import ChevronDownIcon from 'lucide-svelte/icons/chevron-down';
	import LayersIcon from 'lucide-svelte/icons/layers';
	import PlusIcon from 'lucide-svelte/icons/plus';
	import TrashIcon from 'lucide-svelte/icons/trash-2';
	import { getPlatformName, getPlatformColor } from '$lib/utils';
	import PlatformIcon from '$lib/components/platform-icon.svelte';
	import LoaderIcon from 'lucide-svelte/icons/loader-2';
	import SettingsIcon from 'lucide-svelte/icons/settings';

	interface MastodonServer {
		name: string;
		instance_url: string;
	}

	interface SetAccount {
		social_account_id: string;
		platform: string;
		account_username: string;
		is_main: boolean;
	}

	interface SocialMediaSet {
		id: string;
		workspace_id: string;
		name: string;
		is_default: boolean;
		created_at: string;
		accounts: SetAccount[];
	}

	let workspaces = $state<Workspace[] | null>(null);
	let selectedWorkspaceId = $state('');
	let loading = $state(true);
	let error = $state('');

	let accounts = $state<SocialAccount[]>([]);
	let accountsLoading = $state(false);

	let sets = $state<SocialMediaSet[]>([]);
	let setsLoading = $state(false);

	let mastodonServers = $state<MastodonServer[]>([]);
	let selectedWorkspaceName = $derived(
		workspaces?.find((workspace) => workspace.id === selectedWorkspaceId)?.name ||
			'Select workspace'
	);

	let blueskyModalOpen = $state(false);
	let blueskyHandle = $state('');
	let blueskyAppPassword = $state('');
	let blueskyLoading = $state(false);
	let blueskyError = $state('');

	let createSetDialogOpen = $state(false);
	let newSetName = $state('');
	let newSetDefault = $state(false);
	let newSetAccountIds = $state<string[]>([]);
	let createSetLoading = $state(false);

	let editSetDialogOpen = $state(false);
	let editingSet = $state<SocialMediaSet | null>(null);
	let editSetName = $state('');
	let editSetDefault = $state(false);
	let editSetAccountIds = $state<string[]>([]);

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

	async function loadSets() {
		if (!selectedWorkspaceId) return;
		setsLoading = true;
		try {
			const { data, error: err } = await client.GET('/sets', {
				params: { query: { workspace_id: selectedWorkspaceId } }
			});
			sets = (data ?? []) as unknown as SocialMediaSet[];
		} catch (e) {
			console.error('Failed to load sets:', e);
			sets = [];
		} finally {
			setsLoading = false;
		}
	}

	async function loadMastodonServers() {
		try {
			const { data } = await client.GET('/accounts/mastodon/servers', {});
			mastodonServers = (data ?? []) as unknown as MastodonServer[];
		} catch (e) {
			console.error('Failed to load Mastodon servers:', e);
			mastodonServers = [];
		}
	}

	async function disconnectAccount(accountId: string) {
		try {
			await (client as any).DELETE('/accounts/{account_id}', {
				params: { path: { account_id: accountId } }
			});
			await loadAccounts();
			await loadSets();
		} catch (e) {
			error = (e as Error).message;
		}
	}

	async function createSet() {
		if (!newSetName.trim() || !selectedWorkspaceId) return;
		createSetLoading = true;
		try {
			await (client as any).POST('/sets', {
				body: {
					workspace_id: selectedWorkspaceId,
					name: newSetName.trim(),
					is_default: newSetDefault,
					account_ids: newSetAccountIds
				}
			});
			createSetDialogOpen = false;
			newSetName = '';
			newSetDefault = false;
			newSetAccountIds = [];
			await loadSets();
		} catch (e) {
			error = (e as Error).message;
		} finally {
			createSetLoading = false;
		}
	}

	async function updateSet() {
		if (!editingSet || !editSetName.trim()) return;
		createSetLoading = true;
		try {
			await (client as any).PATCH('/sets/{id}', {
				params: { path: { id: editingSet.id } },
				body: {
					name: editSetName.trim(),
					is_default: editSetDefault
				}
			});

			const currentAccIds = editingSet.accounts.map((a) => a.social_account_id);
			const toAdd = editSetAccountIds.filter((id) => !currentAccIds.includes(id));
			const toRemove = currentAccIds.filter((id) => !editSetAccountIds.includes(id));

			for (const accId of toAdd) {
				await (client as any).POST('/sets/{id}/accounts', {
					params: { path: { id: editingSet.id } },
					body: { account_ids: [accId] }
				});
			}

			for (const accId of toRemove) {
				await (client as any).DELETE('/sets/{id}/accounts/{account_id}', {
					params: { path: { id: editingSet.id, account_id: accId } }
				});
			}

			editSetDialogOpen = false;
			editingSet = null;
			await loadSets();
		} catch (e) {
			error = (e as Error).message;
		} finally {
			createSetLoading = false;
		}
	}

	async function deleteSet(setId: string) {
		if (!confirm('Delete this set? Posts using this set will keep their account selections.'))
			return;
		try {
			await (client as any).DELETE('/sets/{id}', {
				params: { path: { id: setId } }
			});
			await loadSets();
		} catch (e) {
			error = (e as Error).message;
		}
	}

	function openEditSet(set: SocialMediaSet) {
		editingSet = set;
		editSetName = set.name;
		editSetDefault = set.is_default;
		editSetAccountIds = set.accounts.map((a) => a.social_account_id);
		editSetDialogOpen = true;
	}

	onMount(() => {
		const params = new URLSearchParams(window.location.search);
		const urlError = params.get('error');
		if (urlError) {
			error = urlError;
			window.history.replaceState({}, document.title, window.location.pathname);
		}

		const unsubscribe = auth.subscribe(async (state) => {
			if (!state.isLoading && !state.isAuthenticated) {
				goto('/login');
			} else if (!state.isLoading && state.isAuthenticated) {
				try {
					const { data, error: err } = await client.GET('/workspaces');
					workspaces = data ?? [];
					if (workspaces && workspaces.length > 0) {
						selectedWorkspaceId = workspaces[0].id;
						await loadAccounts();
						await loadSets();
					}
					await loadMastodonServers();
				} catch (e) {
					console.error('Failed to load workspaces:', e);
				} finally {
					loading = false;
				}
			}
		});
		return unsubscribe;
	});

	$effect(() => {
		if (selectedWorkspaceId) {
			loadAccounts();
			loadSets();
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
			if (err) throw new Error((err as any).detail || 'Failed to get X auth URL');
			if (data?.url) window.location.href = data.url;
		} catch (e) {
			error = (e as Error).message;
		}
	}

	async function connectMastodon(serverName: string) {
		if (!selectedWorkspaceId) {
			alert('Please create a workspace first');
			return;
		}

		try {
			localStorage.setItem('oauth_workspace_id', selectedWorkspaceId);
			localStorage.setItem('oauth_mastodon_server', serverName);

			const { data, error: err } = await client.GET('/accounts/{platform}/auth-url', {
				params: {
					path: { platform: 'mastodon' },
					query: { workspace_id: selectedWorkspaceId, server_name: serverName }
				}
			});
			if (data?.url) window.location.href = data.url;
		} catch (e) {
			error = (e as Error).message;
		}
	}

	async function connectBluesky() {
		if (!selectedWorkspaceId) {
			alert('Please create a workspace first');
			return;
		}
		blueskyHandle = '';
		blueskyAppPassword = '';
		blueskyError = '';
		blueskyModalOpen = true;
	}

	async function submitBlueskyLogin() {
		if (!blueskyHandle.trim() || !blueskyAppPassword.trim()) {
			blueskyError = 'Please enter both handle and app password';
			return;
		}

		blueskyLoading = true;
		blueskyError = '';

		try {
			const { error: err } = await (client as any).POST('/accounts/bluesky/login', {
				body: {
					workspace_id: selectedWorkspaceId,
					handle: blueskyHandle.trim(),
					app_password: blueskyAppPassword.trim()
				}
			});
			if (err) throw new Error(err.detail || 'Login failed');
			blueskyModalOpen = false;
			await loadAccounts();
			await loadSets();
		} catch (e) {
			blueskyError = (e as Error).message;
		} finally {
			blueskyLoading = false;
		}
	}

	async function connectLinkedIn() {
		if (!selectedWorkspaceId) {
			alert('Please create a workspace first');
			return;
		}

		try {
			localStorage.setItem('oauth_workspace_id', selectedWorkspaceId);

			const { data, error: err } = await client.GET('/accounts/{platform}/auth-url', {
				params: {
					path: { platform: 'linkedin' },
					query: { workspace_id: selectedWorkspaceId }
				}
			});
			if (data?.url) window.location.href = data.url;
		} catch (e) {
			error = (e as Error).message;
		}
	}

	async function connectThreads() {
		if (!selectedWorkspaceId) {
			alert('Please create a workspace first');
			return;
		}

		try {
			localStorage.setItem('oauth_workspace_id', selectedWorkspaceId);

			const { data, error: err } = await client.GET('/accounts/{platform}/auth-url', {
				params: {
					path: { platform: 'threads' },
					query: { workspace_id: selectedWorkspaceId }
				}
			});
			if (data?.url) window.location.href = data.url;
		} catch (e) {
			error = (e as Error).message;
		}
	}

	function toggleNewSetAccount(accId: string) {
		if (newSetAccountIds.includes(accId)) {
			newSetAccountIds = newSetAccountIds.filter((id) => id !== accId);
		} else {
			newSetAccountIds = [...newSetAccountIds, accId];
		}
	}

	function toggleEditSetAccount(accId: string) {
		if (editSetAccountIds.includes(accId)) {
			editSetAccountIds = editSetAccountIds.filter((id) => id !== accId);
		} else {
			editSetAccountIds = [...editSetAccountIds, accId];
		}
	}

	const accountsByPlatform = $derived.by(() => {
		const grouped = new Map<string, SocialAccount[]>();
		for (const acc of accounts) {
			const key = acc.platform;
			if (!grouped.has(key)) grouped.set(key, []);
			grouped.get(key)!.push(acc);
		}
		return grouped;
	});
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
		<h1 class="mb-6 text-2xl font-bold">Accounts & Sets</h1>

		{#if error}
			<div
				class="mb-4 rounded-md border border-destructive/20 bg-destructive/10 p-3 text-destructive"
			>
				{error}
				<Button variant="ghost" size="sm" onclick={() => (error = '')}>Dismiss</Button>
			</div>
		{/if}

		<div class="mb-6">
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

		<div class="mb-8 space-y-4">
			<div class="flex items-center justify-between">
				<h2 class="text-lg font-medium">Social Media Sets</h2>
				<Button onclick={() => (createSetDialogOpen = true)} size="sm" class="gap-1">
					<PlusIcon class="h-4 w-4" />
					New Set
				</Button>
			</div>

			{#if setsLoading}
				<div class="flex justify-center py-4">
					<LoaderIcon class="h-6 w-6 animate-spin text-primary" />
				</div>
			{:else if sets.length === 0}
				<div class="rounded-md border border-dashed bg-muted/50 p-6 text-center">
					<LayersIcon class="mx-auto mb-2 h-8 w-8 text-muted-foreground" />
					<p class="mb-1 text-sm font-medium">No sets yet</p>
					<p class="mb-3 text-xs text-muted-foreground">
						Create sets to group accounts for quick posting
					</p>
					<Button onclick={() => (createSetDialogOpen = true)} size="sm">
						Create your first set
					</Button>
				</div>
			{:else}
				<div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
					{#each sets as set}
						<Card class="relative">
							<CardContent class="p-4">
								<div class="mb-3 flex items-start justify-between">
									<div class="flex items-center gap-2">
										<div
											class="flex h-10 w-10 items-center justify-center rounded-full bg-primary/10"
										>
											<LayersIcon class="h-5 w-5 text-primary" />
										</div>
										<div>
											<h3 class="font-medium">{set.name}</h3>
											<p class="text-xs text-muted-foreground">
												{set.accounts.length} account{set.accounts.length !== 1 ? 's' : ''}
											</p>
										</div>
									</div>
									<div class="flex items-center gap-1">
										{#if set.is_default}
											<span class="rounded bg-primary/10 px-2 py-0.5 text-xs text-primary">
												Default
											</span>
										{/if}
										<Button
											variant="ghost"
											size="icon"
											class="h-8 w-8"
											onclick={() => openEditSet(set)}
										>
											<SettingsIcon class="h-4 w-4" />
										</Button>
										<Button
											variant="ghost"
											size="icon"
											class="h-8 w-8 text-destructive hover:text-destructive"
											onclick={() => deleteSet(set.id)}
										>
											<TrashIcon class="h-4 w-4" />
										</Button>
									</div>
								</div>
								{#if set.accounts.length > 0}
									<div class="flex flex-wrap gap-1">
										{#each set.accounts as acc}
											<span
												class="inline-flex items-center gap-1 rounded-full bg-muted px-2 py-0.5 text-xs"
											>
												<PlatformIcon platform={acc.platform} class="h-3 w-3" />
												{acc.account_username || acc.platform}
											</span>
										{/each}
									</div>
								{:else}
									<p class="text-xs text-muted-foreground">No accounts in this set</p>
								{/if}
							</CardContent>
						</Card>
					{/each}
				</div>
			{/if}
		</div>

		<div class="mb-8">
			<h2 class="mb-4 text-lg font-medium">Connected Accounts</h2>

			{#if accountsLoading}
				<div class="flex justify-center py-4">
					<LoaderIcon class="h-6 w-6 animate-spin text-primary" />
				</div>
			{:else if !accounts || accounts.length === 0}
				<div class="rounded-md border bg-muted/50 p-4 text-center text-muted-foreground">
					No accounts connected yet. Connect a platform below to get started.
				</div>
			{:else}
				<div class="space-y-4">
					{#each [...accountsByPlatform.entries()] as [platform, platformAccounts]}
						<Card>
							<CardContent class="p-4">
								<div class="mb-3 flex items-center gap-3">
									<div
										class="h-10 w-10 {getPlatformColor(
											platform
										)} flex items-center justify-center rounded-full"
									>
										<PlatformIcon {platform} class="h-5 w-5 text-white" />
									</div>
									<div>
										<h3 class="font-medium capitalize">{getPlatformName(platform)}</h3>
										<p class="text-sm text-muted-foreground">
											{platformAccounts.length} account{platformAccounts.length !== 1 ? 's' : ''}
										</p>
									</div>
								</div>
								<div class="space-y-2">
									{#each platformAccounts as account}
										<div class="flex items-center justify-between rounded-md border p-3">
											<div class="flex items-center gap-3">
												<div class="flex h-8 w-8 items-center justify-center rounded-full bg-muted">
													<PlatformIcon platform={account.platform} class="h-4 w-4" />
												</div>
												<div>
													<p class="font-medium">
														{#if account.account_username}
															@{account.account_username}
														{:else if account.instance_url}
															{account.instance_url.replace('https://', '')}
														{:else}
															{account.account_id}
														{/if}
													</p>
													<p class="text-xs text-muted-foreground">
														{account.is_active ? 'Connected' : 'Disconnected'}
													</p>
												</div>
											</div>
											<Button
												variant="outline"
												size="sm"
												onclick={() => disconnectAccount(account.id)}
											>
												Disconnect
											</Button>
										</div>
									{/each}
								</div>
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
							<PlatformIcon platform="x" class="h-4 w-4 text-white" />
						</div>
						<div>
							<h3 class="font-medium">X (Twitter)</h3>
							<p class="text-sm text-muted-foreground">Connect your X account to post tweets</p>
						</div>
					</div>
					<Button onclick={connectTwitter}>Connect</Button>
				</CardContent>
			</Card>

			{#if mastodonServers.length > 0}
				{#each mastodonServers as server}
					<Card>
						<CardContent class="flex items-center justify-between p-4">
							<div class="flex items-center gap-3">
								<div class="flex h-10 w-10 items-center justify-center rounded-full bg-indigo-500">
									<PlatformIcon platform="mastodon" class="h-4 w-4 text-white" />
								</div>
								<div>
									<h3 class="font-medium">{server.name}</h3>
									<p class="text-sm text-muted-foreground">
										{server.instance_url.replace('https://', '')}
									</p>
								</div>
							</div>
							<div class="flex gap-2">
								<Button href="/accounts/mastodon/callback" variant="outline">Enter Code</Button>
								<Button onclick={() => connectMastodon(server.name)}>Connect</Button>
							</div>
						</CardContent>
					</Card>
				{/each}
			{:else}
				<Card>
					<CardContent class="flex items-center justify-between p-4">
						<div class="flex items-center gap-3">
							<div class="flex h-10 w-10 items-center justify-center rounded-full bg-indigo-500">
								<PlatformIcon platform="mastodon" class="h-5 w-5 text-white" />
							</div>
							<div>
								<h3 class="font-medium">Mastodon</h3>
								<p class="text-sm text-muted-foreground">
									No Mastodon servers configured. Add MASTODON_SERVERS to your .env file.
								</p>
							</div>
						</div>
					</CardContent>
				</Card>
			{/if}

			<Card>
				<CardContent class="flex items-center justify-between p-4">
					<div class="flex items-center gap-3">
						<div class="flex h-10 w-10 items-center justify-center rounded-full bg-orange-500">
							<PlatformIcon platform="threads" class="h-5 w-5 text-white" />
						</div>
						<div>
							<h3 class="font-medium">Threads</h3>
							<p class="text-sm text-muted-foreground">Connect your Threads account</p>
						</div>
					</div>
					<Button onclick={connectThreads}>Connect</Button>
				</CardContent>
			</Card>

			<Card>
				<CardContent class="flex items-center justify-between p-4">
					<div class="flex items-center gap-3">
						<div class="flex h-10 w-10 items-center justify-center rounded-full bg-sky-500">
							<PlatformIcon platform="bluesky" class="h-5 w-5 text-white" />
						</div>
						<div>
							<h3 class="font-medium">Bluesky</h3>
							<p class="text-sm text-muted-foreground">Connect your Bluesky account</p>
						</div>
					</div>
					<Button onclick={connectBluesky}>Connect</Button>
				</CardContent>
			</Card>

			<Card>
				<CardContent class="flex items-center justify-between p-4">
					<div class="flex items-center gap-3">
						<div class="flex h-10 w-10 items-center justify-center rounded-full bg-blue-600">
							<PlatformIcon platform="linkedin" class="h-5 w-5 text-white" />
						</div>
						<div>
							<h3 class="font-medium">LinkedIn</h3>
							<p class="text-sm text-muted-foreground">Connect your LinkedIn account</p>
						</div>
					</div>
					<Button onclick={connectLinkedIn}>Connect</Button>
				</CardContent>
			</Card>
		</div>
	</div>
{/if}

<Dialog.Root bind:open={blueskyModalOpen}>
	<Dialog.Content class="sm:max-w-md">
		<Dialog.Header>
			<Dialog.Title>Connect Bluesky</Dialog.Title>
			<Dialog.Description>
				Enter your Bluesky handle and an app password. You can create an app password in Bluesky
				Settings &gt; App Passwords.
			</Dialog.Description>
		</Dialog.Header>
		<form
			onsubmit={(e) => {
				e.preventDefault();
				submitBlueskyLogin();
			}}
			class="space-y-4"
		>
			<div class="space-y-2">
				<Label for="bluesky-handle">Handle</Label>
				<Input
					type="text"
					id="bluesky-handle"
					bind:value={blueskyHandle}
					placeholder="user.bsky.social"
					required
				/>
			</div>
			<div class="space-y-2">
				<Label for="bluesky-password">App Password</Label>
				<Input
					type="password"
					id="bluesky-password"
					bind:value={blueskyAppPassword}
					placeholder="xxxx-xxxx-xxxx-xxxx"
					required
				/>
			</div>
			{#if blueskyError}
				<div
					class="rounded-md border border-destructive/20 bg-destructive/10 p-3 text-sm text-destructive"
				>
					{blueskyError}
				</div>
			{/if}
			<div class="flex justify-end gap-2">
				<Dialog.Close>
					<Button variant="outline" type="button">Cancel</Button>
				</Dialog.Close>
				<Button type="submit" disabled={blueskyLoading}>
					{blueskyLoading ? 'Connecting...' : 'Connect'}
				</Button>
			</div>
		</form>
	</Dialog.Content>
</Dialog.Root>

<Dialog.Root bind:open={createSetDialogOpen}>
	<Dialog.Content class="sm:max-w-md">
		<Dialog.Header>
			<Dialog.Title>Create Social Media Set</Dialog.Title>
			<Dialog.Description>
				Group accounts together for quick posting. Sets appear in the compose post dropdown.
			</Dialog.Description>
		</Dialog.Header>
		<form
			onsubmit={(e) => {
				e.preventDefault();
				createSet();
			}}
			class="space-y-4"
		>
			<div class="space-y-2">
				<Label for="set-name">Set Name</Label>
				<Input
					id="set-name"
					bind:value={newSetName}
					placeholder="e.g. Tech Twitter, Professional"
					required
				/>
			</div>
			<div class="flex items-center gap-2">
				<Checkbox id="set-default" bind:checked={newSetDefault} />
				<Label for="set-default" class="text-sm font-normal">
					Set as the default set for this workspace
				</Label>
			</div>
			{#if accounts.length > 0}
				<div class="space-y-2">
					<Label>Accounts to Include</Label>
					<div class="max-h-48 space-y-2 overflow-y-auto rounded-md border p-2">
						{#each accounts as account}
							<label class="flex items-center gap-2 rounded p-2 hover:bg-muted/50">
								<Checkbox
									checked={newSetAccountIds.includes(account.id)}
									onCheckedChange={() => toggleNewSetAccount(account.id)}
								/>
								<PlatformIcon platform={account.platform} class="h-4 w-4" />
								<span class="text-sm">
									{#if account.account_username}
										@{account.account_username}
									{:else if account.instance_url}
										{account.instance_url.replace('https://', '')}
									{:else}
										{account.platform}
									{/if}
								</span>
							</label>
						{/each}
					</div>
				</div>
			{/if}
			<div class="flex justify-end gap-2">
				<Dialog.Close>
					<Button variant="outline" type="button">Cancel</Button>
				</Dialog.Close>
				<Button type="submit" disabled={createSetLoading || !newSetName.trim()}>
					{createSetLoading ? 'Creating...' : 'Create Set'}
				</Button>
			</div>
		</form>
	</Dialog.Content>
</Dialog.Root>

<Dialog.Root bind:open={editSetDialogOpen}>
	<Dialog.Content class="sm:max-w-md">
		<Dialog.Header>
			<Dialog.Title>Edit Social Media Set</Dialog.Title>
			<Dialog.Description>Update the set name, default status, and accounts.</Dialog.Description>
		</Dialog.Header>
		{#if editingSet}
			<form
				onsubmit={(e) => {
					e.preventDefault();
					updateSet();
				}}
				class="space-y-4"
			>
				<div class="space-y-2">
					<Label for="edit-set-name">Set Name</Label>
					<Input
						id="edit-set-name"
						bind:value={editSetName}
						placeholder="e.g. Tech Twitter, Professional"
						required
					/>
				</div>
				<div class="flex items-center gap-2">
					<Checkbox id="edit-set-default" bind:checked={editSetDefault} />
					<Label for="edit-set-default" class="text-sm font-normal">
						Set as the default set for this workspace
					</Label>
				</div>
				{#if accounts.length > 0}
					<div class="space-y-2">
						<Label>Accounts in Set</Label>
						<div class="max-h-48 space-y-2 overflow-y-auto rounded-md border p-2">
							{#each accounts as account}
								<label class="flex items-center gap-2 rounded p-2 hover:bg-muted/50">
									<Checkbox
										checked={editSetAccountIds.includes(account.id)}
										onCheckedChange={() => toggleEditSetAccount(account.id)}
									/>
									<PlatformIcon platform={account.platform} class="h-4 w-4" />
									<span class="text-sm">
										{#if account.account_username}
											@{account.account_username}
										{:else if account.instance_url}
											{account.instance_url.replace('https://', '')}
										{:else}
											{account.platform}
										{/if}
									</span>
								</label>
							{/each}
						</div>
					</div>
				{/if}
				<div class="flex justify-end gap-2">
					<Dialog.Close>
						<Button variant="outline" type="button">Cancel</Button>
					</Dialog.Close>
					<Button type="submit" disabled={createSetLoading || !editSetName.trim()}>
						{createSetLoading ? 'Saving...' : 'Save Changes'}
					</Button>
				</div>
			</form>
		{/if}
	</Dialog.Content>
</Dialog.Root>
