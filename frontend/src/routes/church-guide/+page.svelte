<script lang="ts">
	type GuideResponse = {
		title: string;
		weekStartDate: string;
		text: string;
	};

	type ErrorBody = { error?: string };

	let loading = $state(false);
	let errorMsg = $state('');
	let result = $state<GuideResponse | null>(null);

	function formattedWeek(r: GuideResponse): string {
		const d = new Date(r.weekStartDate);
		if (Number.isNaN(d.getTime())) return r.weekStartDate;
		return d.toLocaleDateString(undefined, { year: 'numeric', month: 'long', day: 'numeric' });
	}

	async function fetchGuide() {
		errorMsg = '';
		result = null;
		loading = true;
		try {
			const res = await fetch('/api/church-guide');
			const raw = await res.text();
			let data: GuideResponse & ErrorBody;
			try {
				data = JSON.parse(raw) as GuideResponse & ErrorBody;
			} catch {
				errorMsg =
					res.status === 502
						? 'Could not reach the API (is the Go server running on port 8080?).'
						: `Invalid response (${res.status}).`;
				return;
			}
			if (!res.ok) {
				errorMsg =
					data.error ??
					(res.status === 502
						? 'Could not reach the API (is the Go server running on port 8080?).'
						: `Request failed (${res.status})`);
				return;
			}
			result = data as GuideResponse;
		} catch (e) {
			errorMsg = e instanceof Error ? e.message : 'Network error';
		} finally {
			loading = false;
		}
	}

	async function copyText() {
		if (!result) return;
		try {
			await navigator.clipboard.writeText(result.text);
		} catch {
			errorMsg = 'Could not copy to clipboard';
		}
	}
</script>

<svelte:head>
	<title>Church Guide</title>
</svelte:head>

<main class="wrap">
	<h1>Church Guide</h1>
	<p class="lede">
		Fetch this week's Community Group discussion guide from the Groups Portal. For use on your private network
		only.
	</p>

	<div class="card">
		<button type="button" onclick={() => void fetchGuide()} disabled={loading}>
			{loading ? 'Fetching…' : "Get this week's guide"}
		</button>
	</div>

	{#if errorMsg}
		<p class="error" role="alert">{errorMsg}</p>
	{/if}

	{#if result}
		<section class="card meta" aria-live="polite">
			<p class="guide-title">{result.title}</p>
			<p class="muted">Week of {formattedWeek(result)}</p>
			<div class="actions">
				<button type="button" onclick={() => void copyText()}>Copy</button>
				<a class="btn-link" href="/api/church-guide/pdf">Download PDF</a>
			</div>
			<textarea readonly rows="18">{result.text}</textarea>
		</section>
	{/if}
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
		gap: 1rem;
	}

	button {
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

	button:disabled {
		opacity: 0.55;
		cursor: not-allowed;
	}

	.guide-title {
		font-size: 1.1rem;
		font-weight: 650;
		margin: 0 0 0.35rem;
		color: #111827;
		line-height: 1.35;
	}

	.meta textarea {
		width: 100%;
		box-sizing: border-box;
		margin-top: 0.75rem;
		font:
			0.9rem/1.45 ui-monospace,
			monospace;
		border-radius: 8px;
		border: 1px solid #d1d5db;
		padding: 0.65rem;
		resize: vertical;
		background: #fff;
	}

	.actions {
		display: flex;
		gap: 0.5rem;
		flex-wrap: wrap;
		margin-top: 0.35rem;
	}

	.actions button,
	.btn-link {
		font: inherit;
		font-size: 0.875rem;
		font-weight: 500;
		padding: 0.45rem 0.85rem;
		border-radius: 8px;
		border: 1px solid #d1d5db;
		background: #fff;
		color: #111827;
		cursor: pointer;
		text-decoration: none;
	}

	.actions button:hover,
	.btn-link:hover {
		background: #f3f4f6;
	}

	.error {
		color: #b91c1c;
		font-size: 0.9rem;
		margin: 0.25rem 0 1rem;
	}

	.muted {
		margin: 0;
		font-size: 0.85rem;
		color: #6b7280;
	}
</style>
