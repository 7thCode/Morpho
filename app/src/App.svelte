<script>
  const API = 'http://localhost:8765'

  let tab = 'analyze'
  let inputText = ''
  let corpus = ''
  let morphemes = []
  let loading = false
  let error = ''
  let trainMessage = ''

  async function analyze() {
    if (!inputText.trim()) return
    loading = true
    error = ''
    morphemes = []
    try {
      const res = await fetch(`${API}/analyze`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ text: inputText }),
      })
      if (!res.ok) throw new Error(await res.text())
      const data = await res.json()
      morphemes = data.morphemes ?? []
    } catch (e) {
      error = e.message
    } finally {
      loading = false
    }
  }

  async function train() {
    if (!corpus.trim()) return
    loading = true
    error = ''
    trainMessage = ''
    try {
      const res = await fetch(`${API}/train`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ corpus }),
      })
      if (!res.ok) throw new Error(await res.text())
      trainMessage = '学習完了・辞書を保存しました'
    } catch (e) {
      error = e.message
    } finally {
      loading = false
    }
  }

  function handleKeydown(e) {
    if ((e.metaKey || e.ctrlKey) && e.key === 'Enter') {
      if (tab === 'analyze') analyze()
      else train()
    }
  }
</script>

<main>
  <header>
    <h1>Morpho</h1>
    <p>形態素解析エンジン</p>
  </header>

  <div class="tabs">
    <button class:active={tab === 'analyze'} on:click={() => tab = 'analyze'}>解析</button>
    <button class:active={tab === 'train'} on:click={() => tab = 'train'}>学習</button>
  </div>

  <div class="card">
    {#if tab === 'analyze'}
      <textarea
        bind:value={inputText}
        on:keydown={handleKeydown}
        placeholder="解析するテキストを入力… (⌘+Enter で実行)"
        rows="5"
      ></textarea>
      <div class="actions">
        <button class="primary" on:click={analyze} disabled={loading || !inputText.trim()}>
          {loading ? '解析中…' : '解析'}
        </button>
        {#if morphemes.length > 0}
          <span class="count">{morphemes.length} 形態素</span>
        {/if}
      </div>

      {#if error}<p class="error">{error}</p>{/if}

      {#if morphemes.length > 0}
        <table>
          <thead>
            <tr><th>#</th><th>表層形</th><th>品詞</th></tr>
          </thead>
          <tbody>
            {#each morphemes as m, i}
              <tr>
                <td class="num">{i + 1}</td>
                <td class="surface">{m.surface}</td>
                <td class="pos">{m.pos}</td>
              </tr>
            {/each}
          </tbody>
        </table>
      {/if}

    {:else}
      <textarea
        bind:value={corpus}
        on:keydown={handleKeydown}
        placeholder="学習コーパスを入力… (⌘+Enter で実行)"
        rows="8"
      ></textarea>
      <div class="actions">
        <button class="primary" on:click={train} disabled={loading || !corpus.trim()}>
          {loading ? '学習中…' : '学習・保存'}
        </button>
      </div>

      {#if error}<p class="error">{error}</p>{/if}
      {#if trainMessage}<p class="success">{trainMessage}</p>{/if}
    {/if}
  </div>
</main>

<style>
  :global(*, *::before, *::after) { box-sizing: border-box; margin: 0; padding: 0; }
  :global(body) {
    font-family: -apple-system, 'Helvetica Neue', Arial, 'Hiragino Sans', sans-serif;
    background: #f0f0f0;
    color: #1a1a1a;
    -webkit-font-smoothing: antialiased;
  }

  main { max-width: 760px; margin: 0 auto; padding: 2rem 1.5rem; }

  header { margin-bottom: 1.5rem; }
  h1 { font-size: 1.6rem; font-weight: 700; letter-spacing: -0.02em; }
  header p { color: #888; font-size: 0.85rem; margin-top: 0.2rem; }

  .tabs { display: flex; gap: 0.25rem; margin-bottom: 1rem; }
  .tabs button {
    padding: 0.35rem 1rem;
    border: 1px solid #ddd;
    background: #fff;
    border-radius: 6px;
    cursor: pointer;
    font-size: 0.9rem;
    color: #555;
    transition: all 0.15s;
  }
  .tabs button.active { background: #1a1a1a; color: #fff; border-color: #1a1a1a; }

  .card {
    background: #fff;
    border-radius: 10px;
    padding: 1.5rem;
    box-shadow: 0 1px 3px rgba(0,0,0,.08), 0 0 0 1px rgba(0,0,0,.04);
  }

  textarea {
    width: 100%;
    padding: 0.75rem;
    border: 1px solid #e0e0e0;
    border-radius: 6px;
    font-size: 1rem;
    font-family: inherit;
    resize: vertical;
    outline: none;
    transition: border-color 0.15s;
    line-height: 1.6;
  }
  textarea:focus { border-color: #888; }

  .actions { display: flex; align-items: center; gap: 1rem; margin-top: 0.75rem; }

  button.primary {
    padding: 0.45rem 1.4rem;
    background: #1a1a1a;
    color: #fff;
    border: none;
    border-radius: 6px;
    cursor: pointer;
    font-size: 0.9rem;
    font-family: inherit;
    transition: background 0.15s;
  }
  button.primary:hover:not(:disabled) { background: #333; }
  button.primary:disabled { background: #bbb; cursor: not-allowed; }

  .count { color: #888; font-size: 0.85rem; }

  .error {
    margin-top: 0.75rem;
    padding: 0.6rem 0.75rem;
    background: #fff0f0;
    border-left: 3px solid #e55;
    border-radius: 4px;
    font-size: 0.9rem;
    color: #c00;
  }
  .success {
    margin-top: 0.75rem;
    padding: 0.6rem 0.75rem;
    background: #f0fff4;
    border-left: 3px solid #4c8;
    border-radius: 4px;
    font-size: 0.9rem;
    color: #2a6;
  }

  table { width: 100%; margin-top: 1.25rem; border-collapse: collapse; font-size: 0.92rem; }
  th {
    text-align: left;
    padding: 0.5rem 0.75rem;
    background: #f8f8f8;
    border-bottom: 2px solid #e8e8e8;
    font-weight: 600;
    color: #555;
    font-size: 0.8rem;
    text-transform: uppercase;
    letter-spacing: 0.04em;
  }
  td { padding: 0.5rem 0.75rem; border-bottom: 1px solid #f0f0f0; }
  tr:last-child td { border-bottom: none; }
  tr:hover td { background: #fafafa; }

  .num { color: #bbb; font-size: 0.8rem; width: 2.5rem; }
  .surface { font-weight: 500; }
  .pos { color: #666; }
</style>
