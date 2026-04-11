<script lang="ts">
	import * as Dialog from '$lib/components/ui/dialog';
	import ComposePost from './compose-post.svelte';
	import { ui } from '$lib/stores/ui.svelte';

	interface Props {
		onSuccess?: () => void;
	}

	let { onSuccess }: Props = $props();

	function handleSuccess() {
		ui.closeCompose();
		ui.triggerRefresh();
		if (onSuccess) onSuccess();
	}
</script>

<Dialog.Root bind:open={ui.isComposeOpen}>
	<Dialog.Content
		class="max-h-[90dvh] min-h-0 w-[calc(100%-1rem)] touch-pan-y overflow-y-auto overscroll-contain p-0 sm:w-full sm:max-w-[1020px]"
	>
		<div class="min-h-0 p-4 sm:p-6">
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
		</div>
	</Dialog.Content>
</Dialog.Root>
