<script lang="ts">
	let {
		image,
		title,
		domain
	}: {
		image?: string;
		title: string;
		domain: string;
	} = $props();
</script>

<a
	href={domain}
	target="_blank"
	class="flex flex-col border rounded-sm focus:border-primary  hover:border-primary transition-colors duration-100"
>
	<div class="aspect-1280/800 border-b">
		{#if image}
			<img src={image} alt={title} class="object-cover w-full h-full" />
		{:else}
			<div class="flex items-center justify-center w-full h-full text-sm text-muted-foreground">
				No Image Found
			</div>
		{/if}
	</div>
	<div class="p-2">
		<h2 class="font-medium">{domain}</h2>
		<p class="text-ellipsis line-clamp-1">{title}</p>
	</div>
</a>
