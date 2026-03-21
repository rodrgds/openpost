<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/stores';
	import { api } from '$lib/api/client';
	import type { Workspace } from '$lib/types';
	import * as Sidebar from '$lib/components/ui/sidebar';
	import * as DropdownMenu from '$lib/components/ui/dropdown-menu';
	import * as Avatar from '$lib/components/ui/avatar';
	import HouseIcon from 'lucide-svelte/icons/home';
	import UsersIcon from 'lucide-svelte/icons/users';
	import FileTextIcon from 'lucide-svelte/icons/file-text';
	import FolderOpenIcon from 'lucide-svelte/icons/folder-open';
	import PlusIcon from 'lucide-svelte/icons/plus';
	import ChevronDownIcon from 'lucide-svelte/icons/chevron-down';
	import BadgeCheckIcon from 'lucide-svelte/icons/badge-check';
	import CreditCardIcon from 'lucide-svelte/icons/credit-card';
	import BellIcon from 'lucide-svelte/icons/bell';
	import LogOutIcon from 'lucide-svelte/icons/log-out';
	import ChevronsUpDownIcon from 'lucide-svelte/icons/chevrons-up-down';
	import { auth } from '$lib/stores/auth';
	import { goto } from '$app/navigation';
	import { toggleMode } from 'mode-watcher';
	import SunIcon from 'lucide-svelte/icons/sun';
	import MoonIcon from 'lucide-svelte/icons/moon';

	let authState = $derived($auth);
	const sidebar = Sidebar.useSidebar();
	let pathname = $derived($page.url.pathname);

	let workspaces = $state<Workspace[]>([]);
	let workspacesLoading = $state(true);
	let selectedWorkspaceId = $derived($page.params.id || workspaces[0]?.id || '');
	let selectedWorkspaceName = $derived(
		workspaces.find((workspace) => workspace.id === selectedWorkspaceId)?.name || 'Select workspace'
	);

	const navItems = [
		{ title: 'Dashboard', url: '/', icon: HouseIcon, isActive: () => pathname === '/' },
		{
			title: 'Workspaces',
			url: '/',
			icon: FileTextIcon,
			isActive: () => pathname.startsWith('/workspace/')
		},
		{ title: 'Accounts', url: '/accounts', icon: UsersIcon, isActive: () => pathname.startsWith('/accounts') }
	];

	onMount(async () => {
		try {
			workspaces = await api.listWorkspaces();
		} catch {
			workspaces = [];
		} finally {
			workspacesLoading = false;
		}
	});

	function handleLogout() {
		auth.logout();
		goto('/login');
	}

	function openWorkspace(workspaceId: string) {
		goto(`/workspace/${workspaceId}`);
	}
</script>

<Sidebar.Root>
	<Sidebar.Header>
		<!-- Workspace Switcher -->
		<Sidebar.Menu>
			<Sidebar.MenuItem>
				<DropdownMenu.Root>
					<DropdownMenu.Trigger>
						{#snippet child({ props })}
							<Sidebar.MenuButton {...props} class="w-full px-1.5">
								<div class="bg-sidebar-primary text-sidebar-primary-foreground flex aspect-square size-5 items-center justify-center rounded-md">
									<FolderOpenIcon class="size-3" />
								</div>
								<span class="truncate font-medium text-sidebar-foreground">{selectedWorkspaceName}</span>
								<ChevronDownIcon class="opacity-50 size-4 text-sidebar-foreground" />
							</Sidebar.MenuButton>
						{/snippet}
					</DropdownMenu.Trigger>
					<DropdownMenu.Content class="w-64 rounded-lg" align="start" side="bottom" sideOffset={4}>
						<DropdownMenu.Label class="text-xs text-muted-foreground">Workspaces</DropdownMenu.Label>
						{#if workspacesLoading}
							<DropdownMenu.Item disabled class="gap-2 p-2">Loading workspaces...</DropdownMenu.Item>
						{:else if workspaces.length === 0}
							<DropdownMenu.Item disabled class="gap-2 p-2">No workspaces found</DropdownMenu.Item>
						{:else}
							{#each workspaces as workspace (workspace.id)}
								<DropdownMenu.Item
									onSelect={() => openWorkspace(workspace.id)}
									class="gap-2 p-2"
								>
									<div class="flex size-6 items-center justify-center rounded-xs border">
										<span class="text-[0.625rem] font-semibold">{workspace.name.slice(0, 1).toUpperCase()}</span>
									</div>
									{workspace.name}
								</DropdownMenu.Item>
							{/each}
						{/if}
						<DropdownMenu.Separator />
						<DropdownMenu.Item class="gap-2 p-2" onSelect={() => goto('/')}>
							<div class="bg-background flex size-6 items-center justify-center rounded-md border">
								<PlusIcon class="size-4" />
							</div>
							<div class="text-muted-foreground font-medium">Add workspace</div>
						</DropdownMenu.Item>
					</DropdownMenu.Content>
				</DropdownMenu.Root>
			</Sidebar.MenuItem>
		</Sidebar.Menu>

		<!-- Main Navigation -->
		<Sidebar.Menu>
			{#each navItems as item (item.title)}
				<Sidebar.MenuItem>
					<Sidebar.MenuButton isActive={item.isActive()}>
						{#snippet child({ props })}
							<a href={item.url} {...props}>
								<item.icon class="text-sidebar-foreground" />
								<span class="text-sidebar-foreground">{item.title}</span>
							</a>
						{/snippet}
					</Sidebar.MenuButton>
				</Sidebar.MenuItem>
			{/each}
		</Sidebar.Menu>
	</Sidebar.Header>

	<Sidebar.Content class="mt-auto">
		<Sidebar.Separator />
		<!-- User Menu -->
		<Sidebar.Menu>
			<Sidebar.MenuItem>
				<DropdownMenu.Root>
					<DropdownMenu.Trigger>
						{#snippet child({ props })}
							<Sidebar.MenuButton {...props} size="lg" class="data-[state=open]:bg-sidebar-accent data-[state=open]:text-sidebar-accent-foreground">
								<Avatar.Root class="size-8 rounded-lg">
									<Avatar.Fallback class="rounded-lg bg-sidebar-primary text-sidebar-primary-foreground">
										{authState.user?.email?.charAt(0).toUpperCase() || 'U'}
									</Avatar.Fallback>
								</Avatar.Root>
								<div class="grid flex-1 text-start text-sm leading-tight">
									<span class="truncate font-medium text-sidebar-foreground">{authState.user?.email?.split('@')[0] || 'User'}</span>
									<span class="truncate text-xs text-sidebar-foreground/70">{authState.user?.email}</span>
								</div>
								<ChevronsUpDownIcon class="ms-auto size-4 text-sidebar-foreground" />
							</Sidebar.MenuButton>
						{/snippet}
					</DropdownMenu.Trigger>
					<DropdownMenu.Content class="w-56 rounded-lg" side={sidebar.isMobile ? "bottom" : "right"} align="start" sideOffset={4}>
						<DropdownMenu.Label class="p-0 font-normal">
							<div class="flex items-center gap-2 px-1 py-1.5 text-start text-sm">
								<Avatar.Root class="size-8 rounded-lg">
									<Avatar.Fallback class="rounded-lg bg-primary text-primary-foreground">
										{authState.user?.email?.charAt(0).toUpperCase() || 'U'}
									</Avatar.Fallback>
								</Avatar.Root>
								<div class="grid flex-1 text-start text-sm leading-tight">
									<span class="truncate font-medium">{authState.user?.email?.split('@')[0] || 'User'}</span>
									<span class="truncate text-xs text-muted-foreground">{authState.user?.email}</span>
								</div>
							</div>
						</DropdownMenu.Label>
						<DropdownMenu.Separator />
						<DropdownMenu.Group>
							<DropdownMenu.Item onclick={toggleMode}>
								<SunIcon class="size-4 rotate-0 scale-100 transition-all dark:-rotate-90 dark:scale-0 mr-2" />
								<MoonIcon class="absolute size-4 rotate-90 scale-0 transition-all dark:rotate-0 dark:scale-100 mr-2" />
								<span>Toggle theme</span>
							</DropdownMenu.Item>
						</DropdownMenu.Group>
						<DropdownMenu.Separator />
						<DropdownMenu.Group>
							<DropdownMenu.Item>
								<BadgeCheckIcon class="text-muted-foreground mr-2" />
								<span>Account</span>
							</DropdownMenu.Item>
							<DropdownMenu.Item>
								<CreditCardIcon class="text-muted-foreground mr-2" />
								<span>Billing</span>
							</DropdownMenu.Item>
							<DropdownMenu.Item>
								<BellIcon class="text-muted-foreground mr-2" />
								<span>Notifications</span>
							</DropdownMenu.Item>
						</DropdownMenu.Group>
						<DropdownMenu.Separator />
						<DropdownMenu.Item onclick={handleLogout}>
							<LogOutIcon class="text-muted-foreground mr-2" />
							<span>Log out</span>
						</DropdownMenu.Item>
					</DropdownMenu.Content>
				</DropdownMenu.Root>
			</Sidebar.MenuItem>
		</Sidebar.Menu>
	</Sidebar.Content>
	<Sidebar.Rail />
</Sidebar.Root>
