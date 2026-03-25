<script lang="ts">
	import { onMount } from 'svelte';
	import { auth } from '$lib/stores/auth';
	import { client, type Workspace } from '$lib/api/client';
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

	let workspaces = $state<Workspace[] | null>(null);
	let loading = $state(true);
	let error = $state('');
	let showCreateWorkspace = $state(false);
	let newWorkspaceName = $state('');

	let authReady = $state(false);
	let isAuthenticated = $state(false);

	onMount(() => {
		const unsubscribe = auth.subscribe((state) => {
			isAuthenticated = state.isAuthenticated;

			if (!state.isLoading && !authReady) {
				authReady = true;
				if (state.isAuthenticated) {
					loadWorkspaces();
				} else {
					loading = false;
				}
			}
		});

		return unsubscribe;
	});

	async function loadWorkspaces() {
		try {
			const { data, error: err } = await client.GET('/workspaces');
			if (err || !data) throw new Error('Failed to load workspaces');
			workspaces = data;
		} catch (e) {
			console.error('Failed to load workspaces:', e);
			error = (e as Error).message;
		} finally {
			loading = false;
		}
	}

	async function createWorkspace(e: Event) {
		e.preventDefault();
		if (!newWorkspaceName.trim()) return;

		try {
			const { error: err } = await client.POST('/workspaces', {
				body: { name: newWorkspaceName }
			});
			if (err) throw new Error(err.detail || 'Failed to create workspace');
			newWorkspaceName = '';
			showCreateWorkspace = false;
			await loadWorkspaces();
		} catch (e) {
			console.error('Failed to create workspace:', e);
			error = (e as Error).message;
		}
	}
</script>

<svelte:head>
	<title>Dashboard - OpenPost</title>
</svelte:head>

{#if loading}
	<div class="flex justify-center py-12">
		<div class="h-8 w-8 animate-spin rounded-full border-b-2 border-primary"></div>
	</div>
{:else if error}
	<div class="mx-auto w-full max-w-[1360px] px-4 py-6 lg:px-8">
		<div class="rounded-md border border-destructive/20 bg-destructive/10 p-4 text-destructive">
			<p>Error: {error}</p>
			<Button
				variant="ghost"
				size="sm"
				onclick={() => {
					error = '';
					loading = true;
					loadWorkspaces();
				}}
				class="mt-2"
			>
				Retry
			</Button>
		</div>
	</div>
{:else if !isAuthenticated}
	<div class="mx-auto w-full max-w-[1360px] px-4 py-6 text-center lg:px-8">
		<p class="text-muted-foreground">Please log in to view your workspaces.</p>
		<a href="/login" class="mt-2 inline-block font-medium text-primary hover:underline">
			Go to Login
		</a>
	</div>
{:else}
	<div class="mx-auto w-full max-w-[1360px] px-4 py-6 lg:px-8">
		<div class="mb-8 flex items-center justify-between">
			<h1 class="text-2xl font-bold">Workspaces</h1>
			<Button onclick={() => (showCreateWorkspace = true)}>New Workspace</Button>
		</div>

		{#if !workspaces || workspaces.length === 0}
			<div class="py-12 text-center">
				<Card class="mx-auto max-w-md">
					<CardHeader>
						<CardTitle>No workspaces yet</CardTitle>
						<CardDescription>Create your first workspace to start scheduling posts.</CardDescription
						>
					</CardHeader>
					<CardContent>
						<Button onclick={() => (showCreateWorkspace = true)}>Create Workspace</Button>
					</CardContent>
				</Card>
			</div>
		{:else}
			<div class="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-3">
				{#each workspaces as workspace}
					<a href="/workspace/{workspace.id}" class="block">
						<Card class="h-full transition-all hover:border-primary/50 hover:shadow-md">
							<CardHeader>
								<CardTitle>{workspace.name}</CardTitle>
								<CardDescription
									>Created {new Date(workspace.created_at).toLocaleDateString()}</CardDescription
								>
							</CardHeader>
						</Card>
					</a>
				{/each}
			</div>
		{/if}
	</div>
{/if}

{#if showCreateWorkspace}
	<!-- svelte-ignore a11y_click_events_have_key_events a11y_no_static_element_interactions -->
	<div
		class="fixed inset-0 z-50 flex items-center justify-center bg-black/50"
		onclick={() => (showCreateWorkspace = false)}
	>
		<Card class="mx-4 w-full max-w-md" onclick={(e: MouseEvent) => e.stopPropagation()}>
			<CardHeader>
				<CardTitle>Create Workspace</CardTitle>
			</CardHeader>
			<CardContent>
				<form onsubmit={createWorkspace} class="space-y-4">
					<div class="space-y-2">
						<Label for="workspace-name">Workspace Name</Label>
						<Input
							type="text"
							id="workspace-name"
							bind:value={newWorkspaceName}
							placeholder="My Workspace"
							required
						/>
					</div>
					<div class="flex justify-end gap-3">
						<Button type="button" variant="outline" onclick={() => (showCreateWorkspace = false)}
							>Cancel</Button
						>
						<Button type="submit">Create</Button>
					</div>
				</form>
			</CardContent>
		</Card>
	</div>
{/if}
