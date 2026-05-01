<script lang="ts">
	import * as DropdownMenu from '$lib/components/ui/dropdown-menu';
	import LanguagesIcon from 'lucide-svelte/icons/languages';
	import CheckIcon from 'lucide-svelte/icons/check';
	import { m } from '$lib/paraglide/messages';
	import { locales, type Locale } from '$lib/paraglide/runtime';
	import { getCurrentLocale, localeLabels, switchLocale } from '$lib/i18n';

	interface Props {
		compact?: boolean;
	}

	let { compact = false }: Props = $props();

	let currentLocale = $derived(getCurrentLocale());

	function selectLocale(locale: Locale) {
		switchLocale(locale);
	}
</script>

<DropdownMenu.Root>
	<DropdownMenu.Trigger>
		{#snippet child({ props })}
			<button
				{...props}
				type="button"
				class="inline-flex items-center gap-2 rounded-full border border-border bg-background px-3 py-1.5 text-sm text-foreground transition-colors hover:border-foreground/30"
				aria-label={m.language_label()}
			>
				<LanguagesIcon class="h-4 w-4" />
				{#if !compact}
					<span>{localeLabels[currentLocale]}</span>
				{/if}
			</button>
		{/snippet}
	</DropdownMenu.Trigger>
	<DropdownMenu.Content class="w-44" align="end">
		<DropdownMenu.Label>{m.language_label()}</DropdownMenu.Label>
		<DropdownMenu.Separator />
		{#each locales as locale (locale)}
			<DropdownMenu.Item onclick={() => selectLocale(locale)}>
				<div class="flex w-full items-center justify-between gap-3">
					<span>{localeLabels[locale]}</span>
					{#if locale === currentLocale}
						<CheckIcon class="h-4 w-4 text-primary" />
					{/if}
				</div>
			</DropdownMenu.Item>
		{/each}
	</DropdownMenu.Content>
</DropdownMenu.Root>
