<script lang="ts">
	import favicon from '$lib/assets/favicon.svg';
	import { page } from '$app/state';

	let { children } = $props();

	const links = [
		{ href: '/', label: 'Transcript' },
		{ href: '/church-guide', label: 'Church Guide' }
	];
</script>

<svelte:head>
	<link rel="icon" href={favicon} />
</svelte:head>

<nav class="nav">
	<div class="nav-inner">
		{#each links as link (link.href)}
			<a href={link.href} class="nav-link" aria-current={page.url.pathname === link.href ? 'page' : undefined}>
				{link.label}
			</a>
		{/each}
	</div>
</nav>

{@render children()}

<style>
	.nav {
		border-bottom: 1px solid #e5e7eb;
	}

	.nav-inner {
		max-width: 42rem;
		margin: 0 auto;
		padding: 0.75rem 1.25rem;
		display: flex;
		gap: 1.25rem;
		font-family:
			system-ui,
			-apple-system,
			'Segoe UI',
			sans-serif;
	}

	.nav-link {
		font-size: 0.9rem;
		font-weight: 600;
		color: #6b7280;
		text-decoration: none;
	}

	.nav-link:hover {
		color: #111827;
	}

	.nav-link[aria-current='page'] {
		color: #111827;
	}
</style>
