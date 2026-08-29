<script lang="ts">
	/* eslint-disable svelte/no-navigation-without-resolve */
	import { goto } from '$app/navigation';
	import { onMount } from 'svelte';
	import './layout.css';
	import { auth, clearSession, getStoredUser, getToken, type User } from '$lib/api/client';

	let { children } = $props();
	let user = $state<User | null>(getStoredUser());
	let checking = $state(true);
	let menuOpen = $state(false);

	onMount(() => {
		const token = getToken();
		if (!token) {
			checking = false;
			return;
		}
		auth
			.me()
			.then((u) => (user = u))
			.catch(() => {
				clearSession();
				user = null;
			})
			.finally(() => (checking = false));
	});

	function logout() {
		clearSession();
		user = null;
		goto('/login');
	}
</script>

<svelte:head>
	<meta name="theme-color" content="#f7f2eb" />
	<meta name="description" content="Wingback — kirim pesan yang punya perjalanan." />
	<meta property="og:site_name" content="Wingback" />
	<meta property="og:title" content="Wingback · Pesan yang punya perjalanan" />
	<meta
		property="og:description"
		content="Kirim pesan yang sengaja menunggu. Lihat perjalanannya, bukan cuma hasilnya."
	/>
	<meta property="og:type" content="website" />
	<meta name="twitter:card" content="summary" />
	<meta name="twitter:title" content="Wingback · Pesan yang punya perjalanan" />
	<meta name="twitter:description" content="Kirim pesan yang sengaja menunggu." />
</svelte:head>

<div class="app-shell">
	{#if !checking}
		<header class="site-header">
			<div class="nav-inner">
				<a href="/" class="brand" aria-label="Wingback home"
					><span class="brand-mark">↗</span><span>wingback</span></a
				>
				{#if user}
					<button
						class="mobile-menu"
						aria-label="Buka menu"
						aria-expanded={menuOpen}
						onclick={() => (menuOpen = !menuOpen)}>☰</button
					>
					<nav class:open={menuOpen} class="main-nav">
						<a href="/" onclick={() => (menuOpen = false)}>Tulis pesan</a>
						<a href="/inbox" onclick={() => (menuOpen = false)}>Kotak masuk</a>
						<a href="/sent" onclick={() => (menuOpen = false)}>Terkirim</a>
						<span class="nav-rule"></span>
						<span class="user-chip"
							><span class="user-avatar">{user.display_name.slice(0, 1).toUpperCase()}</span
							>@{user.username}</span
						>
						<button class="logout-btn" onclick={logout}>Keluar</button>
					</nav>
				{:else}
					<nav class="main-nav">
						<a href="/login">Masuk</a><a class="nav-cta" href="/register">Mulai menulis</a>
					</nav>
				{/if}
			</div>
		</header>
	{/if}
	{@render children()}
	<footer class="site-footer">
		<span>wingback</span><span>Pesan yang punya perjalanan.</span>
	</footer>
</div>

<style>
	.app-shell {
		min-height: 100dvh;
		display: flex;
		flex-direction: column;
	}
	.site-header {
		position: sticky;
		top: 0;
		z-index: 20;
		border-bottom: 1px solid rgba(68, 53, 39, 0.1);
		background: rgba(250, 247, 242, 0.86);
		backdrop-filter: blur(16px);
	}
	.nav-inner {
		display: flex;
		align-items: center;
		justify-content: space-between;
		max-width: 1120px;
		min-height: 70px;
		margin: auto;
		padding: 0 28px;
	}
	.brand {
		display: inline-flex;
		align-items: center;
		gap: 9px;
		color: #302820;
		font-family: Georgia, serif;
		font-size: 20px;
		font-weight: 600;
		letter-spacing: -0.04em;
		text-decoration: none;
	}
	.brand-mark {
		display: grid;
		width: 29px;
		height: 29px;
		place-items: center;
		border-radius: 9px;
		background: #d96d49;
		color: white;
		font-family: Arial, sans-serif;
		font-size: 20px;
		font-weight: 400;
		transform: rotate(-12deg);
	}
	.main-nav {
		display: flex;
		align-items: center;
		gap: 25px;
		color: #83776a;
		font-size: 12px;
		font-weight: 700;
	}
	.main-nav a,
	.logout-btn {
		color: inherit;
		text-decoration: none;
		transition: color 0.2s;
	}
	.main-nav a:hover,
	.logout-btn:hover {
		color: #c45e3e;
	}
	.nav-cta {
		border-radius: 8px;
		background: #302820;
		padding: 11px 15px;
		color: white !important;
	}
	.nav-rule {
		width: 1px;
		height: 23px;
		background: #ded3c7;
	}
	.user-chip {
		display: inline-flex;
		align-items: center;
		gap: 7px;
		color: #5f554c;
	}
	.user-avatar {
		display: grid;
		width: 25px;
		height: 25px;
		place-items: center;
		border-radius: 50%;
		background: #f1d6c9;
		color: #bf6040;
		font-family: Georgia, serif;
		font-size: 12px;
	}
	.logout-btn {
		border: 0;
		background: transparent;
		font: inherit;
		cursor: pointer;
	}
	.mobile-menu {
		display: none;
		border: 0;
		background: transparent;
		color: #5f554c;
		font-size: 20px;
	}
	.site-footer {
		display: flex;
		justify-content: space-between;
		max-width: 1120px;
		width: calc(100% - 56px);
		margin: auto auto 0;
		border-top: 1px solid rgba(68, 53, 39, 0.1);
		padding: 18px 0 24px;
		color: #a09589;
		font-size: 11px;
	}
	@media (max-width: 680px) {
		.nav-inner {
			min-height: 62px;
			padding: 0 16px;
		}
		.mobile-menu {
			display: block;
		}
		.main-nav {
			display: none;
			position: absolute;
			top: 62px;
			right: 16px;
			left: 16px;
			flex-direction: column;
			align-items: stretch;
			gap: 0;
			border: 1px solid #e7ded3;
			border-radius: 13px;
			background: #fffdf9;
			padding: 7px;
			box-shadow: 0 15px 35px rgba(60, 40, 24, 0.12);
		}
		.main-nav.open {
			display: flex;
		}
		.main-nav a,
		.logout-btn {
			padding: 13px 12px;
		}
		.nav-rule {
			width: auto;
			height: 1px;
			margin: 5px 12px;
		}
		.user-chip {
			padding: 10px 12px;
		}
		.site-footer {
			width: calc(100% - 32px);
		}
	}
	:global(html) {
		background: #f7f2eb;
	}
	:global(body) {
		margin: 0;
		background: #f7f2eb;
		color: #302820;
		font-family:
			Inter,
			ui-sans-serif,
			system-ui,
			-apple-system,
			BlinkMacSystemFont,
			'Segoe UI',
			sans-serif;
	}
	:global(button),
	:global(input),
	:global(textarea) {
		font-family: inherit;
	}
	:global(*:focus-visible) {
		outline: 3px solid rgba(217, 109, 73, 0.38);
		outline-offset: 2px;
	}
	:global(.min-h-screen.bg-gray-50) {
		background: transparent;
	}
</style>
