<script lang="ts">
	type TranscriptResponse = {
		videoId: string;
		videoTitle?: string;
		channelTitle?: string;
		lang: string;
		language: string;
		isGenerated: boolean;
		text: string;
		textTimestamped?: string;
		snippetCount: number;
	};

	type ErrorBody = { error?: string };

	type LogEntry = { time: string; level: string; msg: string };

	let urlInput = $state('');
	let langInput = $state('en');
	let loading = $state(false);
	let errorMsg = $state('');
	let result = $state<TranscriptResponse | null>(null);

	let logOpen = $state(false);
	let includeTimestamps = $state(false);

	function displayBody(r: TranscriptResponse): string {
		if (includeTimestamps && r.textTimestamped) return r.textTimestamped;
		return r.text;
	}

	function safeFileSlug(r: TranscriptResponse): string {
		const raw = (r.videoTitle ?? r.videoId).replace(/[^\w\-]+/g, '-').replace(/^-|-$/g, '');
		return raw.slice(0, 80) || r.videoId;
	}
	let logEntries = $state<LogEntry[]>([]);
	let logErr = $state('');
	let logPre = $state<HTMLPreElement | null>(null);

	function formatLog(entries: LogEntry[]): string {
		if (!entries.length) return '— no entries yet —';
		return entries.map((e) => `${e.time} [${e.level}] ${e.msg}`).join('\n');
	}

	async function refreshLogs() {
		logErr = '';
		try {
			const res = await fetch('/api/logs?limit=120');
			const raw = await res.text();
			try {
				const j = JSON.parse(raw) as { entries?: LogEntry[] };
				logEntries = j.entries ?? [];
			} catch {
				logErr = 'Invalid log response (is the API running?)';
			}
		} catch {
			logErr = 'Could not load server log';
		}
	}

	async function clearServerLog() {
		try {
			await fetch('/api/logs/clear', { method: 'POST' });
			await refreshLogs();
		} catch {
			logErr = 'Clear failed';
		}
	}

	$effect(() => {
		if (!logOpen) return;
		let cancelled = false;
		void refreshLogs();
		const id = setInterval(() => {
			if (!cancelled) void refreshLogs();
		}, 1500);
		return () => {
			cancelled = true;
			clearInterval(id);
		};
	});

	$effect(() => {
		const _lines = logEntries;
		const el = logPre;
		if (!el || !logOpen) return;
		queueMicrotask(() => {
			el.scrollTop = el.scrollHeight;
		});
	});

	async function submit() {
		errorMsg = '';
		result = null;
		loading = true;
		try {
			const res = await fetch('/api/transcript', {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({
					url: urlInput.trim(),
					lang: langInput.trim() || 'en',
					includeTimestamps
				})
			});
			const raw = await res.text();
			let data: TranscriptResponse & ErrorBody;
			try {
				data = JSON.parse(raw) as TranscriptResponse & ErrorBody;
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
			if ('error' in data && data.error) {
				errorMsg = data.error;
				return;
			}
			result = data as TranscriptResponse;
		} catch (e) {
			errorMsg = e instanceof Error ? e.message : 'Network error';
		} finally {
			loading = false;
			if (logOpen) void refreshLogs();
		}
	}

	async function copyText() {
		if (!result) return;
		const body = displayBody(result);
		if (!body) return;
		try {
			await navigator.clipboard.writeText(body);
		} catch {
			errorMsg = 'Could not copy to clipboard';
		}
	}

	function downloadText() {
		if (!result) return;
		const body = displayBody(result);
		if (!body) return;
		const blob = new Blob([body], { type: 'text/plain;charset=utf-8' });
		const a = document.createElement('a');
		a.href = URL.createObjectURL(blob);
		a.download = `${safeFileSlug(result)}-transcript.txt`;
		a.click();
		URL.revokeObjectURL(a.href);
	}
</script>

<svelte:head>
	<title>YouTube Transcript</title>
</svelte:head>

<main class="wrap">
	<h1>YouTube transcript</h1>
	<p class="lede">
		Paste a video link or ID to fetch captions (when YouTube exposes them). For use on your private network
		only.
	</p>

	<form
		onsubmit={(e) => {
			e.preventDefault();
			void submit();
		}}
		class="card"
	>
		<label class="field">
			<span>YouTube URL or video ID</span>
			<input type="text" bind:value={urlInput} placeholder="https://www.youtube.com/watch?v=…" />
		</label>
		<label class="field">
			<span>Language code(s)</span>
			<input type="text" bind:value={langInput} placeholder="en or de,en for fallback" />
		</label>
		<label class="field row">
			<input type="checkbox" bind:checked={includeTimestamps} />
			<span>Include timestamps (lines start with <code>[mm:ss]</code> or <code>[hh:mm:ss]</code>); submit again after toggling</span>
		</label>
		<button type="submit" disabled={loading}> {loading ? 'Fetching…' : 'Get transcript'} </button>
	</form>

	{#if errorMsg}
		<p class="error" role="alert">{errorMsg}</p>
	{/if}

	{#if result}
		<section class="card meta" aria-live="polite">
			{#if result.videoTitle}
				<p class="video-title">{result.videoTitle}</p>
			{/if}
			{#if result.channelTitle}
				<p class="channel-line"><strong>Channel</strong> {result.channelTitle}</p>
			{/if}
			<p>
				<strong>Video ID</strong>
				<code>{result.videoId}</code>
			</p>
			<p>
				<strong>Track</strong>
				{result.language} ({result.lang}) — {result.isGenerated ? 'auto-generated' : 'manual'}
			</p>
			<p class="muted">{result.snippetCount} lines</p>
			<div class="actions">
				<button type="button" onclick={() => void copyText()}>Copy</button>
				<button type="button" onclick={() => downloadText()}>Download .txt</button>
			</div>
			<textarea readonly rows="18">{displayBody(result)}</textarea>
		</section>
	{/if}

	<section class="log-panel" aria-label="Server API log">
		<div class="log-toolbar">
			<button type="button" class="btn-ghost" onclick={() => (logOpen = !logOpen)}>
				{logOpen ? 'Hide' : 'Show'} server log
			</button>
			{#if logOpen}
				<button type="button" class="btn-ghost" onclick={() => void refreshLogs()}>Refresh</button>
				<button type="button" class="btn-ghost danger" onclick={() => void clearServerLog()}>Clear buffer</button>
			{/if}
		</div>
		{#if logOpen}
			{#if logErr}<p class="log-err">{logErr}</p>{/if}
			<pre class="console" bind:this={logPre}>{formatLog(logEntries)}</pre>
			<p class="log-hint">In-memory log from the Go server (last ~250 lines). Polls while open.</p>
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

	.field.row {
		flex-direction: row;
		align-items: flex-start;
		gap: 0.5rem;
	}

	.field.row input[type='checkbox'] {
		margin-top: 0.2rem;
	}

	.field.row span {
		font-weight: 400;
		color: #4b5563;
		font-size: 0.85rem;
		line-height: 1.4;
	}

	.video-title {
		font-size: 1.1rem;
		font-weight: 650;
		margin: 0 0 0.35rem;
		color: #111827;
		line-height: 1.35;
	}

	.channel-line {
		margin: 0 0 0.5rem;
		font-size: 0.9rem;
		color: #374151;
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

	.actions button {
		font: inherit;
		padding: 0.45rem 0.85rem;
		border-radius: 8px;
		border: 1px solid #d1d5db;
		background: #fff;
		cursor: pointer;
	}

	.actions button:hover {
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

	code {
		font-size: 0.9em;
		background: #eef2ff;
		padding: 0.12rem 0.35rem;
		border-radius: 4px;
	}

	.meta p {
		margin: 0.2rem 0;
		font-size: 0.9rem;
	}

	.log-panel {
		margin-top: 1.5rem;
		padding-top: 1rem;
		border-top: 1px solid #e5e7eb;
	}

	.log-toolbar {
		display: flex;
		flex-wrap: wrap;
		gap: 0.5rem;
		align-items: center;
		margin-bottom: 0.5rem;
	}

	.btn-ghost {
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

	.btn-ghost.danger {
		color: #b91c1c;
		border-color: #fecaca;
	}

	.console {
		margin: 0;
		max-height: 10.5rem;
		overflow: auto;
		padding: 0.55rem 0.65rem;
		font-size: 0.72rem;
		line-height: 1.35;
		font-family: ui-monospace, 'Cascadia Code', 'SF Mono', Menlo, monospace;
		background: #1e1e1e;
		color: #d4d4d4;
		border-radius: 8px;
		border: 1px solid #333;
		white-space: pre-wrap;
		word-break: break-word;
	}

	.log-err {
		margin: 0 0 0.4rem;
		font-size: 0.8rem;
		color: #b45309;
	}

	.log-hint {
		margin: 0.35rem 0 0;
		font-size: 0.72rem;
		color: #9ca3af;
	}
</style>
