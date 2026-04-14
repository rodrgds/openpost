<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { auth } from '$lib/stores/auth';
	import { client } from '$lib/api/client';
	import { Button } from '$lib/components/ui/button';
	import { Card, CardContent } from '$lib/components/ui/card';
	import { Input } from '$lib/components/ui/input';
	import { Label } from '$lib/components/ui/label';
	import Logo from '$lib/components/Logo.svelte';
	import RocketIcon from 'lucide-svelte/icons/rocket';
	import LoaderIcon from 'lucide-svelte/icons/loader-2';

	let workspaceName = $state('Personal');
	let isLoading = $state(false);
	let error = $state('');
	let authReady = $state(false);
	let pageLoading = $state(true);

	onMount(() => {
		const unsubscribe = auth.subscribe((state) => {
			if (!state.isLoading && !authReady) {
				authReady = true;
				if (!state.isAuthenticated) {
					goto('/login');
					return;
				}
				void loadOnboardingState();
			}
		});
		return unsubscribe;
	});

	async function loadOnboardingState() {
		pageLoading = true;
		error = '';
		try {
			const { data, error: workspacesError } = await client.GET('/workspaces');
			if (workspacesError) {
				throw new Error(
					(workspacesError as { detail?: string })?.detail || 'Failed to load workspaces'
				);
			}

			if ((data ?? []).length > 0) {
				goto('/');
				return;
			}
		} catch (e) {
			error = (e as Error).message;
		} finally {
			pageLoading = false;
		}
	}

	async function handleCreate(e: Event) {
		e.preventDefault();
		if (!workspaceName.trim()) return;

		isLoading = true;
		error = '';

		try {
			const { error: err } = await client.POST('/workspaces', {
				body: { name: workspaceName.trim() }
			});
			if (err) throw new Error((err as any).detail || 'Failed to create workspace');
			goto('/');
		} catch (e) {
			error = (e as Error).message;
		} finally {
			isLoading = false;
		}
	}
</script>

<svelte:head>
	<title>Welcome to OpenPost</title>
</svelte:head>

{#if pageLoading}
	<div class="flex min-h-[80vh] flex-col items-center justify-center gap-4 px-4">
		<LoaderIcon class="h-6 w-6 animate-spin text-primary" />
		<p class="text-sm text-muted-foreground">Loading workspace setup...</p>
	</div>
{:else}
	<div class="flex min-h-[80vh] flex-col items-center justify-center gap-6 px-4">
		<div class="flex justify-center">
			<Logo width={80} height={23} />
		</div>

		<div class="w-full max-w-md text-center">
			<div class="mb-6 flex justify-center">
				<div class="flex h-16 w-16 items-center justify-center rounded-2xl bg-primary/10">
					<RocketIcon class="h-8 w-8 text-primary" />
				</div>
			</div>
			<h1 class="mb-2 text-2xl font-bold tracking-tight">Welcome to OpenPost</h1>
			<p class="mb-8 text-muted-foreground">
				Let's set up your first workspace. This is where you'll organize your posts and connect your
				social accounts.
			</p>

			<Card>
				<CardContent class="pt-6">
					{#if error}
						<div
							class="mb-4 rounded-md border border-destructive/20 bg-destructive/10 p-3 text-sm text-destructive"
						>
							{error}
						</div>
					{/if}

					<form onsubmit={handleCreate} class="space-y-4">
						<div class="space-y-2">
							<Label for="workspace-name">Workspace name</Label>
							<Input
								type="text"
								id="workspace-name"
								bind:value={workspaceName}
								placeholder="e.g. Personal, My Brand"
								required
								autofocus
							/>
							<p class="text-xs text-muted-foreground">
								You can create more workspaces later for different projects or brands.
							</p>
						</div>

						<Button type="submit" disabled={isLoading || !workspaceName.trim()} class="w-full">
							{#if isLoading}
								<LoaderIcon class="mr-2 h-4 w-4 animate-spin" />
								Creating...
							{:else}
								Get Started
							{/if}
						</Button>
					</form>
				</CardContent>
			</Card>
		</div>
	</div>
{/if}
