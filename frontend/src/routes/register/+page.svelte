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
	import CheckCircleIcon from 'lucide-svelte/icons/check-circle-2';
	import { m } from '$lib/paraglide/messages';

	let email = $state('');
	let password = $state('');
	let confirmPassword = $state('');
	let error = $state('');
	let isLoading = $state(false);
	let registrationSuccess = $state(false);

	async function handleSubmit(e: Event) {
		e.preventDefault();
		error = '';

		if (password !== confirmPassword) {
			error = m.auth_register_password_mismatch();
			return;
		}

		if (password.length < 8) {
			error = m.auth_register_password_short();
			return;
		}

		isLoading = true;

		const result = await auth.register(email, password);

		if (result.success) {
			registrationSuccess = true;
			goto('/onboarding');
		} else {
			error = result.error || m.auth_register_failed();
			isLoading = false;
		}
	}
</script>

<svelte:head>
	<title>{m.auth_register_title()}</title>
</svelte:head>

{#if registrationSuccess}
	<div class="flex min-h-[80vh] flex-col items-center justify-center gap-6 px-4">
		<Logo width={80} height={23} />
		<div class="w-full max-w-md text-center">
			<div
				class="mx-auto mb-4 flex h-16 w-16 items-center justify-center rounded-full bg-emerald-500/10"
			>
				<CheckCircleIcon class="h-8 w-8 text-emerald-500" />
			</div>
			<h2 class="mb-2 text-xl font-semibold tracking-tight">{m.auth_register_success_title()}</h2>
			<p class="text-muted-foreground">{m.auth_register_success_description()}</p>
		</div>
	</div>
{:else}
	<div class="flex min-h-[80vh] flex-col items-center justify-center gap-6 px-4">
		<div class="flex justify-center">
			<a href="/">
				<Logo width={80} height={23} />
			</a>
		</div>
		<Card class="w-full max-w-md">
			<CardHeader>
				<CardTitle class="text-center text-lg font-semibold">{m.auth_register_heading()}</CardTitle>
				<CardDescription class="text-center">{m.auth_register_description()}</CardDescription>
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
							placeholder={m.auth_password_min_placeholder()}
						/>
					</div>

					<div class="space-y-2">
						<Label for="confirmPassword">{m.auth_confirm_password()}</Label>
						<Input
							type="password"
							id="confirmPassword"
							bind:value={confirmPassword}
							required
							placeholder={m.auth_password_confirm_placeholder()}
						/>
					</div>

					<Button type="submit" disabled={isLoading} class="w-full">
						{#if isLoading}
							<LoaderIcon class="mr-2 h-4 w-4 animate-spin" />
							{m.auth_register_loading()}
						{:else}
							{m.auth_register_submit()}
						{/if}
					</Button>
				</form>

				<p class="mt-6 text-center text-sm text-muted-foreground">
					{m.auth_register_have_account()}
					<a href="/login" class="font-medium text-primary hover:underline"
						>{m.auth_register_sign_in()}</a
					>
				</p>
			</CardContent>
		</Card>
	</div>
{/if}
