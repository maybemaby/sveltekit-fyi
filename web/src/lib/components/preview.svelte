<script lang="ts">
	import { env } from '$env/dynamic/public';

	let {
		image,
		title,
		domain,
		timeAgo
	}: {
		image: string | null;
		title: string;
		domain: string;
		timeAgo: string;
	} = $props();

	const src = `${env.PUBLIC_STATIC_HOST}/${encodeURIComponent(image ?? '')}`;
</script>

<a
	href={domain}
	target="_blank"
	class="flex flex-col border rounded-sm focus:border-primary hover:border-primary transition-colors duration-100"
>
	<div class="aspect-1280/800 border-b relative">
		{#if image}
			<div class="absolute inset-0">
				<img {src} alt={title} class="object-cover w-full h-full" />
			</div>
		{:else}
			<div class="flex items-center justify-center w-full h-full text-sm text-muted-foreground">
				No Image Found
			</div>
		{/if}
	</div>
	<div class="p-2">
		<h2 class="font-medium text-ellipsis line-clamp-1">{domain}</h2>
		<p class="text-ellipsis line-clamp-1">{title}</p>
		<p class="text-xs text-muted-foreground">{timeAgo}</p>
	</div>
</a>
