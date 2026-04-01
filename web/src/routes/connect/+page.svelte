<script lang="ts">
	import { goto } from '$app/navigation';
	import { instanceStore, isInstanceConfigured } from '$lib/stores/instance.svelte';
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
	import Logo from '$lib/components/Logo.svelte';

	const instance = instanceStore();

	let serverUrl = $state('');
	let error = $state('');
	let isConnecting = $state(false);

	function handleSubmit(e: Event) {
		e.preventDefault();
		connect();
	}

	async function connect() {
		if (!serverUrl.trim()) {
			error = 'Please enter your server URL';
			return;
		}

		error = '';
		isConnecting = true;

		const result = await instance.setInstanceUrl(serverUrl);

		if (result.success) {
			goto('/login');
		} else {
			error = result.error || 'Could not connect to server';
		}

		isConnecting = false;
	}
</script>

<svelte:head>
	<title>Connect - OpenPost</title>
</svelte:head>

<div class="flex min-h-[80vh] flex-col items-center justify-center gap-6">
	<div class="flex justify-center">
		<Logo width={100} height={29} />
	</div>
	<Card class="w-full max-w-md">
		<CardHeader>
			<CardTitle class="text-center">Connect to Server</CardTitle>
			<CardDescription class="text-center">
				Enter the URL of your self-hosted OpenPost instance
			</CardDescription>
		</CardHeader>
		<CardContent>
			{#if error}
				<div
					class="mb-4 rounded-md border border-destructive/20 bg-destructive/10 p-3 text-sm text-destructive"
				>
					{error}
				</div>
			{/if}

			<form onsubmit={handleSubmit} class="space-y-4">
				<div class="space-y-2">
					<Label for="server-url">Server URL</Label>
					<Input
						type="url"
						id="server-url"
						bind:value={serverUrl}
						required
						placeholder="https://openpost.example.com"
						disabled={isConnecting}
					/>
					<p class="text-xs text-muted-foreground">
						The address where your OpenPost server is running
					</p>
				</div>

				<Button type="submit" disabled={isConnecting} class="w-full">
					{isConnecting ? 'Connecting...' : 'Connect'}
				</Button>
			</form>
		</CardContent>
	</Card>
</div>
