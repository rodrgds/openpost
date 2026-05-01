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
	import { m } from '$lib/paraglide/messages';

	let email = $state('');
	let password = $state('');
	let error = $state('');
	let isLoading = $state(false);

	async function handleSubmit(e: Event) {
		e.preventDefault();
		error = '';
		isLoading = true;

		const result = await auth.login(email, password);

		if (result.success) {
			goto('/');
		} else {
			error = result.error || m.auth_login_failed();
		}

		isLoading = false;
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
			<CardTitle class="text-center text-lg font-semibold">{m.auth_login_heading()}</CardTitle>
			<CardDescription class="text-center">{m.auth_login_description()}</CardDescription>
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
						placeholder="••••••••"
					/>
				</div>

				<Button type="submit" disabled={isLoading} class="w-full">
					{isLoading ? m.auth_login_loading() : m.auth_login_submit()}
				</Button>
			</form>

			<p class="mt-6 text-center text-sm text-muted-foreground">
				{m.auth_login_no_account()}
				<a href="/register" class="font-medium text-primary hover:underline"
					>{m.auth_login_create_one()}</a
				>
			</p>
		</CardContent>
	</Card>
</div>
