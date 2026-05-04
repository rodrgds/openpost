<script lang="ts">
	import { getAuthenticatedMediaByID } from '$lib/media-url';
	import { getPlatformKey, getPlatformName } from '$lib/utils';
	import PlatformIcon from './platform-icon.svelte';
	import HeartIcon from 'lucide-svelte/icons/heart';
	import MessageCircleIcon from 'lucide-svelte/icons/message-circle';
	import RepeatIcon from 'lucide-svelte/icons/repeat-2';
	import ShareIcon from 'lucide-svelte/icons/share';
	import MoreHorizontalIcon from 'lucide-svelte/icons/more-horizontal';

	interface Props {
		platform: string;
		content: string;
		mediaIds: string[];
		username?: string;
		displayName?: string;
		avatarUrl?: string;
		variantContent?: string | null;
		isUnsynced?: boolean;
	}

	let {
		platform,
		content,
		mediaIds,
		username = 'username',
		displayName = 'Display Name',
		avatarUrl,
		variantContent = null,
		isUnsynced = false
	}: Props = $props();

	const platformKey = $derived(getPlatformKey(platform));
	const previewContent = $derived(variantContent ?? content);
	const previewName = $derived(getPlatformName(platform));

	function formatContent(text: string): string {
		return text
			.replace(/&/g, '&amp;')
			.replace(/</g, '&lt;')
			.replace(/>/g, '&gt;')
			.replace(/\n/g, '<br>')
			.replace(/(#[a-zA-Z0-9_]+)/g, '<span class="text-blue-500">$1</span>')
			.replace(/(@[a-zA-Z0-9_]+)/g, '<span class="text-blue-500">$1</span>')
			.replace(/(https?:\/\/[^\s]+)/g, '<span class="text-blue-500">$1</span>');
	}

	const mediaLayout = $derived.by(() => {
		if (mediaIds.length === 0) return 'none';
		if (mediaIds.length === 1) return 'single';
		if (mediaIds.length === 2) return 'grid-2';
		if (mediaIds.length === 3) return 'grid-3';
		return 'grid-4';
	});
</script>

{#if platformKey === 'x'}
	<!-- X/Twitter Preview -->
	<div class="w-full max-w-xl rounded-xl border border-border/60 bg-background p-4 shadow-sm">
		<div class="flex gap-3">
			<div class="shrink-0">
				{#if avatarUrl}
					<img src={avatarUrl} alt="" class="h-10 w-10 rounded-full object-cover" />
				{:else}
					<div class="flex h-10 w-10 items-center justify-center rounded-full bg-muted">
						<span class="text-sm font-medium">{displayName.charAt(0)}</span>
					</div>
				{/if}
			</div>
			<div class="min-w-0 flex-1">
				<div class="flex items-center gap-1">
					<span class="truncate font-semibold text-foreground">{displayName}</span>
					<span class="shrink-0 text-muted-foreground">@{username}</span>
					<span class="shrink-0 text-muted-foreground">·</span>
					<span class="shrink-0 text-muted-foreground">now</span>
				</div>
				{#if isUnsynced}
					<div class="mt-0.5 text-xs text-amber-500">Customized for {previewName}</div>
				{/if}
				<div class="mt-1 text-[15px] leading-normal text-foreground">
					<!-- eslint-disable-next-line svelte/no-at-html-tags -->
					{@html formatContent(previewContent)}
				</div>
				{#if mediaIds.length > 0}
					<div class="mt-3 overflow-hidden rounded-xl border border-border/60">
						{#if mediaLayout === 'single'}
							<img
								src={getAuthenticatedMediaByID(mediaIds[0])}
								alt=""
								class="h-auto w-full object-cover"
							/>
						{:else if mediaLayout === 'grid-2'}
							<div class="grid grid-cols-2 gap-0.5">
								{#each mediaIds as id (id)}
									<img
										src={getAuthenticatedMediaByID(id)}
										alt=""
										class="aspect-square w-full object-cover"
									/>
								{/each}
							</div>
						{:else if mediaLayout === 'grid-3'}
							<div class="grid grid-cols-2 gap-0.5">
								<img
									src={getAuthenticatedMediaByID(mediaIds[0])}
									alt=""
									class="col-span-2 aspect-video w-full object-cover"
								/>
								<img
									src={getAuthenticatedMediaByID(mediaIds[1])}
									alt=""
									class="aspect-square w-full object-cover"
								/>
								<img
									src={getAuthenticatedMediaByID(mediaIds[2])}
									alt=""
									class="aspect-square w-full object-cover"
								/>
							</div>
						{:else}
							<div class="grid grid-cols-2 gap-0.5">
								{#each mediaIds as id (id)}
									<img
										src={getAuthenticatedMediaByID(id)}
										alt=""
										class="aspect-square w-full object-cover"
									/>
								{/each}
							</div>
						{/if}
					</div>
				{/if}
				<div class="mt-3 flex items-center justify-between text-muted-foreground">
					<div class="flex items-center gap-1 hover:text-blue-500">
						<MessageCircleIcon class="h-4 w-4" />
					</div>
					<div class="flex items-center gap-1 hover:text-green-500">
						<RepeatIcon class="h-4 w-4" />
					</div>
					<div class="flex items-center gap-1 hover:text-red-500">
						<HeartIcon class="h-4 w-4" />
					</div>
					<div class="flex items-center gap-1 hover:text-blue-500">
						<ShareIcon class="h-4 w-4" />
					</div>
				</div>
			</div>
		</div>
	</div>
{:else if platformKey === 'mastodon'}
	<!-- Mastodon Preview -->
	<div class="w-full max-w-xl rounded-lg border border-border/60 bg-background p-4">
		<div class="flex gap-3">
			<div class="shrink-0">
				{#if avatarUrl}
					<img src={avatarUrl} alt="" class="h-12 w-12 rounded-full object-cover" />
				{:else}
					<div
						class="flex h-12 w-12 items-center justify-center rounded-full bg-indigo-100 dark:bg-indigo-900"
					>
						<span class="text-lg font-semibold text-indigo-600 dark:text-indigo-300"
							>{displayName.charAt(0)}</span
						>
					</div>
				{/if}
			</div>
			<div class="min-w-0 flex-1">
				<div class="flex items-center gap-1.5">
					<span class="font-semibold text-foreground">{displayName}</span>
					<span class="text-sm text-muted-foreground">@{username}</span>
				</div>
				{#if isUnsynced}
					<div class="text-xs text-amber-500">Customized for {previewName}</div>
				{/if}
				<div class="mt-2 text-[15px] leading-relaxed whitespace-pre-wrap text-foreground">
					<!-- eslint-disable-next-line svelte/no-at-html-tags -->
					{@html formatContent(previewContent)}
				</div>
				{#if mediaIds.length > 0}
					<div class="mt-3 grid grid-cols-2 gap-2">
						{#each mediaIds as id (id)}
							<img
								src={getAuthenticatedMediaByID(id)}
								alt=""
								class="rounded-lg object-cover"
								style="max-height: 260px;"
							/>
						{/each}
					</div>
				{/if}
				<div class="mt-3 flex items-center gap-5 text-sm text-muted-foreground">
					<span class="flex items-center gap-1.5 hover:text-indigo-500">
						<MessageCircleIcon class="h-4 w-4" />
						Reply
					</span>
					<span class="flex items-center gap-1.5 hover:text-green-500">
						<RepeatIcon class="h-4 w-4" />
						Boost
					</span>
					<span class="flex items-center gap-1.5 hover:text-red-500">
						<HeartIcon class="h-4 w-4" />
						Favorite
					</span>
					<span class="flex items-center gap-1.5 hover:text-foreground">
						<ShareIcon class="h-4 w-4" />
						Share
					</span>
				</div>
			</div>
		</div>
	</div>
{:else if platformKey === 'bluesky'}
	<!-- Bluesky Preview -->
	<div class="w-full max-w-xl border-b border-border/40 bg-background px-4 py-4">
		<div class="flex gap-3">
			<div class="shrink-0">
				{#if avatarUrl}
					<img src={avatarUrl} alt="" class="h-10 w-10 rounded-full object-cover" />
				{:else}
					<div
						class="flex h-10 w-10 items-center justify-center rounded-full bg-sky-100 dark:bg-sky-900"
					>
						<span class="text-sm font-bold text-sky-600 dark:text-sky-300"
							>{displayName.charAt(0)}</span
						>
					</div>
				{/if}
			</div>
			<div class="min-w-0 flex-1">
				<div class="flex items-center justify-between">
					<div class="flex items-center gap-1.5">
						<span class="font-semibold text-foreground">{displayName}</span>
						<span class="text-sm text-muted-foreground">@{username}</span>
						<span class="text-sm text-muted-foreground">· now</span>
					</div>
					<MoreHorizontalIcon class="h-4 w-4 text-muted-foreground" />
				</div>
				{#if isUnsynced}
					<div class="text-xs text-amber-500">Customized for {previewName}</div>
				{/if}
				<div class="mt-1 text-[15px] leading-normal text-foreground">
					<!-- eslint-disable-next-line svelte/no-at-html-tags -->
					{@html formatContent(previewContent)}
				</div>
				{#if mediaIds.length > 0}
					<div class="mt-3 overflow-hidden rounded-lg border border-border/40">
						{#if mediaLayout === 'single'}
							<img
								src={getAuthenticatedMediaByID(mediaIds[0])}
								alt=""
								class="h-auto w-full object-cover"
							/>
						{:else}
							<div class="grid grid-cols-2 gap-0.5">
								{#each mediaIds as id (id)}
									<img
										src={getAuthenticatedMediaByID(id)}
										alt=""
										class="aspect-square w-full object-cover"
									/>
								{/each}
							</div>
						{/if}
					</div>
				{/if}
				<div class="mt-3 flex items-center justify-between text-muted-foreground">
					<div class="flex items-center gap-1 hover:text-sky-500">
						<MessageCircleIcon class="h-4 w-4" />
						<span class="text-sm">0</span>
					</div>
					<div class="flex items-center gap-1 hover:text-green-500">
						<RepeatIcon class="h-4 w-4" />
						<span class="text-sm">0</span>
					</div>
					<div class="flex items-center gap-1 hover:text-red-500">
						<HeartIcon class="h-4 w-4" />
						<span class="text-sm">0</span>
					</div>
					<div class="flex items-center gap-1 hover:text-sky-500">
						<ShareIcon class="h-4 w-4" />
					</div>
				</div>
			</div>
		</div>
	</div>
{:else if platformKey === 'linkedin'}
	<!-- LinkedIn Preview -->
	<div class="w-full max-w-xl rounded-lg border border-border/60 bg-background p-4">
		<div class="flex items-center gap-3">
			{#if avatarUrl}
				<img src={avatarUrl} alt="" class="h-12 w-12 rounded-full object-cover" />
			{:else}
				<div
					class="flex h-12 w-12 items-center justify-center rounded-full bg-blue-100 dark:bg-blue-900"
				>
					<span class="text-lg font-semibold text-blue-700 dark:text-blue-300"
						>{displayName.charAt(0)}</span
					>
				</div>
			{/if}
			<div class="min-w-0 flex-1">
				<div class="font-semibold text-foreground">{displayName}</div>
				<div class="text-sm text-muted-foreground">@{username} · now</div>
			</div>
			<MoreHorizontalIcon class="h-5 w-5 text-muted-foreground" />
		</div>
		{#if isUnsynced}
			<div class="mt-1 text-xs text-amber-500">Customized for {previewName}</div>
		{/if}
		<div class="mt-3 text-sm leading-relaxed text-foreground">
			<!-- eslint-disable-next-line svelte/no-at-html-tags -->
			{@html formatContent(previewContent)}
		</div>
		{#if mediaIds.length > 0}
			<div class="mt-3 overflow-hidden rounded-lg border border-border/40">
				<img
					src={getAuthenticatedMediaByID(mediaIds[0])}
					alt=""
					class="h-auto w-full object-cover"
				/>
			</div>
			{#if mediaIds.length > 1}
				<div class="mt-2 text-xs text-muted-foreground">
					Preview shows the first attachment. LinkedIn publishing currently sends one media item.
				</div>
			{/if}
		{/if}
		<div class="mt-3 flex items-center gap-4 border-t pt-3 text-sm text-muted-foreground">
			<span class="flex items-center gap-1.5 hover:text-blue-600">
				<HeartIcon class="h-4 w-4" />
				Like
			</span>
			<span class="flex items-center gap-1.5 hover:text-blue-600">
				<MessageCircleIcon class="h-4 w-4" />
				Comment
			</span>
			<span class="flex items-center gap-1.5 hover:text-blue-600">
				<RepeatIcon class="h-4 w-4" />
				Repost
			</span>
			<span class="flex items-center gap-1.5 hover:text-blue-600">
				<ShareIcon class="h-4 w-4" />
				Send
			</span>
		</div>
	</div>
{:else if platformKey === 'threads'}
	<!-- Threads Preview -->
	<div class="w-full max-w-xl border-b border-border/40 bg-background px-4 py-4">
		<div class="flex gap-3">
			<div class="shrink-0">
				{#if avatarUrl}
					<img src={avatarUrl} alt="" class="h-9 w-9 rounded-full object-cover" />
				{:else}
					<div
						class="flex h-9 w-9 items-center justify-center rounded-full bg-orange-100 dark:bg-orange-900"
					>
						<span class="text-sm font-bold text-orange-600 dark:text-orange-300"
							>{displayName.charAt(0)}</span
						>
					</div>
				{/if}
			</div>
			<div class="min-w-0 flex-1">
				<div class="flex items-center justify-between">
					<div class="flex items-center gap-2">
						<span class="font-semibold text-foreground">{displayName}</span>
						<span class="text-sm text-muted-foreground">@{username}</span>
						<span class="text-sm text-muted-foreground">· now</span>
					</div>
					<MoreHorizontalIcon class="h-4 w-4 text-muted-foreground" />
				</div>
				{#if isUnsynced}
					<div class="text-xs text-amber-500">Customized for {previewName}</div>
				{/if}
				<div class="mt-1 text-[15px] leading-normal text-foreground">
					<!-- eslint-disable-next-line svelte/no-at-html-tags -->
					{@html formatContent(previewContent)}
				</div>
				{#if mediaIds.length > 0}
					<div class="mt-3 overflow-hidden rounded-lg border border-border/40">
						<img
							src={getAuthenticatedMediaByID(mediaIds[0])}
							alt=""
							class="h-auto w-full object-cover"
						/>
					</div>
					{#if mediaIds.length > 1}
						<div class="mt-2 text-xs text-muted-foreground">
							Preview shows the first attachment. Threads publishing currently sends one media item.
						</div>
					{/if}
				{/if}
				<div class="mt-3 flex items-center gap-6 text-muted-foreground">
					<div class="flex items-center gap-1.5 hover:text-foreground">
						<MessageCircleIcon class="h-4 w-4" />
						<span class="text-sm">0</span>
					</div>
					<div class="flex items-center gap-1.5 hover:text-foreground">
						<RepeatIcon class="h-4 w-4" />
						<span class="text-sm">0</span>
					</div>
					<div class="flex items-center gap-1.5 hover:text-red-500">
						<HeartIcon class="h-4 w-4" />
						<span class="text-sm">0</span>
					</div>
					<div class="flex items-center gap-1.5 hover:text-foreground">
						<ShareIcon class="h-4 w-4" />
					</div>
				</div>
			</div>
		</div>
	</div>
{:else}
	<!-- Generic Preview -->
	<div class="w-full max-w-xl rounded-lg border border-border/60 bg-background p-4">
		<div class="flex items-center gap-2">
			<PlatformIcon platform={platformKey} class="h-5 w-5" />
			<span class="font-medium">{previewName}</span>
		</div>
		{#if isUnsynced}
			<div class="mt-1 text-xs text-amber-500">Customized for {previewName}</div>
		{/if}
		<div class="mt-2 text-sm text-foreground">
			<!-- eslint-disable-next-line svelte/no-at-html-tags -->
			{@html formatContent(previewContent)}
		</div>
		{#if mediaIds.length > 0}
			<div class="mt-3 grid grid-cols-2 gap-2">
				{#each mediaIds as id (id)}
					<img src={getAuthenticatedMediaByID(id)} alt="" class="rounded-lg object-cover" />
				{/each}
			</div>
		{/if}
	</div>
{/if}
