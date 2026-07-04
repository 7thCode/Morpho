<script>
  import { onMount } from 'svelte'
  import { Analyze, Train, GetStats, GetEntries, SaveWord, DeleteWord } from '../wailsjs/go/main/App.js'

  let tab = 'analyze'
  let inputText = ''
  let corpus = ''
  let morphemes = []
  let loading = false
  let error = ''
  let trainMessage = ''

  // 統計情報
  let stats = { word_count: 0, is_trained: false, pos_tags: [] }
  let entries = []
  let filteredEntries = []

  // 辞書タブ用
  let searchQuery = ''
  let selectedPOS = ''
  let sortBy = 'freq'
  let sortAsc = false

  // モーダル編集用ステート
  let showEditModal = false
  let modalTitle = '単語を追加'
  let editSurface = ''
  let editPOS = '名詞'
  let editFreq = 1
  let isEditingExisting = false
  let originalSurface = ''
  let modalError = ''

  // 削除確認モーダル用ステート
  let showConfirmModal = false
  let confirmMessage = ''
  let confirmError = ''
  let confirmCallback = null

  // リアルタイム解析デバウンス
  let debounceTimer
  function handleInput() {
    clearTimeout(debounceTimer)
    debounceTimer = setTimeout(analyze, 300)
  }

  // 統計データの取得
  async function loadStats() {
    try {
      stats = await GetStats()
    } catch (e) {
      console.error('Failed to load stats:', e)
    }
  }

  // 辞書データの取得
  async function loadEntries() {
    try {
      const result = await GetEntries()
      entries = result ?? []
      filterAndSortEntries()
    } catch (e) {
      console.error('Failed to load entries:', e)
    }
  }

  // 辞書エントリーのフィルタリングとソート
  function filterAndSortEntries() {
    let result = [...entries]

    // 検索
    if (searchQuery.trim()) {
      const q = searchQuery.toLowerCase()
      result = result.filter(e => e.surface.toLowerCase().includes(q) || e.pos.toLowerCase().includes(q))
    }

    // 品詞フィルタ
    if (selectedPOS) {
      result = result.filter(e => e.pos === selectedPOS)
    }

    // ソート
    result.sort((a, b) => {
      let valA = a[sortBy]
      let valB = b[sortBy]

      if (typeof valA === 'string') {
        valA = valA.toLowerCase()
        valB = valB.toLowerCase()
      }

      if (valA < valB) return sortAsc ? -1 : 1
      if (valA > valB) return sortAsc ? 1 : -1
      return 0
    })

    filteredEntries = result
  }

  // 検索条件やソート順が変わったら更新
  $: {
    if (entries.length > 0 || searchQuery || selectedPOS || sortBy || sortAsc) {
      filterAndSortEntries()
    }
  }

  async function analyze() {
    if (!inputText.trim()) {
      morphemes = []
      return
    }
    loading = true
    error = ''
    try {
      const result = await Analyze(inputText)
      morphemes = result ?? []
    } catch (e) {
      error = typeof e === 'string' ? e : e.message
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
      await Train(corpus)
      trainMessage = '学習が完了し、モデルと辞書を保存しました！'
      await loadStats()
      await loadEntries()
    } catch (e) {
      error = typeof e === 'string' ? e : e.message
    } finally {
      loading = false
    }
  }

  const SAMPLE_CORPUS = `私は昨日図書館で面白い本を読みました。
すももももももももののうち。
美味しいラーメンを食べに行きたいです。
日本語の形態素解析エンジンを作っています。
吾輩は猫である。名前はまだ無い。
桜の花が美しく咲く春が来ました。
彼は新幹線で東京へ行きました。`;

  function loadSample() {
    corpus = SAMPLE_CORPUS
  }

  function handleKeydown(e) {
    if ((e.metaKey || e.ctrlKey) && e.key === 'Enter') {
      if (tab === 'analyze') analyze()
      else train()
    }
  }

  function switchTab(newTab) {
    tab = newTab
    if (newTab === 'dict') {
      loadEntries()
    }
    loadStats()
  }

  // 単語ソートのトグル
  function toggleSort(field) {
    if (sortBy === field) {
      sortAsc = !sortAsc
    } else {
      sortBy = field
      sortAsc = false
    }
  }

  // 単語追加モーダルを開く
  function openAddModal() {
    modalTitle = '単語を辞書に追加'
    editSurface = ''
    editPOS = '名詞'
    editFreq = 1
    isEditingExisting = false
    originalSurface = ''
    modalError = ''
    showEditModal = true
  }

  // 単語編集モーダルを開く
  function openEditModal(entry) {
    modalTitle = '単語エントリーの編集'
    editSurface = entry.surface
    editPOS = entry.pos
    editFreq = entry.freq
    isEditingExisting = true
    originalSurface = entry.surface
    modalError = ''
    showEditModal = true
  }

  // モーダルを閉じる
  function closeModal() {
    showEditModal = false
  }

  // 単語の保存処理 (新規・更新)
  async function saveWord() {
    if (!editSurface.trim()) {
      modalError = '単語の表層形を入力してください。'
      return
    }
    if (editFreq < 0) {
      modalError = '出現頻度には0以上の数値を指定してください。'
      return
    }
    try {
      // 既存の編集でキー（表層形）が変わった場合、古いエントリーを先に削除する
      if (isEditingExisting && originalSurface !== editSurface.trim()) {
        await DeleteWord(originalSurface)
      }
      await SaveWord(editSurface.trim(), editPOS, parseInt(editFreq, 10))
      showEditModal = false
      await loadStats()
      await loadEntries()
    } catch (e) {
      modalError = typeof e === 'string' ? e : e.message
    }
  }

  // 単語の削除処理（確認モーダルの呼び出し）
  function deleteWord(surface) {
    confirmMessage = `単語「${surface}」を辞書から本当に削除しますか？`
    confirmError = ''
    confirmCallback = async () => {
      try {
        await DeleteWord(surface)
        await loadStats()
        await loadEntries()
        showConfirmModal = false
      } catch (e) {
        confirmError = typeof e === 'string' ? e : (e.message || '削除に失敗しました')
      }
    }
    showConfirmModal = true
  }

  // 解析結果の品詞割合統計の計算
  let posStats = []
  $: {
    if (morphemes.length > 0) {
      const counts = {}
      morphemes.forEach(m => {
        counts[m.pos] = (counts[m.pos] || 0) + 1
      })
      posStats = Object.entries(counts).map(([pos, count]) => ({
        pos,
        count,
        percentage: ((count / morphemes.length) * 100).toFixed(1)
      })).sort((a, b) => b.count - a.count)
    } else {
      posStats = []
    }
  }

  // 品詞カラーバッジのマッピング
  const POS_COLORS = {
    '名詞': 'noun',
    '動詞': 'verb',
    '形容詞': 'adj',
    '助詞': 'particle',
    '助動詞': 'aux',
    '副詞': 'adv',
    '記号': 'symbol',
    '数詞': 'num',
    '外来語': 'foreign',
    '未知語': 'unknown'
  }
  function getPOSClass(pos) {
    return POS_COLORS[pos] || 'default'
  }

  onMount(async () => {
    await loadStats()
  })
</script>

<div class="titlebar"></div>

<main>
  <header class="app-header">
    <div class="header-logo">
      <div class="logo-icon">M</div>
      <div class="title-meta">
        <h1>Morpho</h1>
        <p>HMM-Based Japanese Morphological Analyzer</p>
      </div>
    </div>
    
    <div class="stats-badge-container">
      <div class="mini-stat-card">
        <span class="label">辞書単語数</span>
        <span class="value">{stats.word_count.toLocaleString()}</span>
      </div>
      <div class="mini-stat-card">
        <span class="label">モデル学習</span>
        <span class="value status" class:trained={stats.is_trained}>
          {stats.is_trained ? '学習済 (HMM)' : '未学習 (Heuristic)'}
        </span>
      </div>
    </div>
  </header>

  <div class="tabs">
    <button class:active={tab === 'analyze'} on:click={() => switchTab('analyze')}>
      <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="tab-icon"><path d="M2 12s3-7 10-7 10 7 10 7-3 7-10 7-10-7-10-7Z"/><circle cx="12" cy="12" r="3"/></svg>
      解析
    </button>
    <button class:active={tab === 'train'} on:click={() => switchTab('train')}>
      <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="tab-icon"><path d="M12 2v20M17 5H9.5a3.5 3.5 0 0 0 0 7h5a3.5 3.5 0 0 1 0 7H6"/></svg>
      学習 (Train)
    </button>
    <button class:active={tab === 'dict'} on:click={() => switchTab('dict')}>
      <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="tab-icon"><path d="M4 19.5v-15A2.5 2.5 0 0 1 6.5 2H20v20H6.5a2.5 2.5 0 0 1-2.5-2.5Z"/><path d="M6 6h10M6 10h10"/></svg>
      辞書ブラウザ
    </button>
  </div>

  <div class="card glass-card">
    {#if tab === 'analyze'}
      <div class="input-section">
        <textarea
          bind:value={inputText}
          on:input={handleInput}
          on:keydown={handleKeydown}
          placeholder="解析する日本語テキストを入力してください… (リアルタイムで解析されます)"
          rows="5"
        ></textarea>
        <div class="actions">
          <button class="btn btn-primary" on:click={analyze} disabled={loading || !inputText.trim()}>
            {#if loading}
              <span class="spinner"></span>解析中…
            {:else}
              再解析
            {/if}
          </button>
          {#if morphemes.length > 0}
            <div class="meta-info fade-in">
              <span class="count">{morphemes.length} 形態素</span>
            </div>
          {/if}
        </div>
      </div>

      {#if error}<p class="message error fade-in">{error}</p>{/if}

      {#if morphemes.length > 0}
        <div class="results-layout fade-in">
          <!-- ビジュアル統計割合 -->
          <div class="pos-chart-card">
            <h3>品詞の構成比</h3>
            <div class="chart-bars">
              {#each posStats as stat}
                <div class="chart-row">
                  <div class="chart-label-group">
                    <span class="pos-tag-label {getPOSClass(stat.pos)}">{stat.pos}</span>
                    <span class="percentage-val">{stat.percentage}% ({stat.count}語)</span>
                  </div>
                  <div class="bar-container">
                    <div class="bar-fill {getPOSClass(stat.pos)}" style="width: {stat.percentage}%"></div>
                  </div>
                </div>
              {/each}
            </div>
          </div>

          <!-- 解析結果テーブル -->
          <div class="table-container">
            <table>
              <thead>
                <tr>
                  <th style="width: 60px;">#</th>
                  <th>表層形 (Word)</th>
                  <th>品詞 (POS)</th>
                </tr>
              </thead>
              <tbody>
                {#each morphemes as m, i}
                  <tr class="fade-in-row">
                    <td class="num">{i + 1}</td>
                    <td class="surface">{m.surface}</td>
                    <td>
                      <span class="pos-badge {getPOSClass(m.pos)}">{m.pos}</span>
                    </td>
                  </tr>
                {/each}
              </tbody>
            </table>
          </div>
        </div>
      {/if}

    {:else}
      {#if tab === 'train'}
        <div class="train-section">
          <div class="train-header">
            <h3>HMMモデルの学習</h3>
            <p class="description">
              スペース区切りの文や、教師ありテキストデータを入力して「学習・保存」を実行すると、遷移確率・放出確率がモデルに記録され、解析の精度が向上します。
            </p>
          </div>
          <textarea
            bind:value={corpus}
            on:keydown={handleKeydown}
            placeholder="学習用のコーパス（テキスト）を入力してください… (例: 私は/名詞 昨日/名詞 図書館/名詞 で/助詞 ...) または通常の文章を入力すると簡易的な自動学習が行われます。"
            rows="8"
          ></textarea>
          <div class="actions">
            <button class="btn btn-primary" on:click={train} disabled={loading || !corpus.trim()}>
              {#if loading}
                <span class="spinner"></span>学習中…
              {:else}
                学習を実行・保存
              {/if}
            </button>
            <button class="btn btn-secondary" on:click={loadSample} disabled={loading}>
              サンプル文章を読込
            </button>
          </div>

          {#if error}<p class="message error fade-in">{error}</p>{/if}
          {#if trainMessage}<p class="message success fade-in">{trainMessage}</p>{/if}
        </div>
      {:else}
        <!-- 辞書ブラウザ -->
        <div class="dict-section">
          <div class="dict-toolbar">
            <div class="search-box">
              <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" stroke-width="2" class="search-icon"><circle cx="11" cy="11" r="8"/><path d="m21 21-4.3-4.3"/></svg>
              <input
                type="text"
                bind:value={searchQuery}
                placeholder="単語または品詞で検索…"
              />
            </div>

            <div class="filter-group">
              <select bind:value={selectedPOS}>
                <option value="">すべての品詞</option>
                <option value="名詞">名詞</option>
                <option value="動詞">動詞</option>
                <option value="形容詞">形容詞</option>
                <option value="助詞">助詞</option>
                <option value="助動詞">助動詞</option>
                <option value="副詞">副詞</option>
                <option value="記号">記号</option>
                <option value="数詞">数詞</option>
                <option value="外来語">外来語</option>
                <option value="未知語">未知語</option>
              </select>
            </div>

            <!-- 単語追加ボタン -->
            <button class="btn btn-primary" style="margin-left: auto;" on:click={openAddModal}>
              <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" style="margin-right: 0.25rem;"><line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/></svg>
              単語を追加
            </button>
          </div>

          <div class="dict-table-container">
            <table>
              <thead>
                <tr>
                  <th class="sortable" on:click={() => toggleSort('surface')}>
                    単語表層形 {sortBy === 'surface' ? (sortAsc ? '▲' : '▼') : ''}
                  </th>
                  <th class="sortable" on:click={() => toggleSort('pos')}>
                    品詞 {sortBy === 'pos' ? (sortAsc ? '▲' : '▼') : ''}
                  </th>
                  <th class="sortable text-right" on:click={() => toggleSort('freq')}>
                    出現頻度 {sortBy === 'freq' ? (sortAsc ? '▲' : '▼') : ''}
                  </th>
                  <th style="width: 110px; text-align: center;">操作</th>
                </tr>
              </thead>
              <tbody>
                {#if filteredEntries.length === 0}
                  <tr>
                    <td colspan="4" class="empty-state">該当する単語が見つかりません</td>
                  </tr>
                {:else}
                  {#each filteredEntries as entry}
                    <tr class="fade-in-row">
                      <td class="surface-dict">{entry.surface}</td>
                      <td><span class="pos-badge {getPOSClass(entry.pos)}">{entry.pos}</span></td>
                      <td class="text-right freq-val">{entry.freq.toLocaleString()}</td>
                      <td style="text-align: center;">
                        <div class="row-actions">
                          <button class="action-btn edit" title="編集" on:click={() => openEditModal(entry)}>
                            <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" stroke-width="2"><path d="M12 20h9"/><path d="M16.5 3.5a2.12 2.12 0 0 1 3 3L7 19l-4 1 1-4Z"/></svg>
                          </button>
                          <button class="action-btn delete" title="削除" on:click={() => deleteWord(entry.surface)}>
                            <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" stroke-width="2"><path d="M3 6h18"/><path d="M19 6v14c0 1-1 2-2 2H7c-1 0-2-1-2-2V6M8 6V4c0-1 1-2 2-2h4c1 0 2 1 2 2v2"/><line x1="10" y1="11" x2="10" y2="17"/><line x1="14" y1="11" x2="14" y2="17"/></svg>
                          </button>
                        </div>
                      </td>
                    </tr>
                  {/each}
                {/if}
              </tbody>
            </table>
          </div>
          <div class="dict-footer">
            全 {entries.length} 語中 {filteredEntries.length} 語を表示
          </div>
        </div>
      {/if}
    {/if}
  </div>
</main>

<!-- 単語編集モーダル -->
{#if showEditModal}
  <div class="modal-overlay" on:click|self={closeModal}>
    <div class="modal-card glass-card fade-in">
      <div class="modal-header">
        <h2>{modalTitle}</h2>
        <button class="close-btn" on:click={closeModal}>&times;</button>
      </div>
      <div class="modal-body">
        {#if modalError}
          <p class="message error fade-in" style="margin-top: 0; margin-bottom: 1rem;">{modalError}</p>
        {/if}
        <div class="form-group">
          <label for="edit-surface">表層形 (単語)</label>
          <input id="edit-surface" type="text" bind:value={editSurface} placeholder="例: 人工知能" />
        </div>
        <div class="form-group">
          <label for="edit-pos">品詞</label>
          <select id="edit-pos" bind:value={editPOS}>
            <option value="名詞">名詞</option>
            <option value="動詞">動詞</option>
            <option value="形容詞">形容詞</option>
            <option value="助詞">助詞</option>
            <option value="助動詞">助動詞</option>
            <option value="副詞">副詞</option>
            <option value="記号">記号</option>
            <option value="数詞">数詞</option>
            <option value="外来語">外来語</option>
            <option value="未知語">未知語</option>
          </select>
        </div>
        <div class="form-group">
          <label for="edit-freq">出現頻度</label>
          <input id="edit-freq" type="number" min="0" bind:value={editFreq} />
        </div>
      </div>
      <div class="modal-footer">
        <button class="btn btn-secondary" on:click={closeModal}>キャンセル</button>
        <button class="btn btn-primary" on:click={saveWord}>保存</button>
      </div>
    </div>
  </div>
{/if}

<!-- 削除確認モーダル -->
{#if showConfirmModal}
  <div class="modal-overlay" on:click|self={() => showConfirmModal = false}>
    <div class="modal-card glass-card fade-in" style="max-width: 400px;">
      <div class="modal-header">
        <h2>削除の確認</h2>
        <button class="close-btn" on:click={() => showConfirmModal = false}>&times;</button>
      </div>
      <div class="modal-body">
        {#if confirmError}
          <p class="message error fade-in" style="margin-top: 0; margin-bottom: 1rem;">{confirmError}</p>
        {/if}
        <p class="description" style="font-size: 1rem; color: #fff; line-height: 1.5;">{confirmMessage}</p>
      </div>
      <div class="modal-footer">
        <button class="btn btn-secondary" on:click={() => showConfirmModal = false}>キャンセル</button>
        <button class="btn btn-primary" style="background: var(--pos-unknown); color: #fff; box-shadow: 0 4px 12px rgba(248, 113, 113, 0.2);" on:click={confirmCallback}>削除する</button>
      </div>
    </div>
  </div>
{/if}

<style>
  :global(:root) {
    /* プレミアムスペースダークテーマ */
    --bg-main: #0a0f1d;
    --bg-card: rgba(17, 24, 39, 0.7);
    --border-color: rgba(255, 255, 255, 0.08);
    --text-primary: #f3f4f6;
    --text-secondary: #9ca3af;
    --primary-gradient: linear-gradient(135deg, #6366f1, #8b5cf6);
    --primary-color: #6366f1;
    --primary-hover: #4f46e5;
    --accent-glow: rgba(99, 102, 241, 0.15);
    
    /* 品詞カラースキーム */
    --pos-noun: #60a5fa;      /* 青 */
    --pos-verb: #34d399;      /* 緑 */
    --pos-adj: #fbbf24;       /* 黄/オレンジ */
    --pos-particle: #c084fc;  /* 紫 */
    --pos-aux: #f472b6;       /* ピンク */
    --pos-adv: #2dd4bf;       /* ティール */
    --pos-symbol: #9ca3af;    /* グレー */
    --pos-num: #fb7185;       /* ローズ */
    --pos-foreign: #38bdf8;   /* ライトブルー */
    --pos-unknown: #f87171;   /* 赤 */
    --pos-default: #a7f3d0;
  }

  :global(*, *::before, *::after) {
    box-sizing: border-box;
    margin: 0;
    padding: 0;
  }

  :global(body) {
    font-family: 'Outfit', 'Inter', -apple-system, sans-serif;
    background: var(--bg-main);
    color: var(--text-primary);
    -webkit-font-smoothing: antialiased;
    min-height: 100vh;
    background-image: 
      radial-gradient(circle at 10% 20%, rgba(99, 102, 241, 0.05) 0%, transparent 40%),
      radial-gradient(circle at 90% 80%, rgba(139, 92, 246, 0.05) 0%, transparent 40%);
  }

  .titlebar {
    height: 32px;
    --wails-draggable: drag;
    background: rgba(10, 15, 29, 0.5);
    backdrop-filter: blur(10px);
  }

  main {
    max-width: 900px;
    margin: 0 auto;
    padding: 0 2rem 3rem;
  }

  /* ヘッダーデザイン */
  .app-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 2rem;
    padding-bottom: 1.5rem;
    border-bottom: 1px solid var(--border-color);
  }

  .header-logo {
    display: flex;
    align-items: center;
    gap: 1rem;
  }

  .logo-icon {
    width: 44px;
    height: 44px;
    background: var(--primary-gradient);
    border-radius: 12px;
    display: flex;
    align-items: center;
    justify-content: center;
    font-weight: 800;
    font-size: 1.5rem;
    color: #fff;
    box-shadow: 0 0 20px rgba(99, 102, 241, 0.4);
  }

  .title-meta h1 {
    font-size: 1.8rem;
    font-weight: 800;
    letter-spacing: -0.03em;
    background: linear-gradient(to right, #ffffff, #c7d2fe);
    -webkit-background-clip: text;
    -webkit-text-fill-color: transparent;
  }

  .title-meta p {
    color: var(--text-secondary);
    font-size: 0.8rem;
    margin-top: 0.15rem;
  }

  .stats-badge-container {
    display: flex;
    gap: 1rem;
  }

  .mini-stat-card {
    background: rgba(255, 255, 255, 0.03);
    border: 1px solid var(--border-color);
    padding: 0.5rem 1rem;
    border-radius: 10px;
    display: flex;
    flex-direction: column;
    align-items: flex-end;
  }

  .mini-stat-card .label {
    font-size: 0.65rem;
    color: var(--text-secondary);
    text-transform: uppercase;
    letter-spacing: 0.05em;
  }

  .mini-stat-card .value {
    font-size: 1rem;
    font-weight: 700;
    color: #fff;
  }

  .mini-stat-card .value.status {
    font-size: 0.8rem;
    color: #f87171;
  }

  .mini-stat-card .value.status.trained {
    color: #34d399;
  }

  /* タブのスタイル */
  .tabs {
    display: flex;
    gap: 0.5rem;
    margin-bottom: 1.5rem;
  }

  .tabs button {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    padding: 0.6rem 1.2rem;
    border: 1px solid var(--border-color);
    background: rgba(255, 255, 255, 0.02);
    color: var(--text-secondary);
    border-radius: 10px;
    cursor: pointer;
    font-size: 0.9rem;
    font-weight: 500;
    transition: all 0.2s cubic-bezier(0.4, 0, 0.2, 1);
  }

  .tabs button:hover {
    background: rgba(255, 255, 255, 0.05);
    color: #fff;
    border-color: rgba(255, 255, 255, 0.15);
  }

  .tabs button.active {
    background: var(--primary-gradient);
    color: #fff;
    border-color: transparent;
    box-shadow: 0 0 15px rgba(99, 102, 241, 0.3);
  }

  .tab-icon {
    opacity: 0.8;
  }

  /* グラスモフィズムカード */
  .glass-card {
    background: var(--bg-card);
    border-radius: 16px;
    padding: 2rem;
    border: 1px solid var(--border-color);
    box-shadow: 
      0 4px 30px rgba(0, 0, 0, 0.2),
      inset 0 1px 1px rgba(255, 255, 255, 0.03);
    backdrop-filter: blur(16px);
    -webkit-backdrop-filter: blur(16px);
  }

  /* テキストエリア */
  textarea {
    width: 100%;
    padding: 1rem;
    background: rgba(10, 15, 29, 0.5);
    border: 1px solid var(--border-color);
    border-radius: 12px;
    color: #fff;
    font-size: 1.05rem;
    font-family: inherit;
    resize: vertical;
    outline: none;
    transition: all 0.2s;
    line-height: 1.7;
  }

  textarea:focus {
    border-color: var(--primary-color);
    box-shadow: 0 0 0 3px var(--accent-glow);
  }

  /* ボタン */
  .actions {
    display: flex;
    align-items: center;
    gap: 1rem;
    margin-top: 1rem;
  }

  .btn {
    display: inline-flex;
    align-items: center;
    gap: 0.5rem;
    padding: 0.6rem 1.6rem;
    border-radius: 10px;
    cursor: pointer;
    font-size: 0.95rem;
    font-weight: 600;
    transition: all 0.2s;
    font-family: inherit;
    border: 1px solid transparent;
  }

  .btn-primary {
    background: var(--primary-gradient);
    color: #fff;
    box-shadow: 0 4px 12px rgba(99, 102, 241, 0.2);
  }

  .btn-primary:hover:not(:disabled) {
    transform: translateY(-1px);
    box-shadow: 0 6px 20px rgba(99, 102, 241, 0.35);
  }

  .btn-primary:disabled {
    background: #4b5563;
    color: #9ca3af;
    cursor: not-allowed;
    box-shadow: none;
    transform: none;
  }

  .btn-secondary {
    background: rgba(255, 255, 255, 0.05);
    color: var(--text-primary);
    border-color: var(--border-color);
  }

  .btn-secondary:hover:not(:disabled) {
    background: rgba(255, 255, 255, 0.1);
    color: #fff;
  }

  .meta-info {
    margin-left: auto;
  }

  .count {
    color: var(--text-secondary);
    font-size: 0.9rem;
    background: rgba(255, 255, 255, 0.03);
    padding: 0.3rem 0.8rem;
    border-radius: 20px;
    border: 1px solid var(--border-color);
  }

  /* ２カラムレイアウト */
  .results-layout {
    display: grid;
    grid-template-columns: 1fr 1.5fr;
    gap: 2rem;
    margin-top: 2rem;
    align-items: start;
  }

  @media (max-width: 768px) {
    .results-layout {
      grid-template-columns: 1fr;
    }
  }

  /* チャート表示 */
  .pos-chart-card {
    background: rgba(255, 255, 255, 0.015);
    border: 1px solid var(--border-color);
    border-radius: 12px;
    padding: 1.25rem;
  }

  .pos-chart-card h3 {
    font-size: 0.95rem;
    margin-bottom: 1rem;
    color: #fff;
    font-weight: 700;
  }

  .chart-bars {
    display: flex;
    flex-direction: column;
    gap: 0.85rem;
  }

  .chart-row {
    display: flex;
    flex-direction: column;
    gap: 0.35rem;
  }

  .chart-label-group {
    display: flex;
    justify-content: space-between;
    font-size: 0.8rem;
    align-items: center;
  }

  .percentage-val {
    color: var(--text-secondary);
    font-size: 0.75rem;
  }

  .bar-container {
    height: 6px;
    background: rgba(255, 255, 255, 0.05);
    border-radius: 3px;
    overflow: hidden;
  }

  .bar-fill {
    height: 100%;
    border-radius: 3px;
    transition: width 0.8s cubic-bezier(0.4, 0, 0.2, 1);
  }

  /* 品詞バッジクラス */
  .pos-tag-label {
    font-size: 0.75rem;
    font-weight: 600;
    padding: 0.1rem 0.4rem;
    border-radius: 4px;
    color: #000;
  }

  .pos-tag-label.noun { background: var(--pos-noun); }
  .pos-tag-label.verb { background: var(--pos-verb); }
  .pos-tag-label.adj { background: var(--pos-adj); }
  .pos-tag-label.particle { background: var(--pos-particle); }
  .pos-tag-label.aux { background: var(--pos-aux); }
  .pos-tag-label.adv { background: var(--pos-adv); }
  .pos-tag-label.symbol { background: var(--pos-symbol); color: #fff; }
  .pos-tag-label.num { background: var(--pos-num); }
  .pos-tag-label.foreign { background: var(--pos-foreign); }
  .pos-tag-label.unknown { background: var(--pos-unknown); color: #fff; }
  .pos-tag-label.default { background: var(--pos-default); }

  .bar-fill.noun { background: var(--pos-noun); }
  .bar-fill.verb { background: var(--pos-verb); }
  .bar-fill.adj { background: var(--pos-adj); }
  .bar-fill.particle { background: var(--pos-particle); }
  .bar-fill.aux { background: var(--pos-aux); }
  .bar-fill.adv { background: var(--pos-adv); }
  .bar-fill.symbol { background: var(--pos-symbol); }
  .bar-fill.num { background: var(--pos-num); }
  .bar-fill.foreign { background: var(--pos-foreign); }
  .bar-fill.unknown { background: var(--pos-unknown); }
  .bar-fill.default { background: var(--pos-default); }

  /* テーブル */
  .table-container {
    max-height: 400px;
    overflow-y: auto;
    border: 1px solid var(--border-color);
    border-radius: 12px;
    background: rgba(10, 15, 29, 0.3);
  }

  table {
    width: 100%;
    border-collapse: collapse;
    font-size: 0.95rem;
    text-align: left;
  }

  th {
    padding: 0.75rem 1rem;
    background: rgba(255, 255, 255, 0.02);
    border-bottom: 1px solid var(--border-color);
    font-weight: 600;
    color: var(--text-secondary);
    font-size: 0.75rem;
    text-transform: uppercase;
    letter-spacing: 0.05em;
  }

  td {
    padding: 0.75rem 1rem;
    border-bottom: 1px solid var(--border-color);
    color: var(--text-primary);
  }

  tr:last-child td {
    border-bottom: none;
  }

  tr:hover td {
    background: rgba(255, 255, 255, 0.015);
  }

  .num {
    color: var(--text-secondary);
    opacity: 0.5;
    font-size: 0.8rem;
  }

  .surface {
    font-weight: 600;
    letter-spacing: 0.02em;
  }

  .pos-badge {
    display: inline-block;
    font-size: 0.75rem;
    font-weight: 600;
    padding: 0.2rem 0.6rem;
    border-radius: 6px;
    color: #111827;
  }

  .pos-badge.noun { background: var(--pos-noun); }
  .pos-badge.verb { background: var(--pos-verb); }
  .pos-badge.adj { background: var(--pos-adj); }
  .pos-badge.particle { background: var(--pos-particle); }
  .pos-badge.aux { background: var(--pos-aux); }
  .pos-badge.adv { background: var(--pos-adv); }
  .pos-badge.symbol { background: var(--pos-symbol); color: #fff; }
  .pos-badge.num { background: var(--pos-num); }
  .pos-badge.foreign { background: var(--pos-foreign); }
  .pos-badge.unknown { background: var(--pos-unknown); color: #fff; }
  .pos-badge.default { background: var(--pos-default); }

  /* メッセージ表示 */
  .message {
    margin-top: 1rem;
    padding: 0.8rem 1rem;
    border-radius: 8px;
    font-size: 0.9rem;
  }

  .message.error {
    background: rgba(239, 68, 68, 0.1);
    border: 1px solid rgba(239, 68, 68, 0.2);
    color: #fca5a5;
  }

  .message.success {
    background: rgba(16, 185, 129, 0.1);
    border: 1px solid rgba(16, 185, 129, 0.2);
    color: #6ee7b7;
  }

  /* 学習タブ */
  .train-section {
    display: flex;
    flex-direction: column;
    gap: 1.25rem;
  }

  .train-header {
    margin-bottom: 0.5rem;
  }

  .train-header h3 {
    font-size: 1.1rem;
    color: #fff;
    margin-bottom: 0.25rem;
  }

  .description {
    font-size: 0.85rem;
    color: var(--text-secondary);
    line-height: 1.5;
  }

  /* 辞書ブラウザ */
  .dict-section {
    display: flex;
    flex-direction: column;
    gap: 1.25rem;
  }

  .dict-toolbar {
    display: flex;
    gap: 1rem;
    align-items: center;
  }

  .search-box {
    position: relative;
    flex: 1;
  }

  .search-box input {
    width: 100%;
    padding: 0.6rem 1rem 0.6rem 2.25rem;
    background: rgba(10, 15, 29, 0.5);
    border: 1px solid var(--border-color);
    border-radius: 8px;
    color: #fff;
    outline: none;
    font-size: 0.9rem;
    transition: all 0.2s;
  }

  .search-box input:focus {
    border-color: var(--primary-color);
  }

  .search-icon {
    position: absolute;
    left: 0.75rem;
    top: 50%;
    transform: translateY(-50%);
    color: var(--text-secondary);
    opacity: 0.6;
    pointer-events: none;
  }

  .filter-group select {
    padding: 0.6rem 2rem 0.6rem 1rem;
    background: rgba(10, 15, 29, 0.5);
    border: 1px solid var(--border-color);
    border-radius: 8px;
    color: #fff;
    outline: none;
    font-size: 0.9rem;
    cursor: pointer;
    appearance: none;
    background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' fill='none' viewBox='0 0 24 24' stroke='%239ca3af'%3E%3Cpath stroke-linecap='round' stroke-linejoin='round' stroke-width='2' d='m19 9-7 7-7-7'/%3E%3C/svg%3E");
    background-repeat: no-repeat;
    background-position: right 0.75rem center;
    background-size: 1rem;
  }

  .dict-table-container {
    max-height: 450px;
    overflow-y: auto;
    border: 1px solid var(--border-color);
    border-radius: 12px;
    background: rgba(10, 15, 29, 0.3);
  }

  .sortable {
    cursor: pointer;
    user-select: none;
    transition: color 0.15s;
  }

  .sortable:hover {
    color: #fff;
  }

  .surface-dict {
    font-weight: 700;
    font-size: 1rem;
  }

  .text-right {
    text-align: right;
  }

  .freq-val {
    font-family: monospace;
    font-weight: 600;
    color: #818cf8;
  }

  .empty-state {
    text-align: center;
    padding: 3rem;
    color: var(--text-secondary);
    font-style: italic;
  }

  .dict-footer {
    font-size: 0.8rem;
    color: var(--text-secondary);
    text-align: right;
  }

  /* アニメーション用クラス */
  .fade-in {
    animation: fadeIn 0.4s ease forwards;
  }

  .fade-in-row {
    animation: fadeInRow 0.3s ease forwards;
  }

  @keyframes fadeIn {
    from { opacity: 0; transform: translateY(8px); }
    to { opacity: 1; transform: translateY(0); }
  }

  @keyframes fadeInRow {
    from { opacity: 0; }
    to { opacity: 1; }
  }

  /* ローディングスピナー */
  .spinner {
    display: inline-block;
    width: 1rem;
    height: 1rem;
    border: 2px solid rgba(255, 255, 255, 0.3);
    border-radius: 50%;
    border-top-color: #fff;
    animation: spin 0.8s linear infinite;
    margin-right: 0.5rem;
  }

  @keyframes spin {
    to { transform: rotate(360deg); }
  }

  /* モーダルのスタイル */
  .modal-overlay {
    position: fixed;
    top: 0;
    left: 0;
    width: 100%;
    height: 100%;
    background: rgba(10, 15, 29, 0.7);
    backdrop-filter: blur(8px);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 1000;
  }

  .modal-card {
    width: 100%;
    max-width: 440px;
    border: 1px solid var(--border-color);
    padding: 1.75rem;
    box-shadow: 0 10px 40px rgba(0, 0, 0, 0.5);
  }

  .modal-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 1.5rem;
  }

  .modal-header h2 {
    font-size: 1.25rem;
    font-weight: 700;
    color: #fff;
  }

  .close-btn {
    background: none;
    border: none;
    color: var(--text-secondary);
    font-size: 1.5rem;
    cursor: pointer;
    transition: color 0.15s;
  }

  .close-btn:hover {
    color: #fff;
  }

  .form-group {
    display: flex;
    flex-direction: column;
    gap: 0.4rem;
    margin-bottom: 1.25rem;
  }

  .form-group label {
    font-size: 0.8rem;
    font-weight: 600;
    color: var(--text-secondary);
  }

  .form-group input, .form-group select {
    padding: 0.6rem 0.8rem;
    background: rgba(10, 15, 29, 0.6);
    border: 1px solid var(--border-color);
    border-radius: 8px;
    color: #fff;
    outline: none;
    font-size: 0.95rem;
    transition: all 0.2s;
  }

  .form-group input:focus, .form-group select:focus {
    border-color: var(--primary-color);
  }

  .modal-footer {
    display: flex;
    justify-content: flex-end;
    gap: 0.75rem;
    margin-top: 1.5rem;
  }

  /* 行アクション */
  .row-actions {
    display: flex;
    justify-content: center;
    gap: 0.5rem;
  }

  .action-btn {
    background: rgba(255, 255, 255, 0.03);
    border: 1px solid var(--border-color);
    border-radius: 6px;
    width: 28px;
    height: 28px;
    display: flex;
    align-items: center;
    justify-content: center;
    cursor: pointer;
    color: var(--text-secondary);
    transition: all 0.15s;
  }

  .action-btn:hover {
    color: #fff;
    background: rgba(255, 255, 255, 0.08);
  }

  .action-btn.edit:hover {
    border-color: var(--pos-noun);
    color: var(--pos-noun);
  }

  .action-btn.delete:hover {
    border-color: var(--pos-unknown);
    color: var(--pos-unknown);
  }
</style>
