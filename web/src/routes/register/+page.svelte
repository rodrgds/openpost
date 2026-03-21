<script lang="ts">
	import { auth } from '$lib/stores/auth';
	import { goto } from '$app/navigation';
	import { Button } from '$lib/components/ui/button';
	import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '$lib/components/ui/card';
	import { Input } from '$lib/components/ui/input';
	import { Label } from '$lib/components/ui/label';
	import Logo from '$lib/components/Logo.svelte';
	
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
			error = 'Passwords do not match';
			return;
		}
		
		if (password.length < 8) {
			error = 'Password must be at least 8 characters';
			return;
		}
		
		isLoading = true;
		
		const result = await auth.register(email, password);
		
		if (result.success) {
			registrationSuccess = true;
			setTimeout(() => {
				goto('/');
			}, 100);
		} else {
			error = result.error || 'Registration failed';
			isLoading = false;
		}
	}
</script>

<svelte:head>
	<title>Register - OpenPost</title>
</svelte:head>

{#if registrationSuccess}
	<div class="min-h-[80vh] flex flex-col items-center justify-center gap-6">
		<div class="flex justify-center">
			<a href="/">
				<Logo width={80} height={23} />
			</a>
		</div>
		<Card class="w-full max-w-md">
			<CardContent class="pt-6 text-center">
				<div class="text-green-600 mb-4">
					<svg class="w-12 h-12 mx-auto" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7"></path>
					</svg>
				</div>
				<h2 class="text-xl font-bold text-foreground mb-2">Account Created!</h2>
				<p class="text-muted-foreground">Redirecting to dashboard...</p>
			</CardContent>
		</Card>
	</div>
{:else}
	<div class="min-h-[80vh] flex flex-col items-center justify-center gap-6">
		<div class="flex justify-center">
			<a href="/">
				<Logo width={80} height={23} />
			</a>
		</div>
		<Card class="w-full max-w-md">
			<CardHeader>
				<CardTitle class="text-center">Create Account</CardTitle>
				<CardDescription class="text-center">Enter your details to get started</CardDescription>
			</CardHeader>
			<CardContent>
				{#if error}
					<div class="mb-4 p-3 bg-destructive/10 border border-destructive/20 rounded-md text-destructive text-sm">
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
						<p class="text-xs text-muted-foreground">At least 8 characters</p>
					</div>
					
					<div class="space-y-2">
						<Label for="confirmPassword">Confirm Password</Label>
						<Input
							type="password"
							id="confirmPassword"
							bind:value={confirmPassword}
							required
							placeholder="••••••••"
						/>
					</div>
					
					<Button type="submit" disabled={isLoading} class="w-full">
						{isLoading ? 'Creating account...' : 'Create Account'}
					</Button>
				</form>
				
				<p class="mt-6 text-center text-sm text-muted-foreground">
					Already have an account? 
					<a href="/login" class="text-primary hover:underline font-medium">Sign in</a>
				</p>
			</CardContent>
		</Card>
	</div>
{/if}