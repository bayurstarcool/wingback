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
			list = await messages.listInbox();
		} catch (e) {
			error = e instanceof Error ? e.message : 'Kotak masuk belum bisa dimuat.';
		} finally {
			loading = false;
		}
	});

	function statusLabel(status: Message['status']) {
		if (status === 'in_transit') return 'Sedang terbang';
		if (status === 'delivered') return 'Sudah tiba';
		return 'Hilang di jalan';
	}
</script>

<svelte:head><title>Kotak masuk · Wingback</title></svelte:head>

<main class="list-page">
	<header class="list-header">
		<div>
			<p class="eyebrow">SURAT UNTUKMU</p>
			<h1>Kotak masuk</h1>
			<p class="intro">Pesan yang sedang menempuh jarak untuk sampai kepadamu.</p>
		</div>
		<a class="write-link" href="/">Tulis pesan <span>↗</span></a>
	</header>
	<div class="list-rule"><span>{list.length} surat</span><span>TERBARU DI ATAS</span></div>

	{#if loading}<div class="empty-state">
			<span class="empty-glyph">◌</span>
			<p>Membuka kotak surat...</p>
		</div>
	{:else if error}<div class="message-error" role="alert">{error}</div>
	{:else if list.length === 0}<div class="empty-state">
			<span class="empty-glyph">✉</span>
			<h2>Masih sunyi di sini.</h2>
			<p>Surat pertama yang datang akan muncul di sini.</p>
		</div>
	{:else}<div class="message-list">
			{#each list as m (m.id)}
				<a class="message-card" href={`/track/${m.id}`}>
					<div class="message-avatar">
						{m.status === 'in_transit' ? '↗' : m.status === 'delivered' ? '✓' : '×'}
					</div>
					<div class="message-main">
						<div class="message-meta">
							<span>DARI PENGIRIM</span><time
								>{new Date(m.departs_at).toLocaleDateString('id-ID', {
									day: 'numeric',
									month: 'short'
								})}</time
							>
						</div>
						<h2>Surat yang menempuh {formatDistance(m.distance_km)}</h2>
						<p>{m.body}</p>
					</div>
					<div class="message-side">
						<span
							class:flight={m.status === 'in_transit'}
							class:arrived={m.status === 'delivered'}
							class:lost={m.status === 'lost'}
							class="status-pill"
							>{m.status === 'in_transit' ? '↗' : m.status === 'delivered' ? '✓' : '×'}
							{statusLabel(m.status)}</span
						>{#if m.status === 'in_transit'}<small>Tiba {formatCountdown(m.arrives_at)}</small
							>{:else}<small>{formatDistance(m.distance_km)}</small>{/if}
					</div>
					<span class="card-arrow">→</span>
				</a>
			{/each}
		</div>{/if}
</main>

<style>
	.list-page {
		max-width: 900px;
		margin: auto;
		padding: 74px 28px 90px;
	}
	.list-header {
		display: flex;
		align-items: flex-end;
		justify-content: space-between;
		gap: 20px;
		margin-bottom: 47px;
	}
	.eyebrow {
		margin: 0 0 14px;
		color: #bc6243;
		font-size: 10px;
		font-weight: 800;
		letter-spacing: 0.18em;
	}
	.list-header h1 {
		margin: 0;
		color: #302820;
		font-family: Georgia, serif;
		font-size: clamp(39px, 6vw, 59px);
		font-weight: 500;
		letter-spacing: -0.06em;
		line-height: 1;
	}
	.intro {
		max-width: 350px;
		margin: 15px 0 0;
		color: #93877b;
		font-size: 14px;
		line-height: 1.6;
	}
	.write-link {
		border-bottom: 1px solid #d96d49;
		padding-bottom: 5px;
		color: #c16040;
		font-size: 12px;
		font-weight: 800;
		text-decoration: none;
		white-space: nowrap;
	}
	.write-link span {
		margin-left: 5px;
		font-size: 16px;
	}
	.list-rule {
		display: flex;
		justify-content: space-between;
		border-top: 1px solid #ded4ca;
		padding: 12px 3px;
		color: #9d9186;
		font-size: 10px;
		font-weight: 800;
		letter-spacing: 0.12em;
	}
	.message-list {
		display: grid;
		gap: 9px;
	}
	.message-card {
		display: grid;
		position: relative;
		grid-template-columns: 42px minmax(0, 1fr) auto 16px;
		align-items: center;
		gap: 15px;
		border: 1px solid #e9e0d7;
		border-radius: 15px;
		background: rgba(255, 253, 249, 0.76);
		padding: 19px 18px;
		text-decoration: none;
		transition: 0.2s;
	}
	.message-card:hover {
		border-color: #d9ad9a;
		background: #fffaf5;
		box-shadow: 0 12px 30px rgba(77, 55, 36, 0.07);
		transform: translateY(-1px);
	}
	.message-avatar {
		display: grid;
		width: 40px;
		height: 40px;
		place-items: center;
		border-radius: 50%;
		background: #f4dfd4;
		color: #c76745;
		font-family: Georgia, serif;
		font-size: 21px;
	}
	.message-meta {
		display: flex;
		justify-content: space-between;
		color: #b0a499;
		font-size: 9px;
		font-weight: 800;
		letter-spacing: 0.12em;
	}
	.message-meta time {
		letter-spacing: 0;
		font-weight: 600;
	}
	.message-main h2 {
		overflow: hidden;
		margin: 6px 0 4px;
		color: #443b33;
		font-family: Georgia, serif;
		font-size: 16px;
		font-weight: 500;
		text-overflow: ellipsis;
		white-space: nowrap;
	}
	.message-main p {
		overflow: hidden;
		margin: 0;
		color: #988d82;
		font-size: 12px;
		text-overflow: ellipsis;
		white-space: nowrap;
	}
	.message-side {
		display: grid;
		justify-items: end;
		gap: 7px;
		min-width: 105px;
	}
	.status-pill {
		border-radius: 99px;
		background: #f0ebe5;
		padding: 6px 9px;
		color: #7d7369;
		font-size: 10px;
		font-weight: 800;
		white-space: nowrap;
	}
	.status-pill.flight {
		background: #fff0e7;
		color: #c36040;
	}
	.status-pill.arrived {
		background: #e8f0e9;
		color: #4c7a5c;
	}
	.status-pill.lost {
		background: #f9e8e5;
		color: #b25b4c;
	}
	.message-side small {
		color: #aa9e92;
		font-size: 10px;
	}
	.card-arrow {
		color: #c6b8aa;
		font-size: 18px;
		transition: 0.2s;
	}
	.message-card:hover .card-arrow {
		color: #c36040;
		transform: translateX(3px);
	}
	.empty-state {
		padding: 80px 20px;
		text-align: center;
	}
	.empty-glyph {
		display: block;
		margin-bottom: 18px;
		color: #d6a28e;
		font-family: Georgia, serif;
		font-size: 42px;
	}
	.empty-state h2 {
		margin: 0 0 9px;
		color: #554a41;
		font-family: Georgia, serif;
		font-size: 24px;
		font-weight: 500;
	}
	.empty-state p {
		margin: 0;
		color: #a89b8f;
		font-size: 13px;
	}
	.message-error {
		border-radius: 10px;
		background: #fff0ec;
		padding: 14px;
		color: #af4435;
		font-size: 13px;
	}
	@media (max-width: 620px) {
		.list-page {
			padding: 47px 16px 70px;
		}
		.list-header {
			align-items: flex-start;
			flex-direction: column;
			margin-bottom: 33px;
		}
		.write-link {
			margin-top: 5px;
		}
		.message-card {
			grid-template-columns: 35px minmax(0, 1fr) 15px;
			gap: 11px;
			padding: 14px 12px;
		}
		.message-avatar {
			width: 34px;
			height: 34px;
			font-size: 17px;
		}
		.message-side {
			grid-column: 2 / 3;
			display: flex;
			align-items: center;
			justify-content: space-between;
			width: 100%;
			min-width: 0;
			margin-top: 4px;
		}
		.message-main h2 {
			font-size: 15px;
		}
		.message-main p {
			font-size: 11px;
		}
		.card-arrow {
			grid-column: 3;
			grid-row: 1;
		}
	}
</style>
