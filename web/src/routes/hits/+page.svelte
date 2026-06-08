<script lang="ts">
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

	let total = $derived(domains[0]?.total ?? 0);
	let pageCount = $derived(Math.max(1, Math.ceil(total / perPage)));
</script>

<div class="h-full">
	<div class="grid grid-cols-3 gap-2">
		{#each domains as domain (domain.domain)}
			<Preview domain={domain.domain} title={domain.title} />
		{/each}
	</div>

	{#if pageCount > 1}
		<div class="my-8 flex justify-center">
			<Pagination.Root count={total} {perPage} page={pg} siblingCount={1}>
				{#snippet children({ pages, currentPage })}
					<Pagination.Content>
						<Pagination.Item>
							<Pagination.Previous href={`?page=${Math.max(1, currentPage - 1)}`} />
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
							<Pagination.Next href={`?page=${Math.min(pageCount, currentPage + 1)}`} />
						</Pagination.Item>
					</Pagination.Content>
				{/snippet}
			</Pagination.Root>
		</div>
	{/if}
</div>
