<script lang="ts">
	/* eslint-disable svelte/no-navigation-without-resolve */
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { messages, getToken, type Message } from '$lib/api/client';
	import { formatCountdown, formatDistance } from '$lib/delivery';

	let list = $state<Message[]>([]);
	let loading = $state(true);
	let error = $state('');

	onMount(async () => {
		if (!getToken()) {
			goto('/login');
			return;
		}
		try {
			list = await messages.listSent();
		} catch (e) {
			error = e instanceof Error ? e.message : 'Gagal memuat';
		} finally {
			loading = false;
		}
	});
</script>

<svelte:head>
	<title>Sent · Wingback</title>
</svelte:head>

<main class="mx-auto max-w-3xl px-4 py-10">
	<h1 class="mb-6 text-2xl font-bold">📤 Sent</h1>

	{#if loading}
		<p class="text-gray-500">Memuat...</p>
	{:else if error}
		<p class="text-red-600">{error}</p>
	{:else if list.length === 0}
		<p class="text-gray-500">Belum ada pesan terkirim.</p>
	{:else}
		<div class="space-y-3">
			{#each list as m (m.id)}
				<a
					href="/track/$m.id"
					class="block rounded-lg border border-gray-200 bg-white p-4 transition hover:border-indigo-400 hover:shadow"
				>
					<div class="flex items-start justify-between">
						<div>
							<p class="text-sm text-gray-500">Ke: {m.recipient_id.slice(0, 8)}...</p>
							<p class="mt-1 line-clamp-2 text-gray-800">{m.body}</p>
						</div>
						<div class="text-right text-xs text-gray-500">
							{#if m.status === 'in_transit'}
								<span class="rounded bg-amber-100 px-2 py-0.5 text-amber-800"
									>🕊️ {formatCountdown(m.arrives_at)}</span
								>
							{:else if m.status === 'delivered'}
								<span class="rounded bg-green-100 px-2 py-0.5 text-green-800">✓ Sampai</span>
							{:else}
								<span class="rounded bg-red-100 px-2 py-0.5 text-red-800">💀 Hilang</span>
							{/if}
							<p class="mt-1">{formatDistance(m.distance_km)}</p>
						</div>
					</div>
				</a>
			{/each}
		</div>
	{/if}
</main>
