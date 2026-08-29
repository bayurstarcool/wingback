// Centralized API client with auth-token plumbing. Token is stored in
// localStorage and added as `Authorization: Bearer` on every request.
// For a real product consider httpOnly cookies to defeat XSS, but for
// an MVP at this stage the trade-off (simpler dev experience) wins.

// Default to same-origin relative paths: in production nginx proxies
// /api/* to the Go backend, so no CORS and no env plumbing needed.
// Override with VITE_API_BASE_URL for local dev (e.g. http://localhost:8080).
const API_BASE = import.meta.env.VITE_API_BASE_URL ?? '';

const TOKEN_KEY = 'wingback_token';
const USER_KEY = 'wingback_user';

export class ApiError extends Error {
	status: number;
	constructor(status: number, message: string) {
		super(message);
		this.status = status;
	}
}

export interface User {
	user_id: string;
	username: string;
	email: string;
	display_name: string;
}

export interface UserSearchResult {
	id: string;
	username: string;
	email: string;
	display_name: string;
	last_lat?: number | null;
	last_lng?: number | null;
	location_ready: boolean;
}

export interface AuthResponse {
	token: string;
	expires_at: string;
	user_id: string;
	username: string;
	email: string;
	display_name: string;
}

export interface Carrier {
	slug: string;
	name: string;
	speed_kmh: number;
	price: number;
	rarity: string;
}

export type MessageStatus = 'in_transit' | 'delivered' | 'lost';

export interface Message {
	id: string;
	sender_id: string;
	recipient_id: string;
	body: string;
	distance_km: number;
	speed_kmh: number;
	sender_lat?: number | null;
	sender_lng?: number | null;
	recipient_lat?: number | null;
	recipient_lng?: number | null;
	location_privacy: 'accurate' | 'hidden';
	from_label?: string;
	to_label?: string;
	same_city?: boolean;
	status: MessageStatus;
	departs_at: string;
	arrives_at: string;
	delivered_at?: string | null;
}

export interface LiveEvent {
	type: 'position' | 'progress' | 'arrived' | 'lost' | 'in_transit' | 'delivered';
	message_id: string;
	lat?: number;
	lng?: number;
	progress?: number;
	phase?: string;
	at: string;
}

export function getToken(): string | null {
	if (typeof localStorage === 'undefined') return null;
	return localStorage.getItem(TOKEN_KEY);
}

export function getStoredUser(): User | null {
	if (typeof localStorage === 'undefined') return null;
	const raw = localStorage.getItem(USER_KEY);
	return raw ? (JSON.parse(raw) as User) : null;
}

export function setSession(auth: AuthResponse): void {
	localStorage.setItem(TOKEN_KEY, auth.token);
	localStorage.setItem(
		USER_KEY,
		JSON.stringify({
			user_id: auth.user_id,
			username: auth.username,
			email: auth.email,
			display_name: auth.display_name
		})
	);
}

export function clearSession(): void {
	localStorage.removeItem(TOKEN_KEY);
	localStorage.removeItem(USER_KEY);
}

async function request<T>(path: string, init: RequestInit = {}): Promise<T> {
	const token = getToken();
	const headers = new Headers(init.headers);
	headers.set('Content-Type', 'application/json');
	if (token) headers.set('Authorization', `Bearer ${token}`);

	const res = await fetch(`${API_BASE}${path}`, { ...init, headers });
	if (!res.ok) {
		const text = await res.text().catch(() => res.statusText);
		throw new ApiError(res.status, text);
	}
	return res.json() as Promise<T>;
}

export const auth = {
	async register(
		email: string,
		username: string,
		password: string,
		displayName: string
	): Promise<AuthResponse> {
		return request<AuthResponse>('/api/auth/register', {
			method: 'POST',
			body: JSON.stringify({ email, username, password, display_name: displayName })
		});
	},
	async login(username: string, password: string): Promise<AuthResponse> {
		return request<AuthResponse>('/api/auth/login', {
			method: 'POST',
			body: JSON.stringify({ username, password })
		});
	},
	async me(): Promise<User> {
		return request<User>('/api/auth/me');
	}
};

export const messages = {
	async compose(input: {
		recipient_id: string;
		body: string;
		carrier_slug?: string;
		sender_lat: number;
		sender_lng: number;
		location_privacy?: 'accurate' | 'hidden';
	}): Promise<{
		message_id: string;
		distance_km: number;
		speed_kmh: number;
		departs_at: string;
		arrives_at: string;
		will_be_lost: boolean;
		carrier: string;
	}> {
		return request('/api/messages', { method: 'POST', body: JSON.stringify(input) });
	},
	async listInbox(): Promise<Message[]> {
		return request<Message[]>('/api/messages/inbox');
	},
	async listSent(): Promise<Message[]> {
		return request<Message[]>('/api/messages/sent');
	},
	async get(id: string): Promise<Message> {
		return request<Message>(`/api/messages/${id}`);
	},
	async searchUsers(query: string): Promise<UserSearchResult[]> {
		return request<UserSearchResult[]>(`/api/users/search?q=${encodeURIComponent(query)}`);
	},
	async updateLocation(lat: number, lng: number, accuracyM?: number): Promise<void> {
		await request('/api/auth/location', {
			method: 'POST',
			body: JSON.stringify({ lat, lng, accuracy_m: accuracyM })
		});
	}
};

export const carriers = {
	async list(): Promise<Carrier[]> {
		return request<Carrier[]>('/api/carriers');
	}
};

export function streamMessage(id: string, onEvent: (e: LiveEvent) => void): () => void {
	const token = getToken();
	// Build an absolute ws(s):// URL. In production API_BASE is ''
	// (same-origin), so derive the host from window.location.
	const base = API_BASE || window.location.origin;
	const url = `${base}/api/messages/${id}/stream`;
	const wsUrl = url.replace(/^http/, 'ws') + (token ? `?token=${encodeURIComponent(token)}` : '');
	const ws = new WebSocket(wsUrl);

	ws.onmessage = (ev) => {
		try {
			onEvent(JSON.parse(ev.data) as LiveEvent);
		} catch {
			// ignore malformed payloads
		}
	};

	return () => {
		if (ws.readyState === WebSocket.OPEN || ws.readyState === WebSocket.CONNECTING) {
			ws.close();
		}
	};
}
