<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import Preview from '$lib/components/preview.svelte';
	import * as Pagination from '$lib/components/ui/pagination/index.js';
	import { getDomains } from '../scans.remote';

	const perPage = 30;

	let pg = $derived.by(() => {
		const pageParam = page.url.searchParams.get('page');

		return pageParam ? parseInt(pageParam) : 1;
	});

	let domains = $derived(
		await getDomains({
			page: pg
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

	function handlePageChange(nextPage: number) {
		const url = new URL(page.url);

		if (nextPage <= 1) {
			url.searchParams.delete('page');
		} else {
			url.searchParams.set('page', String(nextPage));
		}

		void goto(url, {
			noScroll: true,
			keepFocus: true
		});
	}
</script>

<div class="h-full">
	<div class="grid grid-cols-3 gap-2">
		{#each datedDomains as domain (domain.domain)}
			<Preview domain={domain.domain} title={domain.title} timeAgo={domain.timeAgo} />
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
									<Pagination.Link
										href={`?page=${pageItem.value}`}
										page={pageItem}
										isActive={currentPage === pageItem.value}
									>
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
