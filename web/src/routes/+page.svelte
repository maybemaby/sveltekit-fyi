<script lang="ts">
	import { Plot, Line } from 'svelteplot';
	import StatBox from '$lib/components/stat-box.svelte';
	import { getSiteCountSnapshots, getStats } from './scans.remote';

	let stats = await getStats();
	let snapshots = await getSiteCountSnapshots();
	let utcSnapshots = $derived.by(() =>
		snapshots.map((s) => ({ ...s, snapshotAt: s.snapshotAt * 1000 }))
	);

	let signals = $derived.by(() =>
		stats.signals.map((s) => ({
			...s,
			count: Intl.NumberFormat('en-US', {
				notation: 'standard'
			}).format(s.count)
		}))
	);
</script>

<div>
	<h1 class="mb-4 text-2xl font-semibold">Overview</h1>
	<div class="grid md:grid-cols-3 gap-2 mb-4">
		<StatBox count={stats.scans.confirmedSites} caption="Confirmed Sveltekit Sites" />
		<StatBox count={stats.scans.totalScans} caption="Domains Scanned" />
		<StatBox count={stats.scans.totalObserved} caption="Domains Observed" />
	</div>

	<div class="mb-4">
		<h2 class="mb-4 text-xl font-semibold">SvelteKit Sites Detected Over Time</h2>
		<Plot y={{ grid: true, label: false }} x={{ label: 'Time', type: 'utc' }} marginTop={20}>
			<Line lineClass="text-primary" data={utcSnapshots} x="snapshotAt" y="confirmedSites" />
		</Plot>
	</div>

	<h2 class="text-xl font-semibold mb-2">Signals used to detect Sveltekit</h2>
	<div class="grid grid-cols-[300px_1fr] gap-1">
		{#each signals as signal (signal.signals)}
			<div class="font-mono">{signal.signals}</div>
			<div class="font-mono">{signal.count}</div>
		{/each}
	</div>
</div>
