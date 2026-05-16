# Morpho

Go製の日本語形態素解析ライブラリ。外部依存なし、標準ライブラリのみで動作する。

文字種境界でのトークン分割と HMM（隠れマルコフモデル）による品詞推定を組み合わせ、コーパスから学習した統計モデルで解析精度を向上できる。

## アーキテクチャ

```
テキスト
  → tokenizer.Segment     文字種境界でトークン分割
  → viterbi.Decode        HMM で最適品詞列を探索
  → []Morpheme            解析結果
```

モデル未学習時はヒューリスティック（文字種・語尾パターン）にフォールバックする。

学習済みモデルと単語エントリは `dict.json` に JSON で永続化される。

### 品詞タグ

| タグ | 説明 |
|------|------|
| 名詞 | 漢字列など |
| 動詞 | ひらがな動詞語尾で判定 |
| 形容詞 | 〜い・〜く 語尾 |
| 助詞 | は・が・の など固定セット |
| 助動詞 | です・ます など固定セット |
| 副詞 | 上記に当てはまらないひらがな |
| 外来語 | カタカナ・ラテン文字 |
| 数詞 | 数字 |
| 記号 | 句読点など |
| 未知語 | 判定不能 |

## ライブラリとしての使い方

```go
import "github.com/7thCode/morpho"

// 初期化（辞書ファイルが存在しない場合は空の辞書で開始）
analyzer, err := morpho.New("dict.json")

// コーパスで学習（学習後は自動で Save するまで in-memory）
analyzer.Train("東京は日本の首都です。今日は良い天気ですね。")

// 解析
morphemes, err := analyzer.Analyze("今日の東京は良い天気です。")
for _, m := range morphemes {
    fmt.Printf("%s\t%s\n", m.Surface, m.POS)
}

// 辞書の保存
analyzer.Save("dict.json")
```

`Train` を複数回呼ぶと、コーパスのカウントは `Trainer` 内部に**累積**される。`Build()` は毎回全累積カウントを正規化してモデルを生成するため、呼び出すたびに「これまでの全コーパスを合算した」モデルへ更新される。

```go
analyzer.Train("東京は日本の首都です。")   // corpus A でモデル構築
analyzer.Train("今日は良い天気ですね。")   // corpus A+B の累積でモデルを再構築
```

学習をリセットしたい場合は新しい `Analyzer` を作成する。

## コマンド

```bash
# テスト（全体）
go test ./...

# テスト（単一）
go test -run TestAnalyzer ./...

# ビルド
go build ./...

# サンプル実行
go run cmd/example/main.go

# HTTP サーバー起動
go run cmd/server/main.go -port 8765 -dict dict.json
```

## HTTP API（cmd/server）

Electron アプリ等から利用するためのローカル HTTP サーバー。

| メソッド | パス | リクエスト | レスポンス |
|----------|------|-----------|-----------|
| GET | `/health` | — | `{"ok": true}` |
| POST | `/analyze` | `{"text": "..."}` | `{"morphemes": [{...}]}` |
| POST | `/train` | `{"corpus": "..."}` | `{"ok": true}` |

`/train` は学習後に辞書を自動保存する。

## デスクトップアプリ（app/）

Electron + Svelte 製の GUI。Go サーバーを子プロセスとして起動し HTTP で通信する。

```bash
cd app

# 初回セットアップ
npm install
npm run build:go        # bin/server をビルド

# 開発起動
npm run dev             # Vite + Electron を同時起動

# プロダクションビルド
npm run build           # Go バイナリ + Vite ビルド
```

開発時の辞書はプロジェクトルートの `dict.json` に読み書きされる。

## シングルバイナリアプリ（cmd/desktop/）

[Wails v2](https://wails.io/) を使った GUI。Svelte フロントエンドを `go:embed` でバイナリに同梱するため、**配布物は実行ファイル 1 つだけ**。別プロセスや Electron ランタイムは不要。

### 必要なもの

- [Wails CLI](https://wails.io/docs/gettingstarted/installation): `go install github.com/wailsapp/wails/v2/cmd/wails@latest`
- Node.js（フロントエンドのビルドに使用）

### ビルド・起動

```bash
cd cmd/desktop

# 開発モード（Go + Svelte のホットリロード）
wails dev

# プロダクションビルド（シングルバイナリ）
wails build
# → build/bin/Morpho.app (macOS) が生成される
```

### 辞書ファイルのパス

| 実行方法 | 辞書パス |
| -------- | ------- |
| `wails dev` | プロジェクトルートの `dict.json` |
| `wails build`（production） | `~/Library/Application Support/Morpho/dict.json`（macOS） |

### Electron 版との違い

| 項目 | Electron 版（app/） | Wails 版（cmd/desktop/） |
| ---- | ------------------- | ------------------------ |
| 配布形態 | Go バイナリ + Electron | シングルバイナリ |
| プロセス構成 | Go サーバー + Electron | 1 プロセス |
| フロントエンドとの通信 | HTTP（localhost:8765） | Wails バインディング（直接呼び出し） |
