<script lang="ts">
	import { DropdownMenu as DropdownMenuPrimitive } from "bits-ui";
	import RiSubtractLine from 'remixicon-svelte/icons/subtract-line';
	import RiCheckLine from 'remixicon-svelte/icons/check-line';
	import { cn, type WithoutChildrenOrChild } from "$lib/utils.js";
	import type { Snippet } from "svelte";

	let {
		ref = $bindable(null),
		checked = $bindable(false),
		indeterminate = $bindable(false),
		class: className,
		children: childrenProp,
		...restProps
	}: WithoutChildrenOrChild<DropdownMenuPrimitive.CheckboxItemProps> & {
		children?: Snippet;
	} = $props();
</script>

<DropdownMenuPrimitive.CheckboxItem
	bind:ref
	bind:checked
	bind:indeterminate
	data-slot="dropdown-menu-checkbox-item"
	class={cn(
		"focus:bg-accent focus:text-accent-foreground focus:**:text-accent-foreground min-h-7 gap-2 rounded-md py-1.5 pr-8 pl-2 text-xs data-inset:pl-7.5 [&_svg:not([class*='size-'])]:size-3.5 relative flex cursor-default items-center outline-hidden select-none data-[disabled]:pointer-events-none data-[disabled]:opacity-50 [&_svg]:pointer-events-none [&_svg]:shrink-0",
		className
	)}
	{...restProps}
>
	{#snippet children({ checked, indeterminate })}
		<span
			class="absolute right-2 flex items-center justify-center pointer-events-none"
			data-slot="dropdown-menu-checkbox-item-indicator"
		>
			{#if indeterminate}
				<RiSubtractLine  />
			{:else if checked}
				<RiCheckLine  />
			{/if}
		</span>
		{@render childrenProp?.()}
	{/snippet}
</DropdownMenuPrimitive.CheckboxItem>
