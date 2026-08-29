<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { page } from '$app/state';
	import { messages, streamMessage, getToken, type Message, type LiveEvent } from '$lib/api/client';
	import { formatCountdown } from '$lib/delivery';

	let msg = $state<Message | null>(null);
	let countdown = $state('');
	let status = $state<'loading' | 'arrived' | 'lost' | 'in_transit'>('loading');
	let error = $state('');

	let mapEl: HTMLDivElement | undefined = $state(undefined);
	let leaflet: typeof import('leaflet') | null = null;
	let map: import('leaflet').Map | null = null;
	let carrierMarker: import('leaflet').Marker | null = null;
	let streamCleanup: (() => void) | null = null;
	let countdownInterval: ReturnType<typeof setInterval> | null = null;

	onMount(async () => {
		if (!getToken()) {
			window.location.href = '/login';
			return;
		}
		const id = page.params.id;
		if (!id) {
			error = 'ID pesan tidak ada';
			return;
		}

		try {
			msg = await messages.get(id);
		} catch (e) {
			error = e instanceof Error ? e.message : 'Pesan tidak ditemukan';
			return;
		}

		// Initialise Leaflet only in browser context.
		leaflet = await import('leaflet');
		await import('leaflet/dist/leaflet.css');

		if (!mapEl) return;
		map = leaflet!
			.map(mapEl, {
				zoomControl: true,
				attributionControl: true
			})
			.setView([0, 0], 2);

		leaflet!
			.tileLayer('https://tile.openstreetmap.org/{z}/{x}/{y}.png', {
				attribution: '© OpenStreetMap',
				maxZoom: 18
			})
			.addTo(map!);

		// Subscribe to live updates. First position event sets the view.
		streamCleanup = streamMessage(id, (e: LiveEvent) => {
			if (e.type === 'arrived') {
				status = 'arrived';
				carrierMarker?.setLatLng([e.lat, e.lng]);
				stopCountdown();
				return;
			}
			if (e.type === 'lost') {
				status = 'lost';
				stopCountdown();
				return;
			}
			if (e.type === 'position') {
				status = 'in_transit';
				if (!carrierMarker) {
					carrierMarker = leaflet!
						.marker([e.lat, e.lng], { title: 'Carrier' })
						.addTo(map!)
						.bindPopup('🕊️ Carrier');
				} else {
					carrierMarker.setLatLng([e.lat, e.lng]);
				}
				map!.setView([e.lat, e.lng], 5);
			}
		});

		countdownInterval = setInterval(() => {
			if (msg) countdown = formatCountdown(msg.arrives_at);
		}, 1000);
		if (msg) countdown = formatCountdown(msg.arrives_at);
	});

	function stopCountdown() {
		if (countdownInterval) {
			clearInterval(countdownInterval);
			countdownInterval = null;
		}
	}

	onDestroy(() => {
		streamCleanup?.();
		stopCountdown();
		map?.remove();
	});
</script>

<svelte:head>
	<title>Tracking · Wingback</title>
</svelte:head>

<main class="mx-auto max-w-3xl px-4 py-6">
	{#if error}
		<p class="text-red-600">{error}</p>
	{/if}

	{#if msg}
		<div class="mb-4 rounded-lg border border-gray-200 bg-white p-4">
			<div class="flex items-center justify-between">
				<div>
					<p class="text-sm text-gray-500">
						{msg.sender_id.slice(0, 8)} → {msg.recipient_id.slice(0, 8)}
					</p>
					<p class="mt-1 text-gray-800">{msg.body}</p>
				</div>
				<div class="text-right text-sm">
					{#if status === 'in_transit'}
						<p class="rounded bg-amber-100 px-2 py-1 text-amber-800">🕊️ {countdown}</p>
					{:else if status === 'arrived'}
						<p class="rounded bg-green-100 px-2 py-1 text-green-800">✓ Sampai</p>
					{:else if status === 'lost'}
						<p class="rounded bg-red-100 px-2 py-1 text-red-800">💀 Hilang di jalan</p>
					{/if}
					<p class="mt-1 text-xs text-gray-500">{msg.distance_km.toFixed(1)} km</p>
				</div>
			</div>
		</div>

		<div bind:this={mapEl} class="h-96 w-full rounded-lg border border-gray-200"></div>
	{:else if !error}
		<p class="text-gray-500">Memuat...</p>
	{/if}
</main>
