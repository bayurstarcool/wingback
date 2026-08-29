// API client for the Wingback backend message-composition endpoint.
// Kept dependency-free (native fetch) so it works in SvelteKit's
// browser and SSR contexts alike.

export interface ComposeRequest {
	recipient_id: string;
	body: string;
	carrier_slug?: string;
	sender_lat: number;
	sender_lng: number;
	recipient_lat: number;
	recipient_lng: number;
}

export interface ComposeResponse {
	message_id: string;
	distance_km: number;
	speed_kmh: number;
	departs_at: string;
	arrives_at: string;
	will_be_lost: boolean;
}

export class ApiError extends Error {
	status: number;
	constructor(status: number, message: string) {
		super(message);
		this.status = status;
	}
}

const API_BASE = import.meta.env.VITE_API_BASE_URL ?? 'http://localhost:8090';

export async function composeMessage(req: ComposeRequest): Promise<ComposeResponse> {
	const res = await fetch(`${API_BASE}/api/messages`, {
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify(req)
	});

	if (!res.ok) {
		const text = await res.text().catch(() => res.statusText);
		throw new ApiError(res.status, text);
	}

	return res.json();
}
