<script lang="ts">
	import { goto } from '$app/navigation';
	import { onMount } from 'svelte';
	import PageContainer from '$lib/components/page-container.svelte';
	import { Button } from '$lib/components/ui/button';
	import {
		Card,
		CardContent,
		CardDescription,
		CardHeader,
		CardTitle
	} from '$lib/components/ui/card';
	import { getPlatformName } from '$lib/utils';

	let countdown = $state(5);
	let platform = $state('');
	let timeoutId: number | undefined;
	let intervalId: number | undefined;

	onMount(() => {
		const params = new URLSearchParams(window.location.search);
		platform = params.get('platform') ?? '';

		intervalId = window.setInterval(() => {
			if (countdown > 1) {
				countdown -= 1;
			}
		}, 1000);

		timeoutId = window.setTimeout(() => {
			goto('/accounts');
		}, 5000);

		return () => {
			if (intervalId) window.clearInterval(intervalId);
			if (timeoutId) window.clearTimeout(timeoutId);
		};
	});

	function goToAccounts() {
		if (intervalId) window.clearInterval(intervalId);
		if (timeoutId) window.clearTimeout(timeoutId);
		goto('/accounts');
	}

	let platformName = $derived(platform ? getPlatformName(platform) : 'social account');
</script>

<svelte:head>
	<title>Account Connected - OpenPost</title>
</svelte:head>

<div class="flex min-h-screen items-center justify-center px-4 py-10">
	<PageContainer
		title="Account connected"
		description={`Your ${platformName} account was connected successfully.`}
	>
		<div class="mx-auto max-w-xl">
			<Card class="border-border/60 shadow-sm">
				<CardHeader class="text-center">
					<div
						class="mx-auto flex h-16 w-16 items-center justify-center rounded-full bg-emerald-500/12 text-3xl text-emerald-600"
					>
						✓
					</div>
					<CardTitle class="text-2xl">Success</CardTitle>
					<CardDescription>
						Redirecting you back to accounts in {countdown} second{countdown === 1 ? '' : 's'}.
					</CardDescription>
				</CardHeader>
				<CardContent class="flex flex-col items-center gap-3 text-center">
					<p class="max-w-md text-sm text-muted-foreground">
						OpenPost finished the OAuth flow and saved the connected account. You will be taken back
						automatically.
					</p>
					<Button onclick={goToAccounts}>Go to accounts now</Button>
				</CardContent>
			</Card>
		</div>
	</PageContainer>
</div>
