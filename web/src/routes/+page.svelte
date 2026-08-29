<script lang="ts">
	import { composeMessage, ApiError, type ComposeResponse } from '$lib/api/messages';
	import { formatCountdown, formatDistance } from '$lib/delivery';

	const CARRIERS = [
		{ slug: 'pigeon', name: 'Merpati Pos', emoji: '🕊️', speed: 177 },
		{ slug: 'paper_plane', name: 'Pesawat Kertas', emoji: '✈️', speed: 250 },
		{ slug: 'falcon', name: 'Elang Perkasa', emoji: '🦅', speed: 320 },
		{ slug: 'drone', name: 'Mini Drone', emoji: '🚁', speed: 500 }
	];

	let recipientId = $state('');
	let body = $state('');
	let selectedCarrier = $state(CARRIERS[0]);

	let senderLat = $state(-6.2088);
	let senderLng = $state(106.8456);
	let recipientLat = $state(-7.2575);
	let recipientLng = $state(112.7521);

	let sending = $state(false);
	let error = $state('');
	let result = $state<ComposeResponse | null>(null);
	let countdown = $state('');

	function useMyLocation() {
		if (!navigator.geolocation) {
			error = 'Browser tidak mendukung geolokasi';
			return;
		}
		navigator.geolocation.getCurrentPosition(
			(pos) => {
				senderLat = pos.coords.latitude;
				senderLng = pos.coords.longitude;
			},
			() => {
				error = 'Gagal mengambil lokasi. Cek izin browser.';
			}
		);
	}

	async function handleSubmit() {
		error = '';
		result = null;
		if (!recipientId.trim() || !body.trim()) {
			error = 'Isi penerima dan pesan dulu.';
			return;
		}

		sending = true;
		try {
			result = await composeMessage({
				recipient_id: recipientId.trim(),
				body: body.trim(),
				carrier_slug: selectedCarrier.slug,
				sender_lat: senderLat,
				sender_lng: senderLng,
				recipient_lat: recipientLat,
				recipient_lng: recipientLng
			});
			startCountdown();
		} catch (e) {
			if (e instanceof ApiError) {
				error = `Gagal mengirim (${e.status}): ${e.message}`;
			} else {
				error = 'Gagal terhubung ke server. Pastikan backend berjalan.';
			}
		} finally {
			sending = false;
		}
	}

	function startCountdown() {
		if (!result) return;
		const tick = () => {
			if (!result) return;
			countdown = formatCountdown(result.arrives_at);
		};
		tick();
		const interval = setInterval(tick, 1000);
		return () => clearInterval(interval);
	}
</script>

<svelte:head>
	<title>Wingback — Kirim pesan lewat carrier</title>
</svelte:head>

<main class="mx-auto max-w-lg px-4 py-10">
	<header class="mb-8 text-center">
		<h1 class="text-3xl font-bold tracking-tight">🕊️ Wingback</h1>
		<p class="mt-2 text-sm text-gray-500">
			Pesanmu dibawa carrier sungguhan lewat jarak GPS asli. Semakin jauh, semakin lama — itulah
			nilainya.
		</p>
	</header>

	<form
		onsubmit={(e) => {
			e.preventDefault();
			handleSubmit();
		}}
		class="space-y-5"
	>
		<div>
			<label for="recipient" class="mb-1 block text-sm font-medium">ID Penerima</label>
			<input
				id="recipient"
				type="text"
				bind:value={recipientId}
				placeholder="username atau user-id"
				class="w-full rounded-lg border border-gray-300 px-3 py-2 focus:border-indigo-500 focus:outline-none"
			/>
		</div>

		<div>
			<label for="body" class="mb-1 block text-sm font-medium">Pesan</label>
			<textarea
				id="body"
				bind:value={body}
				rows="4"
				maxlength="2000"
				placeholder="Tulis pesanmu..."
				class="w-full rounded-lg border border-gray-300 px-3 py-2 focus:border-indigo-500 focus:outline-none"
			></textarea>
		</div>

		<div>
			<span class="mb-2 block text-sm font-medium">Pilih Carrier</span>
			<div class="grid grid-cols-2 gap-2 sm:grid-cols-4">
				{#each CARRIERS as carrier (carrier.slug)}
					<button
						type="button"
						onclick={() => (selectedCarrier = carrier)}
						class={`rounded-lg border p-3 text-center transition ${
							selectedCarrier.slug === carrier.slug
								? 'border-indigo-500 bg-indigo-50'
								: 'border-gray-200 hover:border-gray-300'
						}`}
					>
						<div class="text-2xl">{carrier.emoji}</div>
						<div class="mt-1 text-xs font-medium">{carrier.name}</div>
						<div class="text-[10px] text-gray-500">{carrier.speed} km/j</div>
					</button>
				{/each}
			</div>
		</div>

		<div class="rounded-lg border border-gray-200 p-3">
			<div class="flex items-center justify-between">
				<span class="text-sm font-medium">Lokasi kamu</span>
				<button
					type="button"
					onclick={useMyLocation}
					class="text-xs font-medium text-indigo-600 hover:underline"
				>
					Gunakan lokasi saat ini
				</button>
			</div>
			<p class="mt-1 text-xs text-gray-500">
				Lat {senderLat.toFixed(4)}, Lng {senderLng.toFixed(4)}
			</p>
		</div>

		{#if error}
			<p class="text-sm text-red-600">{error}</p>
		{/if}

		<button
			type="submit"
			disabled={sending}
			class="w-full rounded-lg bg-indigo-600 py-3 font-medium text-white transition hover:bg-indigo-700 disabled:opacity-50"
		>
			{sending ? 'Mengirim...' : `Kirim via ${selectedCarrier.name}`}
		</button>
	</form>

	{#if result}
		<div class="mt-8 rounded-lg border border-indigo-200 bg-indigo-50 p-4">
			<h2 class="font-semibold text-indigo-900">
				{selectedCarrier.emoji} Pesan sedang dalam perjalanan
			</h2>
			<dl class="mt-2 space-y-1 text-sm text-indigo-800">
				<div class="flex justify-between">
					<dt>Jarak</dt>
					<dd>{formatDistance(result.distance_km)}</dd>
				</div>
				<div class="flex justify-between">
					<dt>Kecepatan</dt>
					<dd>{result.speed_kmh} km/j</dd>
				</div>
				<div class="flex justify-between">
					<dt>Estimasi tiba</dt>
					<dd>{countdown}</dd>
				</div>
			</dl>
			{#if result.will_be_lost}
				<p class="mt-3 text-sm font-medium text-amber-700">
					⚠️ Ada kemungkinan carrier ini hilang di tengah jalan...
				</p>
			{/if}
		</div>
	{/if}
</main>
