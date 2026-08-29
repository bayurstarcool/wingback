<script lang="ts">
	/* eslint-disable svelte/no-navigation-without-resolve */
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { carriers, messages, getToken, type Carrier } from '$lib/api/client';
	import { formatCountdown } from '$lib/delivery';

	let recipientId = $state('');
	let body = $state('');
	let carrierSlug = $state('pigeon');
	let senderLat = $state(-6.2088);
	let senderLng = $state(106.8456);
	let recipientLat = $state(-7.2575);
	let recipientLng = $state(112.7521);

	let carrierList = $state<Carrier[]>([]);
	let sending = $state(false);
	let error = $state('');
	let lastResult = $state<{
		message_id: string;
		arrives_at: string;
		will_be_lost: boolean;
		carrier: string;
	} | null>(null);
	let countdown = $state('');

	onMount(() => {
		if (!getToken()) {
			goto('/login');
			return;
		}
		carriers
			.list()
			.then((cs) => (carrierList = cs))
			.catch(() => {});

		// Try to get my location and update server.
		if (navigator.geolocation) {
			navigator.geolocation.getCurrentPosition(
				(pos) => {
					senderLat = pos.coords.latitude;
					senderLng = pos.coords.longitude;
					messages.updateLocation(senderLat, senderLng).catch(() => {});
				},
				() => {},
				{ timeout: 5000 }
			);
		}
	});

	function useMyLocation() {
		if (!navigator.geolocation) {
			error = 'Browser tidak mendukung geolokasi';
			return;
		}
		navigator.geolocation.getCurrentPosition(
			(pos) => {
				senderLat = pos.coords.latitude;
				senderLng = pos.coords.longitude;
				messages.updateLocation(senderLat, senderLng).catch(() => {});
			},
			() => {
				error = 'Gagal mengambil lokasi.';
			}
		);
	}

	async function handleSubmit(e: Event) {
		e.preventDefault();
		error = '';
		lastResult = null;
		if (!recipientId.trim() || !body.trim()) {
			error = 'Isi penerima dan pesan dulu.';
			return;
		}
		sending = true;
		try {
			const result = await messages.compose({
				recipient_id: recipientId.trim(),
				body: body.trim(),
				carrier_slug: carrierSlug,
				sender_lat: senderLat,
				sender_lng: senderLng,
				recipient_lat: recipientLat,
				recipient_lng: recipientLng
			});
			lastResult = {
				message_id: result.message_id,
				arrives_at: result.arrives_at,
				will_be_lost: result.will_be_lost,
				carrier: result.carrier
			};
			startCountdown();
		} catch (e) {
			error = e instanceof Error ? e.message : 'Gagal terhubung ke server.';
		} finally {
			sending = false;
		}
	}

	function startCountdown() {
		if (!lastResult) return;
		const tick = () => {
			if (!lastResult) return;
			countdown = formatCountdown(lastResult.arrives_at);
		};
		tick();
		const interval = setInterval(tick, 1000);
		return () => clearInterval(interval);
	}
</script>

<svelte:head>
	<title>Compose · Wingback</title>
</svelte:head>

<main class="mx-auto max-w-lg px-4 py-10">
	<header class="mb-8 text-center">
		<h1 class="text-3xl font-bold tracking-tight">🕊️ Wingback</h1>
		<p class="mt-2 text-sm text-gray-500">
			Pesanmu dibawa carrier sungguhan lewat jarak GPS asli. Semakin jauh, semakin lama — itulah
			nilainya.
		</p>
	</header>

	<form onsubmit={handleSubmit} class="space-y-5">
		<div>
			<label for="recipient" class="mb-1 block text-sm font-medium">ID Penerima (user_id)</label>
			<input
				id="recipient"
				type="text"
				bind:value={recipientId}
				placeholder="uuid user tujuan"
				required
				class="w-full rounded-lg border border-gray-300 px-3 py-2"
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
				required
				class="w-full rounded-lg border border-gray-300 px-3 py-2"></textarea>
		</div>

		<div>
			<span class="mb-2 block text-sm font-medium">Pilih Carrier</span>
			<div class="grid grid-cols-2 gap-2 sm:grid-cols-4">
				{#each carrierList as carrier (carrier.slug)}
					<button
						type="button"
						onclick={() => (carrierSlug = carrier.slug)}
						class={`rounded-lg border p-3 text-center transition ${
							carrierSlug === carrier.slug
								? 'border-indigo-500 bg-indigo-50'
								: 'border-gray-200 hover:border-gray-300'
						}`}
					>
						<div class="text-xs font-medium">{carrier.name}</div>
						<div class="text-[10px] text-gray-500">{carrier.speed_kmh} km/j</div>
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

		<div class="rounded-lg border border-gray-200 p-3">
			<span class="mb-2 block text-sm font-medium">Lokasi penerima</span>
			<div class="grid grid-cols-2 gap-2 text-sm">
				<input
					type="number"
					step="0.0001"
					bind:value={recipientLat}
					class="rounded border border-gray-300 px-2 py-1"
					placeholder="lat"
				/>
				<input
					type="number"
					step="0.0001"
					bind:value={recipientLng}
					class="rounded border border-gray-300 px-2 py-1"
					placeholder="lng"
				/>
			</div>
		</div>

		{#if error}
			<p class="text-sm text-red-600">{error}</p>
		{/if}

		<button
			type="submit"
			disabled={sending}
			class="w-full rounded-lg bg-indigo-600 py-3 font-medium text-white transition hover:bg-indigo-700 disabled:opacity-50"
		>
			{sending ? 'Mengirim...' : 'Kirim'}
		</button>
	</form>

	{#if lastResult}
		<div class="mt-8 rounded-lg border border-indigo-200 bg-indigo-50 p-4">
			<h2 class="font-semibold text-indigo-900">Pesan dalam perjalanan</h2>
			<p class="mt-1 text-sm text-indigo-800">Tiba dalam: <strong>{countdown}</strong></p>
			{#if lastResult.will_be_lost}
				<p class="mt-2 text-sm font-medium text-amber-700">
					⚠️ Ada kemungkinan carrier ini hilang di tengah jalan...
				</p>
			{/if}
			<a
				href="/track/$lastResult.message_id"
				class="mt-3 inline-block rounded bg-indigo-600 px-3 py-1 text-sm text-white hover:bg-indigo-700"
			>
				Lihat live tracking →
			</a>
		</div>
	{/if}
</main>
