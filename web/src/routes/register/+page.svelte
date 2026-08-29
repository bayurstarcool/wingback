<script lang="ts">
	/* eslint-disable svelte/no-navigation-without-resolve */
	import { goto } from '$app/navigation';
	import { onMount } from 'svelte';
	import { auth, setSession, getStoredUser } from '$lib/api/client';

	let email = $state('');
	let password = $state('');
	let displayName = $state('');
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
			const result = await auth.register(email, password, displayName);
			setSession(result);
			goto('/');
		} catch (e) {
			error = e instanceof Error ? e.message : 'Gagal';
		} finally {
			submitting = false;
		}
	}
</script>

<svelte:head>
	<title>Register · Wingback</title>
</svelte:head>

<main class="mx-auto mt-20 max-w-sm px-4">
	<h1 class="mb-6 text-center text-2xl font-bold">Daftar 🕊️</h1>
	<form onsubmit={submit} class="space-y-4">
		<div>
			<label for="name" class="mb-1 block text-sm font-medium">Nama tampil</label>
			<input
				id="name"
				type="text"
				bind:value={displayName}
				required
				class="w-full rounded-lg border border-gray-300 px-3 py-2"
			/>
		</div>
		<div>
			<label for="email" class="mb-1 block text-sm font-medium">Email</label>
			<input
				id="email"
				type="email"
				bind:value={email}
				required
				class="w-full rounded-lg border border-gray-300 px-3 py-2"
			/>
		</div>
		<div>
			<label for="password" class="mb-1 block text-sm font-medium">Password (min 8)</label>
			<input
				id="password"
				type="password"
				bind:value={password}
				required
				minlength="8"
				class="w-full rounded-lg border border-gray-300 px-3 py-2"
			/>
		</div>
		{#if error}
			<p class="text-sm text-red-600">{error}</p>
		{/if}
		<button
			type="submit"
			disabled={submitting}
			class="w-full rounded-lg bg-indigo-600 py-3 font-medium text-white hover:bg-indigo-700 disabled:opacity-50"
		>
			{submitting ? 'Mendaftar...' : 'Daftar'}
		</button>
	</form>
	<p class="mt-4 text-center text-sm text-gray-500">
		Sudah punya akun? <a href="/login" class="text-indigo-600 hover:underline">Masuk</a>
	</p>
</main>
