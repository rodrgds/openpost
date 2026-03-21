<script lang="ts">
	import { onMount } from 'svelte';
	import { auth } from '$lib/stores/auth';
	import { api } from '$lib/api/client';
	import { Button } from '$lib/components/ui/button';
	import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '$lib/components/ui/card';
	import { Input } from '$lib/components/ui/input';
	import { Label } from '$lib/components/ui/label';
	import type { Workspace } from '$lib/types';
	
	let workspaces = $state<Workspace[] | null>(null);
	let loading = $state(true);
	let error = $state('');
	let showCreateWorkspace = $state(false);
	let newWorkspaceName = $state('');
	
	let authReady = $state(false);
	let isAuthenticated = $state(false);
	
	onMount(() => {
		const unsubscribe = auth.subscribe(state => {
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
			workspaces = await api.listWorkspaces();
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
			await api.createWorkspace(newWorkspaceName);
			newWorkspaceName = '';
			showCreateWorkspace = false;
			workspaces = await api.listWorkspaces();
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
		<div class="animate-spin rounded-full h-8 w-8 border-b-2 border-primary"></div>
	</div>
{:else if error}
	<div class="mx-auto w-full max-w-[1360px] px-4 py-6 lg:px-8">
		<div class="bg-destructive/10 border border-destructive/20 rounded-md p-4 text-destructive">
			<p>Error: {error}</p>
			<Button variant="ghost" size="sm" onclick={() => { error = ''; loading = true; loadWorkspaces(); }} class="mt-2">
				Retry
			</Button>
		</div>
	</div>
{:else if !isAuthenticated}
	<div class="mx-auto w-full max-w-[1360px] px-4 py-6 text-center lg:px-8">
		<p class="text-muted-foreground">Please log in to view your workspaces.</p>
		<a href="/login" class="text-primary hover:underline font-medium mt-2 inline-block">
			Go to Login
		</a>
	</div>
{:else}
	<div class="mx-auto w-full max-w-[1360px] px-4 py-6 lg:px-8">
		<div class="flex justify-between items-center mb-8">
			<h1 class="text-2xl font-bold">Workspaces</h1>
			<Button onclick={() => showCreateWorkspace = true}>New Workspace</Button>
		</div>
		
		{#if !workspaces || workspaces.length === 0}
			<div class="text-center py-12">
				<Card class="max-w-md mx-auto">
					<CardHeader>
						<CardTitle>No workspaces yet</CardTitle>
						<CardDescription>Create your first workspace to start scheduling posts.</CardDescription>
					</CardHeader>
					<CardContent>
						<Button onclick={() => showCreateWorkspace = true}>Create Workspace</Button>
					</CardContent>
				</Card>
			</div>
		{:else}
			<div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
				{#each workspaces as workspace}
					<a
						href="/workspace/{workspace.id}"
						class="block"
					>
						<Card class="hover:border-primary/50 hover:shadow-md transition-all h-full">
							<CardHeader>
								<CardTitle>{workspace.name}</CardTitle>
								<CardDescription>Created {new Date(workspace.created_at).toLocaleDateString()}</CardDescription>
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
	<div class="fixed inset-0 bg-black/50 flex items-center justify-center z-50" onclick={() => showCreateWorkspace = false}>
		<Card class="w-full max-w-md mx-4" onclick={(e: MouseEvent) => e.stopPropagation()}>
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
					<div class="flex gap-3 justify-end">
						<Button type="button" variant="outline" onclick={() => showCreateWorkspace = false}>Cancel</Button>
						<Button type="submit">Create</Button>
					</div>
				</form>
			</CardContent>
		</Card>
	</div>
{/if}
