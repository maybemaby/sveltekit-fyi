<script lang="ts">
	let { caption, count }: { caption: string; count: number } = $props();

	let displayCount = $derived(
		Intl.NumberFormat('en-US', {
			notation: 'standard'
		}).format(count)
	);
</script>

<div class="h-24 border rounded-md p-4 flex flex-col justify-center">
	<div class="text-2xl text-primary font-semibold">
		{displayCount}
	</div>
	<div>
		{caption}
	</div>
</div>
