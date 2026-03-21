<script lang="ts">
	import '../app.css';
	import './layout.css';
	import { ModeWatcher } from 'mode-watcher';
	import { onMount } from 'svelte';
	import { auth } from '$lib/stores/auth';
	import { goto } from '$app/navigation';
	import { page } from '$app/stores';
	import * as Sidebar from '$lib/components/ui/sidebar';
	import { Separator } from '$lib/components/ui/separator';
	import SidebarLeft from '$lib/components/sidebar-left.svelte';
	import SidebarRight from '$lib/components/sidebar-right.svelte';
	
	let { children } = $props();
	
	let authState = $derived($auth);
	let currentPath = $derived($page.url.pathname);
	const publicRoutes = ['/login', '/register', '/demo', '/demo/paraglide', '/accounts/mastodon/callback'];
	
	onMount(() => {
		auth.initialize();
	});
	
	$effect(() => {
		if (authState.isLoading) return;
		
		const isPublicRoute = publicRoutes.some(route => currentPath.startsWith(route));
		const isLandingPage = currentPath === '/';
		
		if (!authState.isAuthenticated && !isPublicRoute && !isLandingPage) {
			goto('/login');
		}
		
		if (authState.isAuthenticated && (currentPath === '/login' || currentPath === '/register')) {
			goto('/');
		}
	});
</script>

<svelte:head>
	<title>OpenPost</title>
</svelte:head>

<ModeWatcher />
{#if authState.isLoading}
	<div class="min-h-screen flex items-center justify-center">
		<div class="animate-spin rounded-full h-8 w-8 border-b-2 border-primary"></div>
	</div>
{:else if !authState.isAuthenticated}
	{#if currentPath === '/'}
		<div class="min-h-[80vh] flex items-center justify-center">
			<div class="max-w-md mx-auto px-4 py-12 text-center">
				<h1 class="text-3xl font-bold mb-4">OpenPost</h1>
				<p class="text-muted-foreground mb-6">Schedule posts across multiple social platforms.</p>
				<div class="flex gap-4 justify-center">
					<a href="/login" class="inline-flex items-center justify-center rounded-md bg-primary px-4 py-2 text-sm font-medium text-primary-foreground hover:bg-primary/90">Sign In</a>
					<a href="/register" class="inline-flex items-center justify-center rounded-md border border-input bg-background px-4 py-2 text-sm font-medium hover:bg-accent hover:text-accent-foreground">Create Account</a>
				</div>
			</div>
		</div>
	{:else}
		{@render children()}
	{/if}
{:else}
	<Sidebar.Provider>
		<SidebarLeft />
		<Sidebar.Inset>
			<header class="bg-background sticky top-0 flex h-14 shrink-0 items-center gap-2 border-b">
				<div class="flex flex-1 items-center gap-2 px-3">
					<Sidebar.Trigger class="text-sidebar-foreground" />
					<Separator orientation="vertical" class="me-2 h-4 bg-border" />
					<span class="text-sm font-medium text-foreground">OpenPost</span>
				</div>
			</header>
			<div class="flex flex-1 flex-col gap-4 p-4">
				{@render children()}
			</div>
		</Sidebar.Inset>
		<SidebarRight />
	</Sidebar.Provider>
{/if}