<script lang="ts">
	import { env } from '$env/dynamic/public';

	let {
		image,
		title,
		domain,
		timeAgo,
		is_sk,
		is_svelte
	}: {
		image: string | null;
		title: string;
		domain: string;
		timeAgo: string;
		is_sk: boolean;
		is_svelte: boolean | null;
	} = $props();

	const src = `${env.PUBLIC_STATIC_HOST}/${encodeURIComponent(image ?? '')}`;
	let svelteDeclaration = $derived.by(() => {
		if (!is_sk && is_svelte) {
			return 'Svelte';
		}

		return 'SvelteKit';
	});
</script>

<a
	href={domain}
	target="_blank"
	class="flex flex-col border rounded-sm focus:border-primary hover:border-primary transition-colors duration-100"
>
	<div class="aspect-1280/800 border-b relative">
		{#if image}
			<div class="absolute inset-0">
				<img
					loading="lazy"
					{src}
					alt={title}
					class="object-cover w-full h-full"
					width="580"
					height="360"
				/>
			</div>
		{:else}
			<div class="flex items-center justify-center w-full h-full text-sm text-muted-foreground">
				No Image Found
			</div>
		{/if}
	</div>
	<div class="p-2">
		<h2 class="font-medium text-ellipsis line-clamp-1 text-primary">{domain}</h2>
		<p class="text-ellipsis line-clamp-1">{title}</p>
		<div class="flex items-center gap-4 text-xs">
			<p class=" text-muted-foreground">{timeAgo}</p>
			<span class="text-primary p-1 bg-primary/10">{svelteDeclaration}</span>
		</div>
	</div>
</a>
