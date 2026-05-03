<script lang="ts">
	import { auth } from '$lib/stores/auth';
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
	import Logo from '$lib/components/Logo.svelte';
	import LoaderIcon from 'lucide-svelte/icons/loader-2';
	import ShieldIcon from 'lucide-svelte/icons/shield';
	import KeyRoundIcon from 'lucide-svelte/icons/key-round';
	import { m } from '$lib/paraglide/messages';

	let email = $state('');
	let password = $state('');
	let totpCode = $state('');
	let error = $state('');
	let isLoading = $state(false);
	let mfaToken = $state('');
	let mfaMethods = $state<string[]>([]);

	const needsMfa = $derived(mfaToken.length > 0);

	async function handleSubmit(e: Event) {
		e.preventDefault();
		error = '';
		isLoading = true;

		const result = await auth.login(email, password);

		if (result.success) {
			goto('/');
		} else if (result.requiresMfa && result.mfaToken) {
			mfaToken = result.mfaToken;
			mfaMethods = result.mfaMethods ?? [];
			totpCode = '';
		} else {
			error = result.error || m.auth_login_failed();
		}

		isLoading = false;
	}

	async function handleVerifyTOTP(e: Event) {
		e.preventDefault();
		error = '';
		isLoading = true;

		const result = await auth.verifyTOTP(mfaToken, totpCode);
		if (result.success) {
			goto('/');
		} else {
			error = result.error || 'Authenticator verification failed';
		}

		isLoading = false;
	}

	async function handleVerifyPasskey() {
		error = '';
		isLoading = true;

		const result = await auth.verifyPasskey(mfaToken);
		if (result.success) {
			goto('/');
		} else {
			error = result.error || 'Passkey verification failed';
		}

		isLoading = false;
	}

	function resetMfa() {
		mfaToken = '';
		mfaMethods = [];
		totpCode = '';
		error = '';
	}
</script>

<svelte:head>
	<title>{m.auth_login_title()}</title>
</svelte:head>

<div class="flex min-h-[80vh] flex-col items-center justify-center gap-6">
	<div class="flex justify-center">
		<a href="/">
			<Logo width={80} height={23} />
		</a>
	</div>
	<Card class="w-full max-w-md">
		<CardHeader>
			<CardTitle class="text-center text-lg font-semibold">
				{needsMfa ? 'Verify your identity' : m.auth_login_heading()}
			</CardTitle>
			<CardDescription class="text-center">
				{#if needsMfa}
					Use your authenticator app or a saved passkey to finish signing in.
				{:else}
					{m.auth_login_description()}
				{/if}
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

			{#if needsMfa}
				<div class="space-y-4">
					{#if mfaMethods.includes('passkey')}
						<Button
							type="button"
							class="w-full gap-2"
							onclick={handleVerifyPasskey}
							disabled={isLoading}
						>
							<KeyRoundIcon class="h-4 w-4" />
							{#if isLoading}
								<LoaderIcon class="h-4 w-4 animate-spin" />
								Checking passkey...
							{:else}
								Use Passkey
							{/if}
						</Button>
					{/if}

					{#if mfaMethods.includes('totp')}
						<form onsubmit={handleVerifyTOTP} class="space-y-4">
							<div class="space-y-2">
								<Label for="totpCode">Authenticator code</Label>
								<Input
									id="totpCode"
									bind:value={totpCode}
									inputmode="numeric"
									autocomplete="one-time-code"
									pattern="[0-9]*"
									maxlength={6}
									placeholder="123456"
									required
								/>
							</div>

							<Button type="submit" disabled={isLoading} class="w-full gap-2">
								<ShieldIcon class="h-4 w-4" />
								{#if isLoading}
									<LoaderIcon class="h-4 w-4 animate-spin" />
									Verifying...
								{:else}
									Verify Code
								{/if}
							</Button>
						</form>
					{/if}

					<Button
						type="button"
						variant="ghost"
						class="w-full"
						onclick={resetMfa}
						disabled={isLoading}
					>
						Use a different account
					</Button>
				</div>
			{:else}
				<form onsubmit={handleSubmit} class="space-y-4">
					<div class="space-y-2">
						<Label for="email">{m.common_email()}</Label>
						<Input
							type="email"
							id="email"
							bind:value={email}
							required
							placeholder="you@example.com"
						/>
					</div>

					<div class="space-y-2">
						<Label for="password">{m.common_password()}</Label>
						<Input
							type="password"
							id="password"
							bind:value={password}
							required
							placeholder="••••••••"
						/>
					</div>

					<Button type="submit" disabled={isLoading} class="w-full">
						{#if isLoading}
							<LoaderIcon class="mr-2 h-4 w-4 animate-spin" />
							{m.auth_login_loading()}
						{:else}
							{m.auth_login_submit()}
						{/if}
					</Button>
				</form>

				<p class="mt-6 text-center text-sm text-muted-foreground">
					{m.auth_login_no_account()}
					<a href="/register" class="font-medium text-primary hover:underline"
						>{m.auth_login_create_one()}</a
					>
				</p>
			{/if}
		</CardContent>
	</Card>
</div>
