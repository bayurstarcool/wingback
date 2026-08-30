<script lang="ts">
	/* eslint-disable svelte/no-navigation-without-resolve */
	import { onMount, onDestroy } from 'svelte';
	import { page } from '$app/state';
	import { messages, streamMessage, getToken, type Message, type LiveEvent } from '$lib/api/client';
	import { formatCountdown, formatDistance } from '$lib/delivery';

	let msg = $state<Message | null>(null);
	let countdown = $state('');
	let status = $state<'loading' | 'arrived' | 'lost' | 'in_transit'>('loading');
	let error = $state('');
	let mapEl: HTMLDivElement | undefined = $state(undefined);
	let privateMapEl: HTMLDivElement | undefined = $state(undefined);
	let leaflet: typeof import('leaflet') | null = null;
	let map: import('leaflet').Map | null = null;
	let carrierMarker: import('leaflet').CircleMarker | null = null;
	let countdownInterval: ReturnType<typeof setInterval> | null = null;
	let streamCleanup: (() => void) | null = null;
	let lastUpdated = $state('Menunggu koneksi live...');
	let privateProgress = $state(0);
	let privatePhase = $state('Berangkat');
	let privateBird = $state({ x: 50, y: 50, rotation: 0 });
	let privateRoutePoints = $state<Array<[number, number]>>([]);
	let privateAnimationFrame: number | null = null;
	let shareNote = $state('');

	onMount(async () => {
		if (!getToken()) {
			window.location.href = '/login';
			return;
		}
		const id = page.params.id;
		if (!id) {
			error = 'ID pesan tidak ada.';
			return;
		}

		try {
			msg = await messages.get(id);
			status = msg.status === 'delivered' ? 'arrived' : msg.status;
			countdown = formatCountdown(msg.arrives_at);
		} catch (e) {
			error = e instanceof Error ? e.message : 'Pesan tidak ditemukan.';
			return;
		}

		leaflet = await import('leaflet');
		await import('leaflet/dist/leaflet.css');

		if (msg.location_privacy === 'hidden') {
			if (privateMapEl) {
				map = leaflet
					.map(privateMapEl, {
						zoomControl: true,
						attributionControl: true,
						dragging: false,
						scrollWheelZoom: true,
						doubleClickZoom: true,
						boxZoom: true,
						touchZoom: true,
						keyboard: true,
						zoomSnap: 0.5,
						zoomDelta: 1,
						minZoom: 4,
						maxZoom: 12
					})
					.setView([-2.5, 118], 4);
				leaflet
					.tileLayer('https://tile.openstreetmap.org/{z}/{x}/{y}.png', {
						attribution: '© OpenStreetMap',
						maxZoom: 6
					})
					.addTo(map);
				if (msg.from_map_lat != null && msg.from_map_lng != null) {
					const origin: [number, number] = [msg.from_map_lat, msg.from_map_lng];
					const sameCity = msg.same_city === true;
					const destination: [number, number] =
						!sameCity && msg.to_map_lat != null && msg.to_map_lng != null
							? [msg.to_map_lat, msg.to_map_lng]
							: origin;
					leaflet
						.circle(origin, {
							radius: sameCity ? 16000 : 12000,
							color: '#8c5f4b',
							weight: 3,
							fillColor: '#e7bda9',
							fillOpacity: 0.42
						})
						.addTo(map);
					leaflet
						.circleMarker(origin, {
							radius: 5,
							color: '#fffaf5',
							weight: 2,
							fillColor: '#806957',
							fillOpacity: 0.95
						})
						.addTo(map)
						.bindTooltip(privateOriginLabel(), {
							permanent: true,
							direction: 'top',
							className: 'city-label'
						});
					if (!sameCity) {
						leaflet
							.circle(destination, {
								radius: 12000,
								color: '#b54f32',
								weight: 3,
								fillColor: '#e9aa95',
								fillOpacity: 0.42
							})
							.addTo(map);
						leaflet
							.circleMarker(destination, {
								radius: 5,
								color: '#fffaf5',
								weight: 2,
								fillColor: '#c36040',
								fillOpacity: 0.95
							})
							.addTo(map)
							.bindTooltip(privateDestinationLabel(), {
								permanent: true,
								direction: 'top',
								className: 'city-label'
							});
						leaflet
							.polyline(buildPrivateRoute(origin, destination, sameCity), {
								color: '#fffaf5',
								weight: 16,
								opacity: 0.92,
								lineCap: 'round'
							})
							.addTo(map);
						leaflet
							.polyline(buildPrivateRoute(origin, destination, sameCity), {
								color: '#c36040',
								weight: 7,
								dashArray: '15 11',
								lineCap: 'round',
								opacity: 1
							})
							.addTo(map);
					}
					privateRoutePoints = buildPrivateRoute(origin, destination, sameCity);
					updatePrivateBird(privateProgress);
					const fitRoute = () => {
						map?.invalidateSize({ pan: false });
						if (sameCity) {
							map?.setView(origin, 11, { animate: false });
							return;
						}
						map?.fitBounds([origin, destination], {
							padding: [34, 34],
							maxZoom: 10,
							animate: false
						});
					};
					fitRoute();
					requestAnimationFrame(fitRoute);
				}
			}
			privateProgress = flightProgress(msg);
			privatePhase = phaseFor(privateProgress);
			startPrivateBirdAnimation();
			streamCleanup = streamMessage(id, handleLiveEvent);
			countdownInterval = setInterval(() => {
				if (msg && status === 'in_transit') {
					countdown = formatCountdown(msg.arrives_at);
					privateProgress = flightProgress(msg);
					privatePhase = phaseFor(privateProgress);
				}
			}, 1000);
			return;
		}

		if (
			!mapEl ||
			!msg ||
			msg.sender_lat == null ||
			msg.sender_lng == null ||
			msg.recipient_lat == null ||
			msg.recipient_lng == null
		)
			return;

		const start: [number, number] = [msg.sender_lat, msg.sender_lng];
		const end: [number, number] = [msg.recipient_lat, msg.recipient_lng];
		map = leaflet.map(mapEl, { zoomControl: false, attributionControl: true });
		leaflet.control.zoom({ position: 'bottomright' }).addTo(map);
		leaflet
			.tileLayer('https://tile.openstreetmap.org/{z}/{x}/{y}.png', {
				attribution: '© OpenStreetMap',
				maxZoom: 18
			})
			.addTo(map);
		leaflet
			.polyline([start, end], { color: '#d96d49', weight: 2, dashArray: '6 9', opacity: 0.8 })
			.addTo(map);
		leaflet
			.circleMarker(start, {
				radius: 7,
				color: '#fffaf5',
				weight: 3,
				fillColor: '#302820',
				fillOpacity: 1
			})
			.addTo(map)
			.bindTooltip('Berangkat');
		leaflet
			.circleMarker(end, {
				radius: 7,
				color: '#fffaf5',
				weight: 3,
				fillColor: '#d96d49',
				fillOpacity: 1
			})
			.addTo(map)
			.bindTooltip('Tujuan');
		map.fitBounds([start, end], { padding: [42, 42] });

		streamCleanup = streamMessage(id, handleLiveEvent);
		countdownInterval = setInterval(() => {
			if (msg && status === 'in_transit') countdown = formatCountdown(msg.arrives_at);
		}, 1000);
	});

	function handleLiveEvent(event: LiveEvent) {
		lastUpdated = `Update live · ${new Date(event.at).toLocaleTimeString('id-ID', { hour: '2-digit', minute: '2-digit' })}`;
		if (event.type === 'progress') {
			privateProgress = event.progress ?? privateProgress;
			privatePhase = event.phase ?? phaseFor(privateProgress);
			updatePrivateBird(privateProgress);
			status = privateProgress >= 1 ? 'arrived' : 'in_transit';
			if (privatePhase === 'Perjalanan terhenti') status = 'lost';
			if (status !== 'in_transit') stopCountdown();
		} else if (event.type === 'position' && event.lat != null && event.lng != null) {
			status = 'in_transit';
			if (map && leaflet) {
				if (!carrierMarker) {
					carrierMarker = leaflet
						.circleMarker([event.lat, event.lng], {
							radius: 9,
							color: '#fffaf5',
							weight: 3,
							fillColor: '#d96d49',
							fillOpacity: 1
						})
						.addTo(map)
						.bindTooltip('Carrier Wingback');
				} else carrierMarker.setLatLng([event.lat, event.lng]);
			}
		} else if (
			(event.type === 'arrived' || event.type === 'delivered') &&
			event.lat != null &&
			event.lng != null
		) {
			status = 'arrived';
			carrierMarker?.setLatLng([event.lat, event.lng]);
			lastUpdated = 'Pesan baru saja tiba';
			stopCountdown();
		} else if (event.type === 'lost') {
			status = 'lost';
			lastUpdated = 'Perjalanan berakhir';
			stopCountdown();
		}
	}

	function flightProgress(message: Message) {
		const total = new Date(message.arrives_at).getTime() - new Date(message.departs_at).getTime();
		if (total <= 0) return 1;
		return Math.min(1, Math.max(0, (Date.now() - new Date(message.departs_at).getTime()) / total));
	}

	function phaseFor(progress: number) {
		if (progress >= 0.72) return 'Mendekati tujuan';
		if (progress >= 0.18) return 'Sedang melintas';
		return 'Berangkat';
	}

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
		if (privateAnimationFrame) cancelAnimationFrame(privateAnimationFrame);
	});

	function buildPrivateRoute(
		origin: [number, number],
		destination: [number, number],
		sameCity: boolean
	) {
		if (sameCity)
			return [origin, [origin[0] + 0.12, origin[1] + 0.16], origin] as Array<[number, number]>;
		const bend = Math.max(0.18, Math.min(0.5, Math.abs(destination[1] - origin[1]) * 0.22));
		const mid: [number, number] = [
			(origin[0] + destination[0]) / 2 + bend,
			(origin[1] + destination[1]) / 2
		];
		const points: Array<[number, number]> = [];
		for (let index = 0; index <= 24; index += 1) {
			const t = index / 24;
			const inverse = 1 - t;
			points.push([
				inverse * inverse * origin[0] + 2 * inverse * t * mid[0] + t * t * destination[0],
				inverse * inverse * origin[1] + 2 * inverse * t * mid[1] + t * t * destination[1]
			]);
		}
		return points;
	}

	function updatePrivateBird(progress: number) {
		if (!privateRoutePoints.length) return;
		const bounded = Math.max(0, Math.min(1, progress));
		const position = bounded * (privateRoutePoints.length - 1);
		const index = Math.min(privateRoutePoints.length - 2, Math.floor(position));
		const fraction = position - index;
		const current = privateRoutePoints[index];
		const next = privateRoutePoints[index + 1];
		const lat = current[0] + (next[0] - current[0]) * fraction;
		const lng = current[1] + (next[1] - current[1]) * fraction;
		const point = map?.latLngToContainerPoint([lat, lng]);
		const width = privateMapEl?.clientWidth || 1;
		const height = privateMapEl?.clientHeight || 1;
		privateBird = {
			x: point ? (point.x / width) * 100 : 50,
			y: point ? (point.y / height) * 100 : 50,
			rotation: Math.atan2(next[1] - current[1], next[0] - current[0]) * (180 / Math.PI) - 90
		};
	}

	function startPrivateBirdAnimation() {
		if (!msg || msg.location_privacy !== 'hidden') return;
		const tick = () => {
			if (status === 'in_transit') updatePrivateBird(privateProgress);
			privateAnimationFrame = requestAnimationFrame(tick);
		};
		tick();
	}

	function statusTitle() {
		if (status === 'arrived') return 'Pesan sudah tiba.';
		if (status === 'lost') return 'Pesan hilang di jalan.';
		return 'Pesan sedang terbang.';
	}

	async function shareJourney() {
		const url = window.location.href;
		try {
			if (navigator.share) {
				await navigator.share({
					title: 'Flight log · Wingback',
					text: 'Lihat pesan yang sedang menempuh perjalanan di Wingback.',
					url
				});
				shareNote = 'Perjalanan berhasil dibagikan.';
			} else if (navigator.clipboard) {
				await navigator.clipboard.writeText(url);
				shareNote = 'Link perjalanan tersalin.';
			} else {
				shareNote = 'Salin link dari alamat halaman tracking.';
			}
		} catch (e) {
			if (e instanceof DOMException && e.name === 'AbortError') return;
			shareNote = 'Link perjalanan belum bisa dibagikan. Coba lagi.';
		}
	}
	function statusCopy() {
		if (status === 'arrived') return 'Perjalanan selesai. Buka pesanmu dan nikmati momennya.';
		if (status === 'lost') return 'Carrier tidak berhasil menyelesaikan perjalanan ini.';
		return 'Jangan buru-buru. Ia sedang menempuh jarak menuju penerima.';
	}
	function privateRouteLabel() {
		if (!msg) return 'Perjalanan privat';
		if (msg.same_city && msg.from_label) return `${msg.from_label} · Perjalanan dalam kota`;
		if (msg.from_label && msg.to_label) return `${msg.from_label} → ${msg.to_label}`;
		return 'Kota asal → kota tujuan';
	}
	function privateOriginLabel() {
		return msg?.same_city ? msg?.from_label || 'Kota asal' : msg?.from_label || 'Kota asal';
	}
	function privateDestinationLabel() {
		return msg?.same_city ? 'Dalam kota' : msg?.to_label || 'Kota tujuan';
	}
</script>

<svelte:head><title>Lacak perjalanan · Wingback</title></svelte:head>

<main class="tracking-page">
	{#if error}
		<section class="tracking-error" role="alert">
			<span>×</span>
			<div>
				<strong>Perjalanan tidak ditemukan</strong>
				<p>{error}</p>
			</div>
			<a href="/inbox">Kembali ke kotak masuk</a>
		</section>
	{:else if msg}
		<header class="track-heading">
			<div class="track-topline">
				<a href="/inbox" class="back-link">← Kotak masuk</a>
				<button type="button" class="share-button" onclick={shareJourney}
					>Bagikan perjalanan <span>↗</span></button
				>
			</div>
			{#if shareNote}<p class="share-note track-share-note" role="status">{shareNote}</p>{/if}
			<div class="track-heading-row">
				<div>
					<p class="eyebrow">FLIGHT LOG / {msg.id.slice(0, 8).toUpperCase()}</p>
					<h1>{statusTitle()}</h1>
					<p class="heading-copy">{statusCopy()}</p>
					<span
						class:private={msg.location_privacy === 'hidden'}
						class:accurate={msg.location_privacy === 'accurate'}
						class="privacy-badge"
						>{msg.location_privacy === 'hidden'
							? '◌ Area privat aktif'
							: '⌖ Rute akurat aktif'}</span
					>
				</div>
				<span
					class:flight={status === 'in_transit'}
					class:arrived={status === 'arrived'}
					class:lost={status === 'lost'}
					class="big-status"
					>{status === 'in_transit'
						? '↗ Sedang terbang'
						: status === 'arrived'
							? '✓ Sudah tiba'
							: '× Hilang'}</span
				>
			</div>
		</header>

		<section class="track-layout">
			<div class="map-panel">
				{#if msg.location_privacy === 'hidden'}
					<div class="private-map" aria-label="Peta area privat">
						<div bind:this={privateMapEl} class="private-map-tiles"></div>
						<div class="private-map-wash"></div>
						<div class="private-grid"></div>
						<div class="private-signal" aria-hidden="true"></div>
						<div
							class="private-bird"
							class:arrived={status === 'arrived'}
							style={`left: ${privateBird.x}%; top: ${privateBird.y}%; transform: translate(-50%, -50%) rotate(${privateBird.rotation}deg);`}
							aria-hidden="true"
						>
							⌁
						</div>
						<div
							class="private-center"
							class:arrived={status === 'arrived'}
							class:lost={status === 'lost'}
						>
							<span>✦</span><strong>{privatePhase}</strong><small>{privateRouteLabel()}</small>
							<small>Lokasi carrier disembunyikan</small>
							<div class="private-progress" aria-label="Progress perjalanan">
								<span style:width={`${Math.round(privateProgress * 100)}%`}></span>
							</div>
						</div>
					</div>
				{:else}
					<div bind:this={mapEl} class="track-map"></div>
				{/if}
				{#if msg.location_privacy === 'hidden'}
					<div class="map-legend private-legend">
						<span class="legend-line"></span> Kota terlihat · GPS carrier tidak dibagikan
					</div>
				{:else}
					<div class="map-legend">
						<span class="legend-dot start"></span> Berangkat <span class="legend-line"></span> Rute
						pesan <span class="legend-dot end"></span> Tujuan
					</div>
				{/if}
			</div>
			<aside class="track-side">
				<div class="eta-card" class:done={status === 'arrived'} class:failed={status === 'lost'}>
					<p class="card-kicker">
						{status === 'in_transit' ? 'ESTIMASI TIBA' : 'STATUS PERJALANAN'}
					</p>
					<strong
						>{status === 'in_transit'
							? countdown
							: status === 'arrived'
								? 'Sudah sampai'
								: 'Tidak tersampaikan'}</strong
					><span>{formatDistance(msg.distance_km)} · {msg.speed_kmh} km/jam</span>
				</div>
				<div class="privacy-card">
					<span class="privacy-card-icon">{msg.location_privacy === 'hidden' ? '◌' : '⌖'}</span>
					<div>
						<strong>{msg.location_privacy === 'hidden' ? 'Area aman' : 'Rute akurat'}</strong><small
							>{msg.location_privacy === 'hidden'
								? 'Kota asal dan tujuan terlihat. Lokasi detail tetap rahasia.'
								: 'Titik perjalanan dibagikan secara detail kepada penerima.'}</small
						>
					</div>
				</div>
				<div class="timeline-card">
					<p class="card-kicker">PERJALANAN</p>
					<div class="timeline">
						<div class="timeline-item complete">
							<span class="timeline-dot">✓</span>
							<div>
								<strong>Pesan dilepaskan</strong><small
									>{new Date(msg.departs_at).toLocaleString('id-ID', {
										day: 'numeric',
										month: 'short',
										hour: '2-digit',
										minute: '2-digit'
									})}</small
								>
							</div>
						</div>
						<div
							class="timeline-item"
							class:active={status === 'in_transit'}
							class:complete={status === 'arrived'}
							class:failed={status === 'lost'}
						>
							<span class="timeline-dot"
								>{status === 'arrived' ? '✓' : status === 'lost' ? '×' : '↗'}</span
							>
							<div>
								<strong
									>{status === 'arrived'
										? 'Tiba di tujuan'
										: status === 'lost'
											? 'Perjalanan terhenti'
											: 'Menuju penerima'}</strong
								><small>{lastUpdated}</small>
							</div>
						</div>
					</div>
				</div>
				<div class="message-card">
					<div class="message-card-top">
						<p class="card-kicker">ISI SURAT</p>
						<span>{msg.body.length} karakter</span>
					</div>
					<blockquote>“{msg.body}”</blockquote>
				</div>
			</aside>
		</section>
		<footer class="track-footer">
			<span>Jarak dihitung dengan GPS nyata</span><span>Carrier: Wingback flight system</span>
		</footer>
	{:else}
		<section class="track-loading">
			<span>◌</span>
			<p>Membuka catatan perjalanan...</p>
		</section>
	{/if}
</main>

<style>
	.tracking-page {
		max-width: 1120px;
		margin: auto;
		padding: 48px 28px 75px;
	}
	.back-link {
		display: inline-block;
		margin-bottom: 34px;
		color: #aa9d90;
		font-size: 12px;
		font-weight: 700;
		text-decoration: none;
	}
	.back-link:hover {
		color: #c36040;
	}
	.track-heading {
		margin-bottom: 34px;
	}
	.track-topline {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 18px;
		margin-bottom: 28px;
	}
	.track-topline .back-link {
		margin-bottom: 0;
	}
	.share-button {
		border: 1px solid #d96d49;
		border-radius: 99px;
		background: #d96d49;
		padding: 10px 14px;
		color: #fffaf5;
		font: inherit;
		font-size: 11px;
		font-weight: 800;
		cursor: pointer;
		transition: 0.2s;
	}
	.share-button:hover {
		border-color: #bd593a;
		background: #c85e3b;
		transform: translateY(-1px);
	}
	.track-share-note {
		margin: -16px 0 18px;
		text-align: right;
	}
	.eyebrow,
	.card-kicker {
		color: #bc6243;
		font-size: 10px;
		font-weight: 800;
		letter-spacing: 0.17em;
	}
	.track-heading-row {
		display: flex;
		align-items: flex-end;
		justify-content: space-between;
		gap: 24px;
	}
	.track-heading h1 {
		margin: 12px 0 10px;
		color: #302820;
		font-family: Georgia, serif;
		font-size: clamp(38px, 5vw, 62px);
		font-weight: 500;
		letter-spacing: -0.06em;
		line-height: 0.98;
	}
	.heading-copy {
		max-width: 470px;
		margin: 0;
		color: #918579;
		font-size: 14px;
		line-height: 1.6;
	}
	.privacy-badge {
		display: inline-flex;
		margin-top: 14px;
		border-radius: 99px;
		padding: 7px 10px;
		font-size: 10px;
		font-weight: 800;
	}
	.privacy-badge.private {
		background: #f1e9df;
		color: #806957;
	}
	.privacy-badge.accurate {
		background: #fff0e7;
		color: #c36040;
	}
	.big-status {
		border-radius: 99px;
		background: #f0ebe5;
		padding: 10px 14px;
		color: #7d7369;
		font-size: 11px;
		font-weight: 800;
		white-space: nowrap;
	}
	.big-status.flight {
		background: #fff0e7;
		color: #c36040;
	}
	.big-status.arrived {
		background: #e8f0e9;
		color: #4c7a5c;
	}
	.big-status.lost {
		background: #f9e8e5;
		color: #b25b4c;
	}
	.track-layout {
		display: grid;
		grid-template-columns: minmax(0, 1.45fr) minmax(320px, 0.75fr);
		gap: 18px;
		align-items: start;
	}
	.map-panel {
		position: relative;
		overflow: hidden;
		min-height: 430px;
		height: 430px;
		border: 1px solid #e2d8ce;
		border-radius: 20px;
		background: #e9e3dc;
		box-shadow: 0 20px 60px rgba(69, 49, 31, 0.08);
	}
	.track-map {
		width: 100%;
		height: 430px;
	}
	.private-map {
		position: relative;
		display: grid;
		min-height: 430px;
		height: 430px;
		overflow: hidden;
		place-items: center;
		background: #e4d9cd;
	}
	.private-map-tiles,
	.private-map-wash {
		position: absolute;
		inset: 0;
	}
	.private-map-tiles {
		opacity: 0.74;
		filter: saturate(0.76) sepia(0.1) contrast(1.02);
	}
	.private-map-wash {
		z-index: 0;
		background:
			radial-gradient(circle at 50% 44%, rgba(255, 245, 235, 0.5), transparent 38%),
			rgba(248, 239, 229, 0.42);
		pointer-events: none;
		animation: private-wash 12s ease-in-out infinite alternate;
	}
	.private-map > :not(.private-map-tiles) {
		z-index: 1;
	}
	.private-map > .private-grid {
		z-index: 2;
		pointer-events: none;
	}
	.private-map > .private-center {
		z-index: 3;
		pointer-events: none;
	}
	.private-map :global(.city-label) {
		border: 1px solid rgba(104, 82, 65, 0.42);
		border-radius: 99px;
		background: rgba(255, 250, 244, 0.94);
		box-shadow: 0 4px 12px rgba(57, 40, 25, 0.14);
		padding: 6px 10px;
		color: #493d34;
		font-size: 11px;
		font-weight: 800;
	}
	.private-grid {
		position: absolute;
		inset: 0;
		opacity: 0.34;
		background-image:
			linear-gradient(rgba(133, 113, 95, 0.12) 1px, transparent 1px),
			linear-gradient(90deg, rgba(133, 113, 95, 0.12) 1px, transparent 1px);
		background-size: 44px 44px;
		transform: rotate(-8deg) scale(1.2);
		animation: private-grid-drift 20s linear infinite;
	}
	.private-center {
		position: absolute;
		z-index: 1;
		top: 82%;
		display: grid;
		justify-items: center;
		gap: 5px;
		transform: translate(-50%, -50%);
		border: 1px solid rgba(255, 250, 244, 0.9);
		border-radius: 14px;
		background: rgba(255, 250, 244, 0.84);
		padding: 12px 16px;
		box-shadow: 0 8px 24px rgba(57, 40, 25, 0.13);
	}
	.private-center > span {
		color: #d96d49;
		font-size: 27px;
	}
	.private-center strong {
		color: #51473e;
		font-size: 14px;
	}
	.private-center small {
		color: #978b7f;
		font-size: 10px;
	}
	.map-legend {
		position: absolute;
		right: 14px;
		top: 14px;
		left: auto;
		display: flex;
		align-items: center;
		gap: 8px;
		border: 1px solid rgba(255, 255, 255, 0.72);
		border-radius: 10px;
		background: rgba(255, 253, 249, 0.9);
		padding: 9px 11px;
		color: #84786c;
		font-size: 10px;
		font-weight: 700;
		box-shadow: 0 5px 20px rgba(57, 40, 25, 0.1);
	}
	.private-legend {
		bottom: auto;
		max-width: min(300px, calc(100% - 28px));
		border-color: rgba(255, 255, 255, 0.88);
		background: rgba(255, 250, 244, 0.94);
		color: #5f5147;
	}
	.legend-dot {
		width: 8px;
		height: 8px;
		border: 2px solid #fffaf5;
		border-radius: 99px;
		box-shadow: 0 0 0 1px #302820;
	}
	.legend-dot.start {
		background: #302820;
	}
	.legend-dot.end {
		background: #d96d49;
		box-shadow: 0 0 0 1px #d96d49;
	}
	.legend-line {
		width: 30px;
		height: 1px;
		margin-left: 6px;
		background: repeating-linear-gradient(90deg, #d96d49 0 5px, transparent 5px 9px);
	}
	.private-bird {
		position: absolute;
		z-index: 4;
		width: 30px;
		height: 30px;
		border: 2px solid #fffaf5;
		border-radius: 50%;
		background: #d96d49;
		box-shadow:
			0 5px 16px rgba(168, 78, 47, 0.3),
			0 0 0 6px rgba(217, 109, 73, 0.14);
		color: #fffaf5;
		font-size: 25px;
		font-weight: 800;
		line-height: 23px;
		text-align: center;
		pointer-events: none;
		transition:
			left 0.35s linear,
			top 0.35s linear;
		animation: bird-bob 0.9s ease-in-out infinite alternate;
	}
	.private-bird.arrived {
		background: #4c7a5c;
		animation: bird-arrived 1.6s ease-in-out infinite;
	}
	.private-signal {
		position: absolute;
		top: 50%;
		left: 50%;
		width: 116px;
		height: 116px;
		border: 1px solid rgba(217, 109, 73, 0.22);
		border-radius: 50%;
		transform: translate(-50%, -50%);
		animation: signal-expand 3.2s ease-out infinite;
		pointer-events: none;
	}
	:global(.private-center.arrived .private-signal),
	:global(.private-center.lost .private-signal) {
		animation-play-state: paused;
		opacity: 0.35;
	}
	.private-center > span {
		animation: star-float 2.4s ease-in-out infinite;
	}
	.private-center.arrived > span,
	.private-center.lost > span {
		animation: none;
	}
	.private-progress {
		width: min(180px, 70vw);
		height: 4px;
		margin-top: 5px;
		overflow: hidden;
		border-radius: 99px;
		background: rgba(195, 95, 62, 0.14);
	}
	.private-progress span {
		display: block;
		height: 100%;
		border-radius: inherit;
		background: linear-gradient(90deg, #9c705c, #d96d49);
		transition: width 0.8s ease;
	}
	.private-center.arrived .private-progress span {
		background: #4c7a5c;
	}
	.private-center.lost .private-progress span {
		background: #b25b4c;
	}
	@keyframes bird-bob {
		from {
			margin-top: 0;
		}
		to {
			margin-top: -4px;
		}
	}
	@keyframes bird-arrived {
		0%,
		100% {
			scale: 1;
		}
		50% {
			scale: 1.08;
		}
	}
	@keyframes private-wash {
		from {
			opacity: 0.72;
		}
		to {
			opacity: 1;
		}
	}
	@keyframes private-grid-drift {
		from {
			background-position: 0 0;
		}
		to {
			background-position: 44px 44px;
		}
	}
	@keyframes route-dash {
		to {
			stroke-dashoffset: -52px;
		}
	}
	@keyframes route-breathe {
		0%,
		100% {
			opacity: 0.72;
		}
		50% {
			opacity: 1;
		}
	}
	@keyframes signal-expand {
		0% {
			opacity: 0.7;
			transform: translate(-50%, -50%) scale(0.72);
		}
		70%,
		100% {
			opacity: 0;
			transform: translate(-50%, -50%) scale(1.35);
		}
	}
	@keyframes star-float {
		0%,
		100% {
			transform: translateY(0) rotate(0);
		}
		50% {
			transform: translateY(-4px) rotate(8deg);
		}
	}
	.private-map :global(.leaflet-overlay-pane path[stroke='#c36040']) {
		animation:
			route-dash 1.8s linear infinite,
			route-breathe 2.8s ease-in-out infinite;
	}
	.track-side {
		display: grid;
		align-content: start;
		gap: 12px;
	}
	.eta-card,
	.timeline-card,
	.message-card {
		border: 1px solid #e7ded4;
		border-radius: 16px;
		background: rgba(255, 253, 249, 0.78);
		padding: 21px;
	}
	.eta-card {
		background: #fff2eb;
		border-color: #f0d7c9;
	}
	.eta-card.done {
		background: #edf4ee;
		border-color: #d7e7d9;
	}
	.eta-card.failed {
		background: #faece9;
		border-color: #efd1cb;
	}
	.eta-card strong {
		display: block;
		margin: 12px 0 5px;
		color: #c15f3e;
		font-family: Georgia, serif;
		font-size: 32px;
		font-weight: 500;
		letter-spacing: -0.04em;
	}
	.eta-card.done strong {
		color: #4c7a5c;
	}
	.eta-card.failed strong {
		color: #b25b4c;
	}
	.eta-card span {
		color: #9d8779;
		font-size: 11px;
	}
	.privacy-card {
		display: flex;
		gap: 10px;
		border: 1px solid #e4dbd1;
		border-radius: 14px;
		background: #faf7f2;
		padding: 14px;
	}
	.privacy-card-icon {
		color: #c36040;
		font-size: 21px;
		line-height: 1;
	}
	.privacy-card div {
		display: grid;
		gap: 4px;
	}
	.privacy-card strong {
		color: #5b5047;
		font-size: 11px;
	}
	.privacy-card small {
		color: #9e9287;
		font-size: 10px;
		line-height: 1.45;
	}
	.timeline-card .card-kicker {
		margin-bottom: 19px;
	}
	.timeline {
		display: grid;
		gap: 0;
	}
	.timeline-item {
		display: flex;
		position: relative;
		gap: 12px;
		min-height: 59px;
	}
	.timeline-item:first-child::after {
		content: '';
		position: absolute;
		top: 25px;
		bottom: 0;
		left: 11px;
		width: 1px;
		background: #d8cfc5;
	}
	.timeline-dot {
		z-index: 1;
		display: grid;
		width: 23px;
		height: 23px;
		flex: 0 0 23px;
		place-items: center;
		border: 1px solid #d8cfc5;
		border-radius: 50%;
		background: #fffdf9;
		color: #aa9c8f;
		font-size: 11px;
	}
	.timeline-item.complete .timeline-dot {
		border-color: #b8d2bc;
		background: #e8f0e9;
		color: #4c7a5c;
	}
	.timeline-item.active .timeline-dot {
		border-color: #e8b49f;
		background: #fff0e7;
		color: #c36040;
		box-shadow: 0 0 0 4px #fdf0e9;
	}
	.timeline-item.failed .timeline-dot {
		border-color: #e6b8af;
		background: #f9e8e5;
		color: #b25b4c;
	}
	.timeline-item div {
		display: grid;
		align-content: start;
		gap: 5px;
	}
	.timeline-item strong {
		color: #5b5047;
		font-size: 12px;
	}
	.timeline-item small {
		color: #a59a8f;
		font-size: 10px;
	}
	.message-card-top {
		display: flex;
		justify-content: space-between;
	}
	.message-card-top > span {
		color: #afa398;
		font-size: 10px;
	}
	.message-card blockquote {
		margin: 13px 0 0;
		color: #5b5047;
		font-family: Georgia, serif;
		font-size: 16px;
		line-height: 1.55;
		overflow-wrap: anywhere;
	}
	.track-footer {
		display: flex;
		justify-content: space-between;
		margin-top: 16px;
		color: #a69a8e;
		font-size: 10px;
	}
	.tracking-error {
		display: flex;
		align-items: flex-start;
		gap: 13px;
		max-width: 650px;
		margin: 90px auto;
		border: 1px solid #efd1cb;
		border-radius: 14px;
		background: #fff3f0;
		padding: 18px;
		color: #b25b4c;
	}
	.tracking-error > span {
		display: grid;
		width: 25px;
		height: 25px;
		place-items: center;
		border-radius: 50%;
		background: #f4d8d1;
		font-size: 18px;
	}
	.tracking-error div {
		flex: 1;
	}
	.tracking-error strong {
		font-size: 13px;
	}
	.tracking-error p {
		margin: 5px 0 0;
		font-size: 11px;
	}
	.tracking-error a {
		color: #b25b4c;
		font-size: 11px;
		font-weight: 800;
		text-decoration: none;
	}
	.track-loading {
		padding: 130px 20px;
		color: #a6998d;
		text-align: center;
	}
	.track-loading span {
		display: block;
		color: #d5a18d;
		font-family: Georgia, serif;
		font-size: 40px;
	}
	.track-loading p {
		font-size: 13px;
	}
	:global(.leaflet-container) {
		font-family: inherit;
	}
	.track-map :global(.leaflet-control-zoom a) {
		color: #554a40;
	}
	.private-map :global(.leaflet-control-zoom) {
		margin-top: 58px;
		margin-left: 14px;
		border: 1px solid rgba(104, 82, 65, 0.2);
		border-radius: 10px;
		overflow: hidden;
		box-shadow: 0 5px 16px rgba(57, 40, 25, 0.14);
	}
	.private-map :global(.leaflet-control-zoom a) {
		width: 30px;
		height: 30px;
		line-height: 28px;
		border-color: rgba(104, 82, 65, 0.16);
		background: rgba(255, 250, 244, 0.94);
		color: #554a40;
	}
	@media (max-width: 820px) {
		.track-topline {
			margin-bottom: 22px;
		}
	}
	@media (max-width: 620px) {
		.track-topline {
			align-items: flex-start;
			gap: 10px;
		}
		.share-button {
			padding: 9px 11px;
			font-size: 10px;
			white-space: nowrap;
		}
		.track-share-note {
			margin-top: -10px;
			font-size: 10px;
		}
		.tracking-page {
			padding: 34px 16px 60px;
		}
		.track-heading-row {
			align-items: flex-start;
			flex-direction: column;
			gap: 18px;
		}
		.track-heading h1 {
			font-size: 43px;
		}
		.track-layout {
			grid-template-columns: 1fr;
		}
		.map-panel,
		.track-map,
		.private-map {
			min-height: 320px;
			height: 320px;
		}
		.map-legend {
			overflow-x: auto;
			white-space: nowrap;
		}
		.track-footer {
			align-items: flex-start;
			flex-direction: column;
			gap: 7px;
		}
		.tracking-error {
			align-items: flex-start;
			flex-wrap: wrap;
			margin: 55px auto;
		}
		.tracking-error a {
			margin-left: 39px;
		}
	}
	:global(.tracking-page) {
		color: #302820;
	}
</style>
