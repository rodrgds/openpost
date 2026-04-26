<script lang="ts">
	import { onMount } from 'svelte';
	import { client } from '$lib/api/client';
	import { goto } from '$app/navigation';
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
	import PageContainer from '$lib/components/page-container.svelte';
	import { Skeleton } from '$lib/components/ui/skeleton/index.js';

	let code = $state('');
	let serverName = $state('');
	let workspaceId = $state('');
	let loading = $state(false);
	let error = $state('');
	let success = $state(false);
	let pageLoading = $state(true);

	onMount(() => {
		const params = new URLSearchParams(window.location.search);
		const storedWorkspace = localStorage.getItem('oauth_workspace_id');
		const storedServer = localStorage.getItem('oauth_mastodon_server');

		if (storedWorkspace) workspaceId = storedWorkspace;
		if (storedServer) serverName = storedServer;

		const codeFromUrl = params.get('code');
		if (codeFromUrl) {
			code = codeFromUrl;
		}
		pageLoading = false;
	});

	async function submitCode() {
		if (!code.trim()) {
			error = 'Please enter the authorization code';
			return;
		}
		if (!workspaceId) {
			error = 'Workspace ID not found. Please start the connection from the accounts page.';
			return;
		}
		if (!serverName) {
			error = 'Server name not found. Please start the connection from the accounts page.';
			return;
		}

		loading = true;
		error = '';

		try {
			const { error: err } = await client.POST('/accounts/mastodon/exchange', {
				body: { workspace_id: workspaceId, server_name: serverName, code: code.trim() }
			});
			if (err) throw new Error(err.detail || 'Exchange failed');
			localStorage.removeItem('oauth_workspace_id');
			localStorage.removeItem('oauth_mastodon_server');
			success = true;
			setTimeout(() => goto('/accounts'), 2000);
		} catch (e) {
			error = (e as Error).message;
		} finally {
			loading = false;
		}
	}
</script>

<svelte:head>
	<title>Mastodon Callback - OpenPost</title>
</svelte:head>

{#if pageLoading}
	<div class="flex flex-1 flex-col items-center justify-center gap-3">
		<Skeleton class="h-10 w-10 rounded-full" />
		<Skeleton class="h-4 w-40" />
		<Skeleton class="h-3 w-56" />
	</div>
{:else if success}
	<PageContainer title="Account Connected!">
		<Card>
			<CardContent class="pt-6 text-center">
				<div class="mb-4 text-5xl text-green-600">✓</div>
				<CardTitle class="mb-2">Success</CardTitle>
				<CardDescription>Redirecting to accounts...</CardDescription>
			</CardContent>
		</Card>
	</PageContainer>
{:else}
	<PageContainer
		title="Connect Mastodon Account"
		description="Paste the authorization code from Mastodon below"
	>
		<Card>
			<CardContent class="pt-6">
				{#if serverName}
					<p class="mb-4 text-sm text-muted-foreground">
						Connecting to server: <strong>{serverName}</strong>
					</p>
				{/if}
				<form
					class="space-y-4"
					onsubmit={(e: SubmitEvent) => {
						e.preventDefault();
						submitCode();
					}}
				>
					<div class="space-y-2">
						<Label for="code">Authorization Code</Label>
						<Input
							type="text"
							id="code"
							bind:value={code}
							placeholder="Paste code here..."
							class="font-mono"
							required
						/>
					</div>

					{#if error}
						<div
							class="rounded-md border border-destructive/20 bg-destructive/10 p-3 text-sm text-destructive"
						>
							{error}
						</div>
					{/if}

					<Button type="submit" disabled={loading} class="w-full">
						{loading ? 'Connecting...' : 'Connect Account'}
					</Button>
				</form>

				<div class="mt-4 text-center">
					<a href="/accounts" class="text-sm text-primary hover:underline">Cancel</a>
				</div>
			</CardContent>
		</Card>
	</PageContainer>
{/if}
