<script lang="ts">
	import { onMount } from 'svelte';
	import { client } from '$lib/api/client';
	import { workspaceCtx } from '$lib/stores/workspace.svelte';
	import { ui } from '$lib/stores/ui.svelte';
	import { Button } from '$lib/components/ui/button';
	import * as Select from '$lib/components/ui/select';
	import PageContainer from '$lib/components/page-container.svelte';
	import LoaderIcon from 'lucide-svelte/icons/loader-2';
	import LightbulbIcon from 'lucide-svelte/icons/lightbulb';
	import PlusIcon from 'lucide-svelte/icons/plus';
	import TrashIcon from 'lucide-svelte/icons/trash';
	import ShuffleIcon from 'lucide-svelte/icons/shuffle';
	import { Skeleton } from '$lib/components/ui/skeleton/index.js';
	import { goto } from '$app/navigation';

	interface Prompt {
		id: string;
		workspace_id?: string;
		user_id?: string;
		text: string;
		category: string;
		is_built_in: boolean;
		created_at: string;
	}

	let prompts = $state<Prompt[]>([]);
	let categories = $state<string[]>([]);
	let loading = $state(false);
	let loadingCategories = $state(false);
	let selectedCategory = $state<string>('all');
	let showAddPrompt = $state(false);
	let newPromptText = $state('');
	let newPromptCategory = $state('');
	let submitting = $state(false);
	let toastMessage = $state('');

	interface PromptsQueryParams {
		workspace_id: string;
		category?: string;
	}

	async function loadPrompts() {
		if (!workspaceCtx.currentWorkspace) return;
		loading = true;
		try {
			const params: PromptsQueryParams = { workspace_id: workspaceCtx.currentWorkspace.id };
			if (selectedCategory !== 'all') {
				params.category = selectedCategory;
			}
			const { data, error: err } = await (client as any).GET('/prompts', {
				params: { query: params }
			});
			if (!err && data) {
				prompts = data;
			}
		} catch (e) {
			console.error('Failed to load prompts:', e);
		} finally {
			loading = false;
		}
	}

	async function loadCategories() {
		loadingCategories = true;
		try {
			const { data, error: err } = await (client as any).GET('/prompts/categories');
			if (!err && data) {
				categories = data.categories;
				if (categories.length > 0 && !newPromptCategory) {
					newPromptCategory = categories[0];
				}
			}
		} catch (e) {
			console.error('Failed to load categories:', e);
		} finally {
			loadingCategories = false;
		}
	}

	async function addPrompt() {
		if (!workspaceCtx.currentWorkspace || !newPromptText.trim() || !newPromptCategory) return;
		submitting = true;
		try {
			const { error: err } = await (client as any).POST('/prompts', {
				body: {
					workspace_id: workspaceCtx.currentWorkspace.id,
					text: newPromptText.trim(),
					category: newPromptCategory
				}
			});
			if (err) throw err;
			showAddPrompt = false;
			newPromptText = '';
			toastMessage = 'Prompt created successfully';
			await loadPrompts();
		} catch (e) {
			toastMessage = (e as Error).message || 'Failed to create prompt';
		} finally {
			submitting = false;
		}
	}

	async function deletePrompt(id: string) {
		try {
			const { error: err } = await (client as any).DELETE('/prompts/{id}', {
				params: { path: { id } }
			});
			if (err) throw err;
			toastMessage = 'Prompt deleted successfully';
			await loadPrompts();
		} catch (e) {
			toastMessage = (e as Error).message || 'Failed to delete prompt';
		}
	}

	async function getRandomPrompt() {
		if (!workspaceCtx.currentWorkspace) return;
		try {
			const params: PromptsQueryParams = { workspace_id: workspaceCtx.currentWorkspace.id };
			if (selectedCategory !== 'all') {
				params.category = selectedCategory;
			}
			const { data, error: err } = await (client as any).GET('/prompts/random', {
				params: { query: params }
			});
			if (!err && data) {
				ui.setPrompt(data.text);
				goto('/');
			}
		} catch (e) {
			console.error('Failed to get random prompt:', e);
		}
	}

	function usePrompt(text: string) {
		ui.setPrompt(text);
		goto('/');
	}

	$effect(() => {
		if (workspaceCtx.currentWorkspace) {
			loadPrompts();
			loadCategories();
		}
	});
</script>

<svelte:head>
	<title>Prompts - OpenPost</title>
</svelte:head>

{#if toastMessage}
	<div
		class="pointer-events-auto fixed right-4 bottom-4 z-50 mb-4 flex items-center gap-2 rounded-lg border bg-background px-4 py-3 shadow-lg"
	>
		<span class="text-sm">{toastMessage}</span>
		<button onclick={() => (toastMessage = '')}>
			<span class="sr-only">Close</span>
			<svg class="size-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
				<path d="M18 6L6 18M6 6l12 12" />
			</svg>
		</button>
	</div>
{/if}

<PageContainer
	title="Writing Prompts"
	description="Get inspired with writing prompts for your social media content"
	icon={LightbulbIcon}
	loading={!workspaceCtx.currentWorkspace}
	loadingMessage="Loading workspace..."
>
	<div class="space-y-6">
		<!-- Header Actions -->
		<div class="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
			<div class="flex items-center gap-2">
				{#if loadingCategories}
					<div class="h-9 w-32 animate-pulse rounded-md bg-muted"></div>
				{:else}
					<Select.Root
						type="single"
						value={selectedCategory}
						onValueChange={(v) => {
							selectedCategory = v;
							loadPrompts();
						}}
					>
						<Select.Trigger class="w-40">
							{selectedCategory === 'all' ? 'All Categories' : selectedCategory}
						</Select.Trigger>
						<Select.Content>
							<Select.Item value="all">All Categories</Select.Item>
							{#each categories as category}
								<Select.Item value={category}>{category}</Select.Item>
							{/each}
						</Select.Content>
					</Select.Root>
				{/if}
			</div>
			<div class="flex gap-2">
				<Button onclick={getRandomPrompt} variant="outline" class="gap-2">
					<ShuffleIcon class="h-4 w-4" />
					Random
				</Button>
				<Button onclick={() => (showAddPrompt = true)} class="gap-2">
					<PlusIcon class="h-4 w-4" />
					Add Prompt
				</Button>
			</div>
		</div>

		<!-- Prompts Grid -->
		{#if loading}
			<div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
				{#each Array(6) as _}
					<div class="h-32 rounded-lg bg-muted">
						<Skeleton class="h-full w-full" />
					</div>
				{/each}
			</div>
		{:else if prompts.length === 0}
			<div class="rounded-lg border border-dashed p-12 text-center">
				<LightbulbIcon class="mx-auto h-12 w-12 text-muted-foreground" />
				<h3 class="mt-4 text-base font-semibold">No prompts found</h3>
				<p class="mt-2 text-sm text-muted-foreground">
					Get started by adding your first writing prompt.
				</p>
			</div>
		{:else}
			{@const groupedPrompts = prompts.reduce(
				(acc, prompt) => {
					if (!acc[prompt.category]) acc[prompt.category] = [];
					acc[prompt.category].push(prompt);
					return acc;
				},
				{} as Record<string, Prompt[]>
			)}

			<div class="space-y-6">
				{#each Object.entries(groupedPrompts) as [category, categoryPrompts]}
					<section>
						<h2 class="mb-3 text-xs font-semibold tracking-wider text-muted-foreground uppercase">
							{category}
						</h2>
						<div class="grid gap-3 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
							{#each categoryPrompts as prompt}
								<button
									type="button"
									class="group relative flex flex-col items-start rounded-md border bg-card p-3 text-left transition-all hover:border-accent hover:bg-accent"
									onclick={() => usePrompt(prompt.text)}
								>
									<p
										class="line-clamp-4 text-sm leading-relaxed text-foreground/80 group-hover:text-foreground"
									>
										{prompt.text}
									</p>
									<div class="mt-2 flex w-full items-center justify-between">
										<span class="text-xs text-muted-foreground">
											{prompt.is_built_in ? 'Built-in' : 'Custom'}
										</span>
										{#if !prompt.is_built_in}
											<button
												type="button"
												class="text-xs text-muted-foreground hover:text-destructive"
												onclick={(e) => {
													e.stopPropagation();
													deletePrompt(prompt.id);
												}}
											>
												<TrashIcon class="h-3.5 w-3.5" />
											</button>
										{/if}
									</div>
								</button>
							{/each}
						</div>
					</section>
				{/each}
			</div>
		{/if}

		<!-- Add Prompt Modal -->
		{#if showAddPrompt}
			<div
				class="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4"
				onclick={(e) => {
					if (e.target === e.currentTarget) showAddPrompt = false;
				}}
			>
				<div class="w-full max-w-md rounded-lg border bg-background p-6 shadow-lg">
					<h3 class="mb-4 text-lg font-semibold">Add New Prompt</h3>
					<div class="space-y-4">
						<div class="space-y-2">
							<label class="text-sm font-medium" for="prompt-text">Prompt Text</label>
							<textarea
								id="prompt-text"
								bind:value={newPromptText}
								placeholder="What's your favorite tool for productivity and why?"
								rows={3}
								class="w-full rounded-md border border-input bg-transparent px-3 py-2 text-sm"
							></textarea>
						</div>
						<div class="space-y-2">
							<label class="text-sm font-medium" for="prompt-category">Category</label>
							{#if loadingCategories}
								<div class="h-9 animate-pulse rounded-md bg-muted"></div>
							{:else}
								<Select.Root
									type="single"
									value={newPromptCategory}
									onValueChange={(v) => (newPromptCategory = v)}
								>
									<Select.Trigger id="prompt-category" class="w-full">
										{newPromptCategory || 'Select a category'}
									</Select.Trigger>
									<Select.Content>
										{#each categories as category}
											<Select.Item value={category}>{category}</Select.Item>
										{/each}
									</Select.Content>
								</Select.Root>
							{/if}
						</div>
					</div>
					<div class="mt-6 flex justify-end gap-2">
						<Button onclick={() => (showAddPrompt = false)} variant="outline">Cancel</Button>
						<Button
							onclick={addPrompt}
							disabled={!newPromptText.trim() || !newPromptCategory || submitting}
						>
							{#if submitting}
								<LoaderIcon class="mr-2 h-4 w-4 animate-spin" />
							{/if}
							Add Prompt
						</Button>
					</div>
				</div>
			</div>
		{/if}
	</div>
</PageContainer>
