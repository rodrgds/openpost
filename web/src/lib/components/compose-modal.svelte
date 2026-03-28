<script lang="ts">
	import * as Dialog from '$lib/components/ui/dialog';
	import * as Drawer from '$lib/components/ui/drawer';
	import { IsMobile } from '$lib/hooks/is-mobile.svelte';
	import ComposePost from './compose-post.svelte';
	import { ui } from '$lib/stores/ui.svelte';

	interface Props {
		onSuccess?: () => void;
	}

	let { onSuccess }: Props = $props();
	const isMobile = new IsMobile();

	function handleSuccess() {
		ui.closeCompose();
		ui.triggerRefresh();
		if (onSuccess) onSuccess();
	}
</script>

{#if !isMobile.current}
	<Dialog.Root bind:open={ui.isComposeOpen}>
		<Dialog.Content class="p-6 sm:max-w-[1020px]">
			<Dialog.Header>
				<Dialog.Title class="text-2xl font-bold">Compose Post</Dialog.Title>
				<Dialog.Description>Schedule your post across multiple platforms.</Dialog.Description>
			</Dialog.Header>
			<div class="mt-4">
				<ComposePost
					initialDate={ui.composeInitialDate}
					onSuccess={handleSuccess}
					onCancel={() => ui.closeCompose()}
				/>
			</div>
		</Dialog.Content>
	</Dialog.Root>
{:else}
	<Drawer.Root bind:open={ui.isComposeOpen}>
		<Drawer.Content class="max-h-[95vh]">
			<div class="scrollbar-hide mx-auto w-full max-w-4xl overflow-auto p-6">
				<Drawer.Header class="px-0">
					<Drawer.Title class="text-2xl font-bold">Compose Post</Drawer.Title>
					<Drawer.Description>Schedule your post across multiple platforms.</Drawer.Description>
				</Drawer.Header>
				<div class="mt-4">
					<ComposePost
						initialDate={ui.composeInitialDate}
						onSuccess={handleSuccess}
						onCancel={() => ui.closeCompose()}
					/>
				</div>
			</div>
		</Drawer.Content>
	</Drawer.Root>
{/if}
