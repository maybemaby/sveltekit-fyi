<script lang="ts">
	import StatBox from '$lib/components/stat-box.svelte';
	import { getStats } from './scans.remote';

	const stats = await getStats();
</script>

<div>
	<h1 class="mb-4 text-2xl font-semibold">Overview</h1>
	<div class="grid md:grid-cols-3 gap-2">
		<StatBox count={stats.confirmedSites} caption="Confirmed Sveltekit Sites" />
		<StatBox count={stats.totalScans} caption="Domains Scanned" />
		<StatBox count={stats.totalObserved} caption="Domains Observed" />
	</div>
</div>
