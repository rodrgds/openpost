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

	let code = $state('');
	let instance = $state('');
	let workspaceId = $state('');
	let loading = $state(false);
	let error = $state('');
	let success = $state(false);

	onMount(() => {
		const params = new URLSearchParams(window.location.search);
		const storedWorkspace = localStorage.getItem('oauth_workspace_id');
		const storedInstance = localStorage.getItem('oauth_mastodon_instance');

		if (storedWorkspace) workspaceId = storedWorkspace;
		if (storedInstance) instance = storedInstance;

		const codeFromUrl = params.get('code');
		if (codeFromUrl) {
			code = codeFromUrl;
		}
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
		if (!instance) {
			instance = 'https://mastodon.social';
		}

		loading = true;
		error = '';

		try {
			const { error: err } = await client.POST('/accounts/mastodon/exchange', {
				body: { workspace_id: workspaceId, instance: instance, code: code.trim() }
			});
			if (err) throw new Error(err.detail || 'Exchange failed');
			localStorage.removeItem('oauth_workspace_id');
			localStorage.removeItem('oauth_mastodon_instance');
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

<div class="mx-auto max-w-md px-4 py-12">
	{#if success}
		<Card>
			<CardContent class="pt-6 text-center">
				<div class="mb-4 text-5xl text-green-600">✓</div>
				<CardTitle class="mb-2">Account Connected!</CardTitle>
				<CardDescription>Redirecting to accounts...</CardDescription>
			</CardContent>
		</Card>
	{:else}
		<Card>
			<CardHeader>
				<CardTitle>Connect Mastodon Account</CardTitle>
				<CardDescription>Paste the authorization code from Mastodon below:</CardDescription>
			</CardHeader>
			<CardContent>
				<form
					onsubmit={(e) => {
						e.preventDefault();
						submitCode();
					}}
					class="space-y-4"
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
	{/if}
</div>
