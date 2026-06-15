import { query } from '$app/server';
import { apiClient } from '$lib/api';
import * as v from 'valibot';

export const getDomains = query(
	v.object({
		page: v.optional(v.pipe(v.number(), v.integer())),
		order: v.optional(v.picklist(['seen_at', 'seen_count']), 'seen_at')
	}),
	async (data) => {
		const res = await apiClient.getDomainListings(data.page, data.order);

		if (res.isErr()) {
			throw res.error;
		}

		return res.value;
	}
);

export const getStats = query(async () => {
	const res = await apiClient.getScanStats();

	if (res.isErr()) {
		throw res.error;
	}

	return res.value;
});

export const getSiteCountSnapshots = query(async () => {
	const res = await apiClient.getSnapshots();

	if (res.isErr()) {
		throw res.error;
	}

	return res.value;
});
