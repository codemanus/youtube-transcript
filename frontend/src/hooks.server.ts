import { dev } from '$app/environment';
import type { Handle } from '@sveltejs/kit';

/** Go API (see README). Override for non-default dev setups: TRANSCRIPT_API_ORIGIN=http://127.0.0.1:9090 */
const defaultOrigin = 'http://127.0.0.1:8080';

export const handle: Handle = async ({ event, resolve }) => {
	if (!dev || !event.url.pathname.startsWith('/api')) {
		return resolve(event);
	}

	const origin = process.env.TRANSCRIPT_API_ORIGIN ?? defaultOrigin;
	const target = new URL(event.url.pathname + event.url.search, origin);

	const incoming = event.request;
	const headers = new Headers(incoming.headers);
	for (const h of ['host', 'connection', 'keep-alive']) {
		headers.delete(h);
	}

	const init: RequestInit = {
		method: incoming.method,
		headers,
		redirect: 'manual'
	};

	if (incoming.method !== 'GET' && incoming.method !== 'HEAD') {
		init.body = await incoming.arrayBuffer();
	}

	try {
		return await fetch(target, init);
	} catch {
		return new Response(
			JSON.stringify({
				error: `Dev proxy could not reach Go at ${origin}. Start the API (e.g. make run-backend).`
			}),
			{ status: 502, headers: { 'content-type': 'application/json' } }
		);
	}
};
