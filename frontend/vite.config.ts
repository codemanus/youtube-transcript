import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig } from 'vite';

export default defineConfig({
	plugins: [sveltekit()]
	// API proxy: SvelteKit handles /api in dev before Vite's server.proxy — see src/hooks.server.ts
});
