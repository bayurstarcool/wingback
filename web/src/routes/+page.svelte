<script lang="ts">
	/* eslint-disable svelte/no-navigation-without-resolve */
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import {
		carriers,
		messages,
		getToken,
		type Carrier,
		type UserSearchResult
	} from '$lib/api/client';
	import { formatCountdown } from '$lib/delivery';

	let recipientQuery = $state('');
	let recipientId = $state('');

	let selectedRecipient = $state<UserSearchResult | null>(null);
	let suggestions = $state<UserSearchResult[]>([]);
	let body = $state('');
	let carrierSlug = $state('pigeon');
	let senderLat = $state<number | null>(null);
	let senderLng = $state<number | null>(null);
	let senderAccuracy = $state<number | null>(null);
	let locationPrivacy = $state<'accurate' | 'hidden'>('hidden');
	let carrierList = $state<Carrier[]>([]);
	let sending = $state(false);
	let searching = $state(false);
	let error = $state('');
	let locationNote = $state('Mendeteksi lokasi kamu...');
	let lastResult = $state<{
		message_id: string;
		arrives_at: string;
		will_be_lost: boolean;
		carrier: string;
		distance_km: number;
	} | null>(null);
	let countdown = $state('');
	let shareNote = $state('');
	let searchTimer: ReturnType<typeof setTimeout> | null = null;
	let countdownTimer: ReturnType<typeof setInterval> | null = null;

	onMount(() => {
		if (!getToken()) {
			goto('/login');
			return;
		}
		carriers
			.list()
			.then((cs) => (carrierList = cs))
			.catch(() => (error = 'Carrier belum bisa dimuat. Coba refresh.'));

		if (navigator.geolocation) {
			navigator.geolocation.getCurrentPosition(
				(pos) => {
					setSenderLocation(pos);
				},
				() => {
					locationNote = 'GPS belum diizinkan';
				},
				{ timeout: 5000 }
			);
		} else {
			locationNote = 'Browser tidak mendukung GPS';
		}

		return () => {
			if (searchTimer) clearTimeout(searchTimer);
			if (countdownTimer) clearInterval(countdownTimer);
		};
	});

	function searchRecipients() {
		if (searchTimer) clearTimeout(searchTimer);
		const query = recipientQuery.trim();
		selectedRecipient = null;
		recipientId = '';
		if (query.length < 2) {
			suggestions = [];
			searching = false;
			return;
		}
		searching = true;
		searchTimer = setTimeout(async () => {
			try {
				suggestions = await messages.searchUsers(query);
			} catch {
				suggestions = [];
			} finally {
				searching = false;
			}
		}, 250);
	}

	function chooseRecipient(user: UserSearchResult) {
		selectedRecipient = user;
		recipientQuery = user.display_name;
		recipientId = user.id;
		suggestions = [];
	}

	function setSenderLocation(pos: GeolocationPosition) {
		senderLat = pos.coords.latitude;
		senderLng = pos.coords.longitude;
		senderAccuracy = Math.round(pos.coords.accuracy);
		locationNote = `Lokasi siap · akurasi ±${senderAccuracy} m`;
		messages.updateLocation(senderLat, senderLng, senderAccuracy).catch(() => {});
	}

	function useMyLocation() {
		if (!navigator.geolocation) {
			error = 'Browser tidak mendukung lokasi.';
			return;
		}
		navigator.geolocation.getCurrentPosition(
			(pos) => {
				setSenderLocation(pos);
			},
			() => (error = 'Lokasi tidak diizinkan. Izinkan GPS sebelum mengirim pesan.')
		);
	}

	async function handleSubmit(e: Event) {
		e.preventDefault();
		error = '';
		lastResult = null;
		if (!recipientId || !body.trim() || senderLat === null || senderLng === null) {
			error = 'Pilih penerima, izinkan GPS, lalu tulis pesan dulu.';
			return;
		}
		if (!selectedRecipient?.location_ready) {
			error = 'Penerima belum menetapkan lokasi tujuan.';
			return;
		}
		sending = true;
		try {
			const result = await messages.compose({
				recipient_id: recipientId,
				body: body.trim(),
				carrier_slug: carrierSlug,
				sender_lat: senderLat,
				sender_lng: senderLng,
				location_privacy: locationPrivacy
			});
			lastResult = result;
			countdown = formatCountdown(result.arrives_at);
			if (countdownTimer) clearInterval(countdownTimer);
			countdownTimer = setInterval(() => {
				if (lastResult) countdown = formatCountdown(lastResult.arrives_at);
			}, 1000);
		} catch (e) {
			error = e instanceof Error ? e.message : 'Gagal mengirim pesan.';
		} finally {
			sending = false;
		}
	}

	async function shareJourney(messageId: string) {
		const url = `${window.location.origin}/track/${messageId}`;
		try {
			if (navigator.share) {
				await navigator.share({
					title: 'Pesan sedang terbang · Wingback',
					text: 'Aku mengirim pesan yang punya perjalanan. Lihat flight log-nya di Wingback.',
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
</script>

<svelte:head>
	<title>Tulis surat · Wingback</title>
</svelte:head>

<main class="compose-shell">
	<section class="compose-hero">
		<div class="eyebrow"><span class="eyebrow-dot"></span> Pesan yang sengaja menunggu</div>
		<h1>Kirim sesuatu yang<br /><em>punya perjalanan.</em></h1>
		<p>Wingback membawa pesanmu melintasi jarak nyata. Bukan instan. Lebih personal.</p>
		<div class="hero-route" aria-label="Perjalanan pesan">
			<div><span class="route-dot route-dot-start"></span><small>Lokasi kamu</small></div>
			<span class="route-line"></span>
			<div>
				<span class="route-dot route-dot-end"></span><small
					>{selectedRecipient?.display_name ?? 'Seseorang'}</small
				>
			</div>
		</div>
	</section>

	<section class="compose-card">
		<div class="card-heading">
			<div>
				<p class="card-kicker">SURAT BARU</p>
				<h2>Siapa yang ingin kamu sapa?</h2>
			</div>
			<span class="card-index">01 / 03</span>
		</div>

		<form onsubmit={handleSubmit}>
			<div class="field recipient-field">
				<label for="recipient-search">Penerima</label>
				<div class="input-wrap" class:has-selection={!!selectedRecipient}>
					<span class="input-icon">⌕</span>
					<input
						id="recipient-search"
						type="search"
						bind:value={recipientQuery}
						oninput={searchRecipients}
						placeholder="Cari @username..."
						autocomplete="off"
						required
					/>
					{#if selectedRecipient}<span class="selected-check">✓</span>{/if}
				</div>
				{#if searching}<p class="field-hint">Mencari...</p>{/if}
				{#if suggestions.length > 0}
					<div class="suggestions" role="listbox" aria-label="Hasil pencarian penerima">
						{#each suggestions as user (user.id)}
							<button
								type="button"
								role="option"
								aria-selected="false"
								onclick={() => chooseRecipient(user)}
							>
								<span class="avatar">{user.display_name.slice(0, 1).toUpperCase()}</span>
								<span
									><strong>@{user.username}</strong><small
										>{user.display_name} · {user.location_ready
											? 'lokasi siap'
											: 'lokasi belum diatur'}</small
									></span
								>
								<span class="suggest-arrow">↗</span>
							</button>
						{/each}
					</div>
				{:else if recipientQuery.trim().length >= 2 && !searching && !selectedRecipient}
					<p class="field-hint">Belum menemukan orang ini.</p>
				{/if}
			</div>

			<div class="field message-field">
				<div class="label-row">
					<label for="body">Pesanmu</label><span>{body.length} / 2000</span>
				</div>
				<textarea
					id="body"
					bind:value={body}
					rows="5"
					maxlength="2000"
					placeholder="Tulis sesuatu yang layak ditunggu..."
					required></textarea>
			</div>

			<div class="section-divider"></div>
			<div class="field">
				<div class="label-row">
					<span class="field-label">Pengantar pilihanmu</span><span>Kecepatan bukan tujuan</span>
				</div>
				<div class="carrier-grid">
					{#each carrierList as carrier (carrier.slug)}
						<button
							type="button"
							class:selected={carrierSlug === carrier.slug}
							class="carrier-option"
							onclick={() => (carrierSlug = carrier.slug)}
						>
							<span class="carrier-glyph"
								>{carrier.slug === 'pigeon'
									? '◉'
									: carrier.slug === 'falcon'
										? '✦'
										: carrier.slug === 'drone'
											? '⊞'
											: '◈'}</span
							>
							<span class="carrier-copy"
								><strong>{carrier.name}</strong><small>{carrier.speed_kmh} km/jam</small></span
							>
							{#if carrierSlug === carrier.slug}<span class="carrier-selected">✓</span>{/if}
						</button>
					{/each}
				</div>
			</div>

			<div class="location-row">
				<div class="location-copy">
					<span class="location-pin">⌖</span><span
						><strong>{locationNote}</strong><small>Jarak dihitung dari GPS asli</small></span
					>
				</div>
				<button type="button" class="location-btn" onclick={useMyLocation}>Perbarui lokasi</button>
			</div>
			{#if selectedRecipient && !selectedRecipient.location_ready}
				<p class="recipient-location-warning" role="status">
					<span>⌖</span><span
						><strong>@{selectedRecipient.username} belum mengatur lokasi tujuan.</strong><small
							>Minta mereka membuka Wingback dan memperbarui lokasi sebelum pesan dikirim.</small
						></span
					>
				</p>
			{/if}

			<div class="privacy-section">
				<div class="label-row">
					<span class="field-label">Privasi perjalanan</span><span>Pilih sebelum pesan dilepas</span
					>
				</div>
				<div class="privacy-grid" role="radiogroup" aria-label="Privasi perjalanan">
					<button
						type="button"
						class:active={locationPrivacy === 'hidden'}
						class="privacy-option"
						role="radio"
						aria-checked={locationPrivacy === 'hidden'}
						onclick={() => (locationPrivacy = 'hidden')}
					>
						<span class="privacy-icon">◌</span><span
							><strong>Area privat</strong><small
								>Kota asal dan tujuan terlihat. Lokasi detail tetap rahasia.</small
							></span
						><span class="privacy-check">{locationPrivacy === 'hidden' ? '✓' : ''}</span>
					</button>
					<button
						type="button"
						class:active={locationPrivacy === 'accurate'}
						class="privacy-option"
						role="radio"
						aria-checked={locationPrivacy === 'accurate'}
						onclick={() => (locationPrivacy = 'accurate')}
					>
						<span class="privacy-icon">⌖</span><span
							><strong>Titik akurat</strong><small>Rute detail terlihat di halaman tracking.</small
							></span
						><span class="privacy-check">{locationPrivacy === 'accurate' ? '✓' : ''}</span>
					</button>
				</div>
				<p class="privacy-note">
					{locationPrivacy === 'hidden'
						? 'Mode aman aktif · lokasi asli tidak ditampilkan di peta.'
						: 'Mode detail aktif · gunakan hanya jika kamu percaya penerima.'}
				</p>
			</div>

			{#if error}<p class="error-box" role="alert">{error}</p>{/if}
			<button
				type="submit"
				class="send-button"
				disabled={sending ||
					!selectedRecipient ||
					!selectedRecipient.location_ready ||
					senderLat === null}
			>
				<span>{sending ? 'Menyiapkan perjalanan...' : 'Kirim dalam perjalanan'}</span><span
					class="send-arrow">→</span
				>
			</button>
		</form>
	</section>

	{#if lastResult}
		<section class="result-card">
			<div class="result-icon">✦</div>
			<div class="result-copy">
				<p class="card-kicker">PERJALANAN DIMULAI</p>
				<h2>Pesanmu sedang terbang.</h2>
				<p>Tiba dalam <strong>{countdown}</strong> · {lastResult.distance_km.toFixed(0)} km</p>
			</div>
			<div class="result-actions">
				<a class="result-link" href={`/track/${lastResult.message_id}`}
					>Lihat perjalanan <span>→</span></a
				>
				<button
					class="share-link"
					type="button"
					onclick={() => shareJourney(lastResult!.message_id)}
				>
					Bagikan <span>↗</span>
				</button>
			</div>
			{#if shareNote}<small class="share-note" role="status">{shareNote}</small>{/if}
		</section>
	{/if}
</main>

<style>
	.compose-shell {
		max-width: 1120px;
		margin: 0 auto;
		padding: 72px 28px 96px;
	}
	.compose-hero {
		max-width: 680px;
		margin: 0 auto 42px;
		text-align: center;
	}
	.eyebrow,
	.card-kicker {
		color: #756b5e;
		font-size: 10px;
		font-weight: 800;
		letter-spacing: 0.18em;
	}
	.eyebrow {
		display: inline-flex;
		align-items: center;
		gap: 8px;
		margin-bottom: 22px;
	}
	.eyebrow-dot {
		width: 7px;
		height: 7px;
		border-radius: 99px;
		background: #e1744f;
		box-shadow: 0 0 0 4px #f9e5dc;
	}
	.compose-hero h1 {
		margin: 0;
		color: #27241f;
		font-family: Georgia, 'Times New Roman', serif;
		font-size: clamp(42px, 7vw, 76px);
		font-weight: 500;
		letter-spacing: -0.055em;
		line-height: 0.98;
	}
	.compose-hero h1 em {
		color: #d96d49;
		font-style: italic;
	}
	.compose-hero p {
		max-width: 450px;
		margin: 22px auto 0;
		color: #82796e;
		font-size: 15px;
		line-height: 1.7;
	}
	.hero-route {
		display: flex;
		align-items: center;
		justify-content: center;
		gap: 14px;
		margin-top: 30px;
		color: #958b7e;
	}
	.hero-route > div {
		display: flex;
		align-items: center;
		gap: 7px;
		font-size: 11px;
	}
	.route-dot {
		width: 9px;
		height: 9px;
		border: 2px solid #d96d49;
		border-radius: 99px;
	}
	.route-dot-end {
		background: #d96d49;
	}
	.route-line {
		width: 70px;
		height: 1px;
		background: repeating-linear-gradient(90deg, #d5c9bb 0 5px, transparent 5px 9px);
	}
	.compose-card {
		max-width: 700px;
		margin: 0 auto;
		padding: 34px;
		border: 1px solid #e7ded3;
		border-radius: 24px;
		background: rgba(255, 253, 249, 0.88);
		box-shadow: 0 24px 80px rgba(78, 58, 38, 0.08);
	}
	.card-heading {
		display: flex;
		align-items: flex-start;
		justify-content: space-between;
		margin-bottom: 30px;
	}
	.card-heading h2,
	.result-card h2 {
		margin: 7px 0 0;
		color: #302b25;
		font-family: Georgia, serif;
		font-size: 25px;
		font-weight: 500;
		letter-spacing: -0.025em;
	}
	.card-index {
		color: #afa397;
		font-size: 11px;
		font-weight: 700;
		letter-spacing: 0.1em;
	}
	.field {
		position: relative;
		margin-bottom: 24px;
	}
	.field label,
	.field-label {
		display: block;
		margin-bottom: 9px;
		color: #5b534a;
		font-size: 12px;
		font-weight: 750;
	}
	.label-row {
		display: flex;
		align-items: center;
		justify-content: space-between;
	}
	.label-row span,
	.field-hint {
		color: #aaa095;
		font-size: 11px;
	}
	.input-wrap {
		display: flex;
		align-items: center;
		border: 1px solid #ded4c9;
		border-radius: 11px;
		background: #fffefa;
		transition:
			border-color 0.2s,
			box-shadow 0.2s;
	}
	.input-wrap:focus-within,
	.input-wrap.has-selection {
		border-color: #d96d49;
		box-shadow: 0 0 0 3px #f8e4da;
	}
	.input-icon {
		padding-left: 14px;
		color: #b2a79b;
		font-size: 22px;
		line-height: 1;
	}
	.input-wrap input {
		min-width: 0;
		width: 100%;
		border: 0;
		outline: 0;
		background: transparent;
		padding: 14px 10px;
		color: #302b25;
		font: inherit;
		font-size: 14px;
	}
	.input-wrap input::-webkit-search-cancel-button {
		display: none;
	}
	.selected-check {
		display: grid;
		width: 22px;
		height: 22px;
		margin-right: 12px;
		place-items: center;
		border-radius: 99px;
		background: #d96d49;
		color: white;
		font-size: 12px;
	}
	.suggestions {
		position: absolute;
		z-index: 5;
		top: 74px;
		right: 0;
		left: 0;
		overflow: hidden;
		border: 1px solid #ded4c9;
		border-radius: 12px;
		background: #fffefa;
		box-shadow: 0 18px 36px rgba(66, 48, 31, 0.12);
	}
	.suggestions button {
		display: flex;
		align-items: center;
		width: 100%;
		gap: 11px;
		border: 0;
		border-bottom: 1px solid #f0eae3;
		background: transparent;
		padding: 12px 14px;
		text-align: left;
		cursor: pointer;
	}
	.suggestions button:last-child {
		border-bottom: 0;
	}
	.suggestions button:hover {
		background: #fff3ec;
	}
	.avatar {
		display: grid;
		width: 32px;
		height: 32px;
		flex: 0 0 32px;
		place-items: center;
		border-radius: 50%;
		background: #f6dfd3;
		color: #c25f3d;
		font-family: Georgia, serif;
		font-size: 16px;
	}
	.suggestions button span:nth-child(2) {
		display: grid;
		gap: 3px;
	}
	.suggestions strong {
		color: #453d35;
		font-size: 13px;
	}
	.suggestions small {
		color: #a1988e;
		font-size: 11px;
	}
	.suggest-arrow {
		margin-left: auto;
		color: #c2b6aa;
	}
	.field-hint {
		margin: 7px 0 0;
	}
	textarea {
		display: block;
		box-sizing: border-box;
		width: 100%;
		resize: vertical;
		border: 1px solid #ded4c9;
		border-radius: 11px;
		outline: 0;
		background: #fffefa;
		padding: 14px;
		color: #302b25;
		font: inherit;
		font-size: 14px;
		line-height: 1.6;
		transition:
			border-color 0.2s,
			box-shadow 0.2s;
	}
	textarea:focus {
		border-color: #d96d49;
		box-shadow: 0 0 0 3px #f8e4da;
	}
	textarea::placeholder,
	input::placeholder {
		color: #b9afa4;
	}
	.section-divider {
		height: 1px;
		margin: 30px 0 25px;
		background: #eee7df;
	}
	.carrier-grid {
		display: grid;
		grid-template-columns: repeat(4, 1fr);
		gap: 8px;
	}
	.carrier-option {
		display: flex;
		position: relative;
		flex-direction: column;
		align-items: flex-start;
		min-height: 94px;
		gap: 10px;
		border: 1px solid #e7ded3;
		border-radius: 12px;
		background: #fffefa;
		padding: 13px;
		text-align: left;
		cursor: pointer;
		transition: 0.2s;
	}
	.carrier-option:hover {
		border-color: #d8a28d;
		transform: translateY(-1px);
	}
	.carrier-option.selected {
		border-color: #d96d49;
		background: #fff4ee;
		box-shadow: inset 0 0 0 1px #d96d49;
	}
	.carrier-glyph {
		color: #d96d49;
		font-size: 21px;
	}
	.carrier-copy {
		display: grid;
		gap: 4px;
	}
	.carrier-copy strong {
		color: #51473e;
		font-size: 12px;
	}
	.carrier-copy small {
		color: #9e9489;
		font-size: 10px;
	}
	.carrier-selected {
		position: absolute;
		top: 11px;
		right: 11px;
		color: #d96d49;
		font-size: 12px;
	}
	.location-row {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 12px;
		margin: 24px 0;
		border: 1px dashed #dcd0c4;
		border-radius: 12px;
		padding: 13px 14px;
	}
	.location-copy {
		display: flex;
		align-items: center;
		gap: 10px;
	}
	.location-copy span:last-child {
		display: grid;
		gap: 3px;
	}
	.location-copy strong {
		color: #675d53;
		font-size: 12px;
		font-weight: 700;
	}
	.location-copy small {
		color: #a49a90;
		font-size: 10px;
	}
	.location-pin {
		color: #d96d49;
		font-size: 21px;
	}
	.location-btn {
		border: 0;
		background: transparent;
		color: #c26242;
		font: inherit;
		font-size: 11px;
		font-weight: 750;
		cursor: pointer;
	}
	.recipient-location-warning {
		display: flex;
		gap: 10px;
		margin: -10px 0 22px;
		border-radius: 10px;
		background: #fff5df;
		padding: 11px 12px;
		color: #9a6b31;
	}
	.recipient-location-warning > span:first-child {
		color: #c58535;
		font-size: 18px;
		line-height: 1;
	}
	.recipient-location-warning > span:last-child {
		display: grid;
		gap: 3px;
	}
	.recipient-location-warning strong {
		font-size: 11px;
	}
	.recipient-location-warning small {
		color: #b28a57;
		font-size: 10px;
		line-height: 1.4;
	}
	.privacy-section {
		margin: 24px 0;
		border-top: 1px solid #eee5dc;
		padding-top: 22px;
	}
	.privacy-grid {
		display: grid;
		grid-template-columns: 1fr 1fr;
		gap: 10px;
	}
	.privacy-option {
		display: flex;
		align-items: flex-start;
		gap: 10px;
		min-height: 92px;
		border: 1px solid #e2d8ce;
		border-radius: 12px;
		background: #fffefa;
		padding: 13px;
		text-align: left;
		cursor: pointer;
		transition: 0.2s;
	}
	.privacy-option:hover,
	.privacy-option.active {
		border-color: #d96d49;
		background: #fff4ee;
		box-shadow: inset 0 0 0 1px #d96d49;
	}
	.privacy-icon {
		color: #d96d49;
		font-size: 22px;
		line-height: 1;
	}
	.privacy-option > span:nth-child(2) {
		display: grid;
		gap: 5px;
		min-width: 0;
	}
	.privacy-option strong {
		color: #51473e;
		font-size: 12px;
	}
	.privacy-option small {
		color: #978b7f;
		font-size: 10px;
		line-height: 1.45;
	}
	.privacy-check {
		margin-left: auto;
		color: #d96d49;
		font-weight: 800;
	}
	.privacy-note {
		margin: 9px 0 0;
		color: #9e9185;
		font-size: 10px;
	}
	.error-box {
		margin: 14px 0;
		border-radius: 9px;
		background: #fff0ec;
		padding: 11px 13px;
		color: #b04435;
		font-size: 12px;
	}
	.send-button {
		display: flex;
		align-items: center;
		justify-content: space-between;
		width: 100%;
		min-height: 53px;
		border: 0;
		border-radius: 12px;
		background: #d96d49;
		padding: 0 18px 0 20px;
		color: white;
		font: inherit;
		font-size: 13px;
		font-weight: 800;
		cursor: pointer;
		box-shadow: 0 10px 22px rgba(195, 91, 55, 0.2);
		transition: 0.2s;
	}
	.send-button:hover:not(:disabled) {
		background: #c85e3b;
		transform: translateY(-1px);
	}
	.send-button:disabled {
		cursor: not-allowed;
		opacity: 0.5;
	}
	.send-arrow {
		font-size: 22px;
		font-weight: 400;
	}
	.result-card {
		display: flex;
		align-items: center;
		gap: 17px;
		max-width: 700px;
		margin: 16px auto 0;
		border: 1px solid #e7d5c8;
		border-radius: 18px;
		background: #fff7f1;
		padding: 20px 22px;
	}
	.result-icon {
		display: grid;
		width: 39px;
		height: 39px;
		flex: 0 0 39px;
		place-items: center;
		border-radius: 50%;
		background: #f3d6c8;
		color: #cb6644;
	}
	.result-copy {
		min-width: 0;
	}
	.result-copy h2 {
		font-size: 20px;
	}
	.result-copy p:last-child {
		margin: 7px 0 0;
		color: #8e7e70;
		font-size: 12px;
	}
	.result-copy strong {
		color: #c45e3e;
	}
	.result-link {
		color: #c45e3e;
		font-size: 12px;
		font-weight: 800;
		text-decoration: none;
	}
	.result-actions {
		display: flex;
		align-items: center;
		gap: 15px;
		margin-left: auto;
		flex: 0 0 auto;
	}
	.share-link {
		border: 1px solid #e2bca9;
		border-radius: 99px;
		background: #fffdf9;
		padding: 8px 11px;
		color: #c45e3e;
		font: inherit;
		font-size: 12px;
		font-weight: 800;
		cursor: pointer;
	}
	.share-link:hover {
		background: #fff0e7;
	}
	.share-note {
		color: #4c7a5c;
		font-size: 10px;
	}
	.result-link span {
		margin-left: 5px;
		font-size: 17px;
	}
	@media (max-width: 640px) {
		.compose-shell {
			padding: 45px 15px 70px;
		}
		.compose-hero {
			margin-bottom: 30px;
		}
		.compose-hero h1 {
			font-size: 47px;
		}
		.compose-hero p {
			font-size: 14px;
		}
		.compose-card {
			padding: 23px 17px;
			border-radius: 18px;
		}
		.card-heading {
			margin-bottom: 24px;
		}
		.card-heading h2 {
			font-size: 21px;
		}
		.carrier-grid {
			grid-template-columns: repeat(2, 1fr);
		}
		.carrier-option {
			min-height: 82px;
		}
		.location-row {
			align-items: flex-start;
			flex-direction: column;
		}
		.location-btn {
			padding: 0;
		}
		.result-card {
			align-items: flex-start;
			flex-wrap: wrap;
			margin-top: 12px;
			padding: 17px;
		}
		.result-actions {
			width: calc(100% - 56px);
			margin-left: 56px;
			justify-content: space-between;
		}
		.share-note {
			margin-left: 56px;
		}
	}
</style>
