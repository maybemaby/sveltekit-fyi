import { Result, TaggedError, UnhandledException } from 'better-result';

class HttpError extends TaggedError('HttpError')<{
	status: number;
}>() {}

function safeFetch<T>(
	input: Parameters<typeof fetch>[0],
	init?: Parameters<typeof fetch>[1]
): Promise<Result<T, HttpError | UnhandledException>> {
	return Result.tryPromise({
		try: async () => {
			const res = await fetch(input, init);

			if (!res.ok) {
				throw new HttpError({
					status: res.status
				});
			}

			return (await res.json()) as T;
		},
		catch: (err) => {
			if (err instanceof HttpError) {
				return err;
			}
			return new UnhandledException({ cause: err });
		}
	});
}

interface ScanStats {
	confirmedSites: number;
	totalScans: number;
	totalObserved: number;
}

interface SignalCount {
	signals: string;
	count: number;
}

interface CombinedStats {
	scans: ScanStats;
	signals: SignalCount[];
}

interface DomainListing {
	domain: string;
	first_seen_at: number;
	last_seen_at: number;
	seen_count: number;
	signals: string;
	title: string;
	og_image: string | null;
	total: number;
}

interface SiteCountSnapshot {
	snapshotAt: number;
	confirmedSites: number;
	totalScans: number;
	totalObserved: number;
}

type ClientFetchOptions = Omit<Parameters<typeof fetch>[1], 'method' | 'body'>;

export class ApiClient {
	private baseUrl: string;
	private fetchOptions?: ClientFetchOptions;

	constructor(baseUrl: string, fetchOptions?: ClientFetchOptions) {
		this.baseUrl = baseUrl;
		this.fetchOptions = fetchOptions;
	}

	async getScanStats() {
		return safeFetch<CombinedStats>(`${this.baseUrl}/stats`, this.fetchOptions);
	}

	async getDomainListings(page: number = 1, order: 'seen_at' | 'seen_count' = 'seen_at') {
		return safeFetch<DomainListing[]>(
			`${this.baseUrl}/scans?page=${page}&order=${order}`,
			this.fetchOptions
		);
	}

	async getSnapshots() {
		return safeFetch<SiteCountSnapshot[]>(`${this.baseUrl}/snapshots`, this.fetchOptions);
	}
}

export const apiClient = new ApiClient(process.env.API_HOST || 'http://localhost:8080');
