<script lang="ts">
	/* eslint-disable svelte/no-navigation-without-resolve */
	import { goto } from '$app/navigation';
	import { onMount } from 'svelte';
	import { auth, getStoredUser, getToken } from '$lib/api/client';

	let { children } = $props();

	let user = $state(getStoredUser());
	let checking = $state(true);

	onMount(() => {
		const token = getToken();
		if (token) {
			auth
				.me()
				.then((u) => {
					user = u;
				})
				.catch(() => {
					user = null;
				})
				.finally(() => {
					checking = false;
				});
		} else {
			checking = false;
		}
	});

	function logout() {
		import('$lib/api/client').then((m) => m.clearSession());
		user = null;
		goto('/login');
	}
</script>

<div class="min-h-screen bg-gray-50 text-gray-900">
	{#if !checking}
		<header class="border-b border-gray-200 bg-white">
			<div class="mx-auto flex max-w-5xl items-center justify-between px-4 py-3">
				<a href="/" class="text-lg font-bold tracking-tight">🕊️ Wingback</a>
				<nav class="flex items-center gap-4 text-sm">
					{#if user}
						<a href="/" class="hover:text-indigo-600">Compose</a>
						<a href="/inbox" class="hover:text-indigo-600">Inbox</a>
						<a href="/sent" class="hover:text-indigo-600">Sent</a>
						<span class="text-gray-500">|</span>
						<span class="text-gray-600">{user.display_name}</span>
						<button onclick={logout} class="text-gray-500 hover:text-red-600">Logout</button>
					{:else}
						<a href="/login" class="hover:text-indigo-600">Login</a>
						<a href="/register" class="hover:text-indigo-600">Register</a>
					{/if}
				</nav>
			</div>
		</header>
	{/if}
	{@render children()}
</div>
