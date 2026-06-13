<script lang="ts">
	import './layout.css';
	import favicon from '$lib/assets/favicon.svg';
	import { resolve } from '$app/paths';

	let { children } = $props();
</script>

<svelte:head><link rel="icon" href={favicon} /></svelte:head>

<div class="flex flex-col h-full">
	<header
		class="h-(--header-height) flex items-center justify-between p-4 w-full max-w-screen-xl mx-auto"
	>
		<div class="flex items-center gap-4">
			<a class="text-xl mr-4" href={resolve('/')}>sveltekit.fyi</a>
			<a href={resolve('/hits')}>
				<div>sites</div>
			</a>
		</div>

		<nav class="flex items-center gap-4">
			<a
				href="https://github.com/maybemaby"
				target="_blank"
				rel="noopener noreferrer"
				class="font-mono">Source</a
			>
		</nav>
	</header>
	<main class="flex-1 p-4 w-full max-w-screen-xl mx-auto">
		{@render children()}
	</main>
</div>
