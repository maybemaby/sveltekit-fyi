<script lang="ts">

	const { caption, count }: { caption: string; count: number } = $props();
</script>

<div class="h-24 border rounded-md p-4 flex flex-col justify-center">
	<div class="text-2xl text-primary font-semibold">
		{count}
	</div>
	<div>
		{caption}
	</div>
</div>
