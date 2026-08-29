<script lang="ts">
	/* eslint-disable svelte/no-navigation-without-resolve */
	import { goto } from '$app/navigation';
	import { onMount } from 'svelte';
	import { auth, setSession, getStoredUser } from '$lib/api/client';

	let username = $state('');
	let password = $state('');
	let error = $state('');
	let submitting = $state(false);

	onMount(() => {
		if (getStoredUser()) goto('/');
	});

	async function submit(e: Event) {
		e.preventDefault();
		error = '';
		submitting = true;
		try {
			setSession(await auth.login(username.trim(), password));
			goto('/');
		} catch (e) {
			error = e instanceof Error ? e.message : 'Tidak bisa masuk. Coba lagi.';
		} finally {
			submitting = false;
		}
	}
</script>

<svelte:head><title>Masuk · Wingback</title></svelte:head>

<main class="auth-page">
	<section class="auth-story">
		<div class="auth-stamp">WB / 01</div>
		<p class="auth-eyebrow">PESAN YANG PUNYA PERJALANAN</p>
		<h1>Hal baik<br /><em>layak ditunggu.</em></h1>
		<p class="auth-lead">
			Wingback membuat jarak terasa lagi. Tulis pesan, pilih pengantar, lalu biarkan ia menempuh
			jalannya.
		</p>
		<div class="auth-quote">
			“Tidak semua hal harus tiba sekarang.”<span>— catatan di sayap</span>
		</div>
	</section>
	<section class="auth-card">
		<div class="card-kicker">SELAMAT DATANG KEMBALI</div>
		<h2>Masuk ke Wingback</h2>
		<p class="auth-subtitle">Lanjutkan surat-surat yang sedang menuju seseorang.</p>
		<form onsubmit={submit}>
			<div class="auth-field">
				<label for="username">Username</label><input
					id="username"
					type="text"
					bind:value={username}
					autocomplete="username"
					placeholder="@username"
					required
				/>
			</div>
			<div class="auth-field">
				<label for="password">Password</label><input
					id="password"
					type="password"
					bind:value={password}
					autocomplete="current-password"
					placeholder="Minimal 8 karakter"
					minlength="8"
					required
				/>
			</div>
			{#if error}<p class="auth-error" role="alert">{error}</p>{/if}
			<button class="auth-submit" type="submit" disabled={submitting}
				>{submitting ? 'Membuka surat...' : 'Masuk'} <span>→</span></button
			>
		</form>
		<p class="auth-switch">Belum punya akun? <a href="/register">Mulai menulis</a></p>
	</section>
</main>

<style>
	.auth-page {
		display: grid;
		grid-template-columns: minmax(0, 0.95fr) minmax(360px, 1fr);
		gap: clamp(45px, 8vw, 130px);
		align-items: center;
		max-width: 1050px;
		min-height: calc(100dvh - 150px);
		margin: auto;
		padding: 70px 28px;
	}
	.auth-story {
		position: relative;
		padding: 20px 0;
	}
	.auth-stamp {
		display: inline-grid;
		width: 54px;
		height: 54px;
		margin-bottom: 45px;
		place-items: center;
		border: 1px solid #d7c9bc;
		border-radius: 50%;
		color: #bc6243;
		font-size: 9px;
		font-weight: 800;
		letter-spacing: 0.08em;
		transform: rotate(-12deg);
	}
	.auth-eyebrow,
	.card-kicker {
		color: #bc6243;
		font-size: 10px;
		font-weight: 800;
		letter-spacing: 0.17em;
	}
	.auth-story h1 {
		margin: 14px 0 24px;
		color: #302820;
		font-family: Georgia, serif;
		font-size: clamp(48px, 6vw, 72px);
		font-weight: 500;
		letter-spacing: -0.06em;
		line-height: 0.94;
	}
	.auth-story h1 em {
		color: #d76e4b;
		font-style: italic;
	}
	.auth-lead {
		max-width: 390px;
		color: #84786b;
		font-size: 15px;
		line-height: 1.75;
	}
	.auth-quote {
		max-width: 330px;
		margin-top: 55px;
		border-left: 2px solid #e5b9a7;
		padding-left: 16px;
		color: #a3978a;
		font-family: Georgia, serif;
		font-size: 14px;
		font-style: italic;
		line-height: 1.5;
	}
	.auth-quote span {
		display: block;
		margin-top: 7px;
		color: #b7aaa0;
		font-family: inherit;
		font-size: 10px;
		font-style: normal;
	}
	.auth-card {
		border: 1px solid #e6ddd3;
		border-radius: 24px;
		background: rgba(255, 253, 249, 0.9);
		padding: clamp(26px, 5vw, 43px);
		box-shadow: 0 25px 75px rgba(79, 57, 37, 0.09);
	}
	.auth-card h2 {
		margin: 10px 0 7px;
		color: #302820;
		font-family: Georgia, serif;
		font-size: 29px;
		font-weight: 500;
		letter-spacing: -0.04em;
	}
	.auth-subtitle {
		margin: 0 0 31px;
		color: #968b80;
		font-size: 13px;
		line-height: 1.6;
	}
	.auth-field {
		margin-bottom: 20px;
	}
	.auth-field label {
		display: block;
		margin-bottom: 8px;
		color: #63594f;
		font-size: 12px;
		font-weight: 750;
	}
	.auth-field input {
		box-sizing: border-box;
		width: 100%;
		border: 1px solid #ddd2c6;
		border-radius: 10px;
		outline: 0;
		background: #fffefa;
		padding: 14px;
		color: #302820;
		font: inherit;
		font-size: 14px;
		transition: 0.2s;
	}
	.auth-field input:focus {
		border-color: #d96d49;
		box-shadow: 0 0 0 3px #f8e4da;
	}
	.auth-field input::placeholder {
		color: #b7aca1;
	}
	.auth-error {
		border-radius: 9px;
		background: #fff0ec;
		padding: 10px 12px;
		color: #af4435;
		font-size: 12px;
	}
	.auth-submit {
		display: flex;
		align-items: center;
		justify-content: space-between;
		width: 100%;
		min-height: 52px;
		margin-top: 25px;
		border: 0;
		border-radius: 10px;
		background: #d96d49;
		padding: 0 16px 0 18px;
		color: white;
		font: inherit;
		font-size: 13px;
		font-weight: 800;
		cursor: pointer;
		transition: 0.2s;
	}
	.auth-submit:hover:not(:disabled) {
		background: #c65d3d;
		transform: translateY(-1px);
	}
	.auth-submit:disabled {
		opacity: 0.55;
		cursor: wait;
	}
	.auth-submit span {
		font-size: 21px;
		font-weight: 400;
	}
	.auth-switch {
		margin: 24px 0 0;
		color: #9b9085;
		text-align: center;
		font-size: 12px;
	}
	.auth-switch a {
		color: #bd6040;
		font-weight: 800;
		text-decoration: none;
	}
	@media (max-width: 720px) {
		.auth-page {
			display: block;
			min-height: auto;
			padding: 46px 16px 70px;
		}
		.auth-story {
			margin-bottom: 38px;
			text-align: center;
		}
		.auth-stamp {
			margin-bottom: 28px;
		}
		.auth-story h1 {
			font-size: 50px;
		}
		.auth-lead {
			margin: auto;
			font-size: 14px;
		}
		.auth-quote {
			margin: 28px auto 0;
			text-align: left;
		}
		.auth-card {
			padding: 25px 19px;
		}
	}
</style>
