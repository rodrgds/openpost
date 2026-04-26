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
			error = result.error || 'Login failed';
		}

		isLoading = false;
	}
</script>

<svelte:head>
	<title>Login - OpenPost</title>
</svelte:head>

<div class="flex min-h-[80vh] flex-col items-center justify-center gap-6">
	<div class="flex justify-center">
		<a href="/">
			<Logo width={80} height={23} />
		</a>
	</div>
	<Card class="w-full max-w-md">
		<CardHeader>
			<CardTitle class="text-center text-lg font-semibold">Sign In</CardTitle>
			<CardDescription class="text-center"
				>Enter your credentials to access your account</CardDescription
			>
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
					<Label for="email">Email</Label>
					<Input
						type="email"
						id="email"
						bind:value={email}
						required
						placeholder="you@example.com"
					/>
				</div>

				<div class="space-y-2">
					<Label for="password">Password</Label>
					<Input
						type="password"
						id="password"
						bind:value={password}
						required
						placeholder="••••••••"
					/>
				</div>

				<Button type="submit" disabled={isLoading} class="w-full">
					{isLoading ? 'Signing in...' : 'Sign In'}
				</Button>
			</form>

			<p class="mt-6 text-center text-sm text-muted-foreground">
				Don't have an account?
				<a href="/register" class="font-medium text-primary hover:underline">Create one</a>
			</p>
		</CardContent>
	</Card>
</div>
