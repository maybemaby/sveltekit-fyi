<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import Preview from '$lib/components/preview.svelte';
	import * as Pagination from '$lib/components/ui/pagination/index.js';
	import { cn } from '$lib/utils';
	import { getDomains, getStats, type DomainListingOrder } from '../scans.remote';

	const perPage = 30;

	let pg = $derived.by(() => {
		const pageParam = page.url.searchParams.get('page');

		return pageParam ? parseInt(pageParam) : 1;
	});

	const validOrders = new Set<DomainListingOrder>(['seen_at', 'seen_count']);

	let order = $derived.by<DomainListingOrder>(() => {
		const orderParam = page.url.searchParams.get('order');

		if (orderParam && validOrders.has(orderParam as DomainListingOrder)) {
			return orderParam as DomainListingOrder;
		}

		return 'seen_at';
	});

	let stats = await getStats();

	let domains = $derived(
		await getDomains({
			page: pg,
			order
		})
	);

	let datedDomains = $derived.by(() => {
		return domains.map((d) => {
			const currentUnixSeconds = Math.floor(Date.now() / 1000);
			let timeAgo: string;
			const secondsAgo = currentUnixSeconds - d.first_seen_at;

			const hoursAgo = Math.floor(secondsAgo / 3600);

			if (hoursAgo >= 24) {
				const daysAgo = Math.floor(hoursAgo / 24);
				timeAgo = `${daysAgo} day${daysAgo !== 1 ? 's' : ''} ago`;
			} else if (hoursAgo < 1) {
				const minutesAgo = Math.floor(secondsAgo / 60);
				timeAgo = `${minutesAgo} minute${minutesAgo !== 1 ? 's' : ''} ago`;
			} else {
				timeAgo = `${hoursAgo} hour${hoursAgo !== 1 ? 's' : ''} ago`;
			}

			return {
				...d,
				timeAgo
			};
		});
	});

	let total = $derived(domains[0]?.total ?? 0);
	let pageCount = $derived(Math.max(1, Math.ceil(total / perPage)));

	async function handlePageChange(nextPage: number) {
		const url = new URL(page.url);

		if (nextPage <= 1) {
			url.searchParams.delete('page');
		} else {
			url.searchParams.set('page', String(nextPage));
		}

		await goto(url, {
			noScroll: true,
			keepFocus: true
		});
	}

	async function handleOrderChange(newOrder: DomainListingOrder) {
		const url = new URL(page.url);

		if (newOrder === 'seen_at') {
			url.searchParams.delete('order');
		} else {
			url.searchParams.set('order', newOrder);
		}

		await goto(url, {
			noScroll: true,
			keepFocus: true
		});
	}
</script>

<div class="h-full">
	<p class="mb-2 text-muted-foreground">{stats.scans.confirmedSites} sites</p>
	<div class="flex mb-4">
		<button
			class={cn('p-2 border', order === 'seen_at' && 'bg-primary/10 text-primary border-primary')}
			onclick={() => handleOrderChange('seen_at')}>Last Seen</button
		>
		<button
			class={cn(
				'p-2 border border-l-0',
				order === 'seen_count' && 'bg-primary/10 text-primary border-primary border-l'
			)}
			onclick={() => handleOrderChange('seen_count')}>Most Mentions on Bluesky</button
		>
	</div>

	<div class="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 gap-2">
		{#each datedDomains as domain (domain.domain)}
			<Preview
				domain={domain.domain}
				title={domain.title}
				timeAgo={domain.timeAgo}
				image={domain.og_image || domain.screenshot_path}
				is_sk={domain.is_sk}
				is_svelte={domain.is_svelte}
			/>
		{/each}
	</div>

	{#if pageCount > 1}
		<div class="my-8 flex justify-center">
			<Pagination.Root
				count={total}
				{perPage}
				page={pg}
				siblingCount={1}
				onPageChange={handlePageChange}
			>
				{#snippet children({ pages, currentPage })}
					<Pagination.Content>
						<Pagination.Item>
							<Pagination.Previous />
						</Pagination.Item>

						{#each pages as pageItem (pageItem.key)}
							{#if pageItem.type === 'ellipsis'}
								<Pagination.Item>
									<Pagination.Ellipsis />
								</Pagination.Item>
							{:else}
								<Pagination.Item>
									<Pagination.Link page={pageItem} isActive={currentPage === pageItem.value}>
										{pageItem.value}
									</Pagination.Link>
								</Pagination.Item>
							{/if}
						{/each}

						<Pagination.Item>
							<Pagination.Next />
						</Pagination.Item>
					</Pagination.Content>
				{/snippet}
			</Pagination.Root>
		</div>
	{/if}
</div>
