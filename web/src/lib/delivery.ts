// Shared formatting helpers for delivery ETAs, mirrored conceptually
// from the Go backend's delivery engine so the UI can render countdowns
// without re-fetching every second.

export function formatCountdown(arrivesAt: string, now: Date = new Date()): string {
	const diffMs = new Date(arrivesAt).getTime() - now.getTime();
	if (diffMs <= 0) return 'Sudah sampai';

	const totalSeconds = Math.floor(diffMs / 1000);
	const days = Math.floor(totalSeconds / 86400);
	const hours = Math.floor((totalSeconds % 86400) / 3600);
	const minutes = Math.floor((totalSeconds % 3600) / 60);
	const seconds = totalSeconds % 60;

	if (days > 0) return `${days}h ${hours}j ${minutes}m`;
	if (hours > 0) return `${hours}j ${minutes}m`;
	if (minutes > 0) return `${minutes}m ${seconds}d`;
	return `${seconds}d`;
}

export function formatDistance(km: number): string {
	if (km < 1) return `${Math.round(km * 1000)} m`;
	return `${km.toFixed(1)} km`;
}
