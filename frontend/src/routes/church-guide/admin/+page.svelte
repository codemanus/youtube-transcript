<script lang="ts">
	type StatusResponse = { configured: boolean; updatedAt?: string };
	type ErrorBody = { error?: string };

	let statusLoading = $state(true);
	let statusErr = $state('');
	let status = $state<StatusResponse | null>(null);

	let cookieInput = $state('');
	let saving = $state(false);
	let saveErr = $state('');
	let saveOk = $state(false);

	function formattedUpdatedAt(iso: string): string {
		const d = new Date(iso);
		if (Number.isNaN(d.getTime())) return iso;
		return d.toLocaleString(undefined, {
			year: 'numeric',
			month: 'long',
			day: 'numeric',
			hour: 'numeric',
			minute: '2-digit'
		});
	}

	function authMessage(httpStatus: number): string {
		return httpStatus === 401
			? 'Authentication required or incorrect. Reload the page to be prompted again.'
			: '';
	}

	async function fetchStatus() {
		statusLoading = true;
		statusErr = '';
		try {
			const res = await fetch('/api/church-guide/admin/status');
			const raw = await res.text();
			let data: StatusResponse & ErrorBody;
			try {
				data = JSON.parse(raw) as StatusResponse & ErrorBody;
			} catch {
				statusErr =
					res.status === 502
						? 'Could not reach the API (is the Go server running on port 8080?).'
						: `Invalid response (${res.status}).`;
				return;
			}
			if (!res.ok) {
				statusErr = authMessage(res.status) || data.error || `Request failed (${res.status})`;
				return;
			}
			status = data as StatusResponse;
		} catch (e) {
			statusErr = e instanceof Error ? e.message : 'Network error';
		} finally {
			statusLoading = false;
		}
	}

	$effect(() => {
		void fetchStatus();
	});

	async function saveCookie() {
		saveErr = '';
		saveOk = false;

		const candidate = cookieInput.trim();
		if (!candidate) {
			saveErr = 'Cookie value is required.';
			return;
		}

		saving = true;
		try {
			const res = await fetch('/api/church-guide/admin/cookie', {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ cookie: candidate })
			});
			const raw = await res.text();
			let data: StatusResponse & ErrorBody;
			try {
				data = JSON.parse(raw) as StatusResponse & ErrorBody;
			} catch {
				saveErr =
					res.status === 502
						? 'Could not reach the API (is the Go server running on port 8080?).'
						: `Invalid response (${res.status}).`;
				return;
			}
			if (!res.ok) {
				saveErr = authMessage(res.status) || data.error || `Request failed (${res.status})`;
				return;
			}
			status = data as StatusResponse;
			saveOk = true;
			cookieInput = '';
		} catch (e) {
			saveErr = e instanceof Error ? e.message : 'Network error';
		} finally {
			saving = false;
		}
	}
</script>

<svelte:head>
	<title>Church Guide Admin</title>
</svelte:head>

<main class="wrap">
	<h1>Church Guide admin</h1>
	<p class="lede">
		View and refresh the Groups Portal session cookie the Church Guide fetch depends on. This page is protected
		by a separate password — your browser will prompt for it.
	</p>

	<section class="card">
		<h2>Status</h2>
		{#if statusLoading}
			<p class="muted">Checking…</p>
		{:else if statusErr}
			<p class="error" role="alert">{statusErr}</p>
			<button type="button" class="btn-ghost" onclick={() => void fetchStatus()}>Retry</button>
		{:else if status?.configured}
			<p>
				<strong>Configured</strong> — last updated {status.updatedAt
					? formattedUpdatedAt(status.updatedAt)
					: 'unknown'}
			</p>
		{:else}
			<p><strong>Not configured</strong> — Church Guide fetches will fail until a cookie is saved below.</p>
		{/if}
	</section>

	<section class="card">
		<h2>Update cookie</h2>
		<p class="muted">
			From a browser already logged into <code>mycgconnect.com</code>: open DevTools → <strong>Network</strong>,
			click any request to <code>mycgconnect.com/api/…</code>, then under <strong>Headers → Request Headers</strong>
			copy the full <code>cookie</code> value (starts with <code>connect.sid=</code>).
		</p>
		<form
			onsubmit={(e) => {
				e.preventDefault();
				void saveCookie();
			}}
		>
			<label class="field">
				<span>New cookie value</span>
				<input
					type="text"
					bind:value={cookieInput}
					placeholder="connect.sid=…"
					autocomplete="off"
					spellcheck="false"
				/>
			</label>
			<button type="submit" disabled={saving}>{saving ? 'Validating…' : 'Save'}</button>
		</form>

		{#if saveOk}
			<p class="success" role="status">Saved. The next Church Guide fetch will use this cookie.</p>
		{/if}
		{#if saveErr}
			<p class="error" role="alert">{saveErr}</p>
		{/if}
	</section>
</main>

<style>
	.wrap {
		max-width: 42rem;
		margin: 0 auto;
		padding: 2rem 1.25rem 3rem;
		font-family:
			system-ui,
			-apple-system,
			'Segoe UI',
			sans-serif;
		line-height: 1.5;
		color: #111827;
	}

	h1 {
		font-size: 1.75rem;
		font-weight: 650;
		margin: 0 0 0.5rem;
		letter-spacing: -0.02em;
	}

	h2 {
		font-size: 1rem;
		font-weight: 650;
		margin: 0;
	}

	.lede {
		margin: 0 0 1.5rem;
		color: #4b5563;
		font-size: 0.95rem;
	}

	.card {
		background: #f9fafb;
		border: 1px solid #e5e7eb;
		border-radius: 12px;
		padding: 1.25rem 1.25rem 1.35rem;
		margin-bottom: 1rem;
		display: flex;
		flex-direction: column;
		gap: 0.75rem;
	}

	.card p {
		margin: 0;
		font-size: 0.9rem;
	}

	form {
		display: flex;
		flex-direction: column;
		gap: 1rem;
	}

	.field {
		display: flex;
		flex-direction: column;
		gap: 0.35rem;
		font-size: 0.875rem;
		font-weight: 500;
		color: #374151;
	}

	.field input {
		font: inherit;
		padding: 0.55rem 0.65rem;
		border-radius: 8px;
		border: 1px solid #d1d5db;
		background: #fff;
	}

	.field input:focus {
		outline-offset: 2px;
		outline: 2px solid #2563eb;
	}

	button[type='submit'] {
		align-self: flex-start;
		font: inherit;
		font-weight: 600;
		padding: 0.55rem 1.1rem;
		border-radius: 8px;
		border: none;
		background: #111827;
		color: #fff;
		cursor: pointer;
	}

	button[type='submit']:disabled {
		opacity: 0.55;
		cursor: not-allowed;
	}

	.btn-ghost {
		align-self: flex-start;
		font: inherit;
		font-size: 0.8rem;
		padding: 0.35rem 0.65rem;
		border-radius: 6px;
		border: 1px solid #d1d5db;
		background: #fff;
		color: #374151;
		cursor: pointer;
	}

	.btn-ghost:hover {
		background: #f3f4f6;
	}

	.error {
		color: #b91c1c;
		font-size: 0.9rem;
		margin: 0.25rem 0 0;
	}

	.success {
		color: #15803d;
		font-size: 0.9rem;
		margin: 0.25rem 0 0;
	}

	.muted {
		margin: 0;
		font-size: 0.85rem;
		color: #6b7280;
	}

	code {
		font-size: 0.9em;
		background: #eef2ff;
		padding: 0.12rem 0.35rem;
		border-radius: 4px;
	}
</style>
