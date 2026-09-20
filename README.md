# Morpho

[![Go Reference](https://pkg.go.dev/badge/github.com/7thCode/morpho.svg)](https://pkg.go.dev/github.com/7thCode/morpho)
[![Test](https://github.com/7thCode/Morpho/actions/workflows/test.yml/badge.svg)](https://github.com/7thCode/Morpho/actions/workflows/test.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

Go製の日本語形態素解析ライブラリ。外部依存なし、標準ライブラリのみで動作する。

文字種境界でのトークン分割と HMM（隠れマルコフモデル）による品詞推定を組み合わせる。単語の区切りは文字種の境界で決まる（漢字の連続は1語）ため、MeCab のような辞書ベースの分割は行わない。区切りを変えたい語は SaveWord でユーザー辞書に登録する。品詞は、品詞を付けたコーパス（`単語/品詞` 形式）で学習させるとその付け方を学ぶ。通常の文章で学習した場合は文字種ルールによる仮ラベルを学習するため、精度が上がるとは限らない。

## インストール

```bash
go get github.com/7thCode/morpho
```

```go
import "github.com/7thCode/morpho"

analyzer, err := morpho.New("dict.json") // 辞書ファイルが存在しない場合は空の辞書で開始
analyzer.Train("東京は日本の首都です。今日は良い天気ですね。")

morphemes, err := analyzer.Analyze("今日の東京は良い天気です。")
for _, m := range morphemes {
    fmt.Printf("%s\t%s\n", m.Surface, m.POS)
}
```

詳しい使い方は [pkg.go.dev](https://pkg.go.dev/github.com/7thCode/morpho) のパッケージドキュメント、または後述の「ライブラリとしての使い方」を参照。

## アーキテクチャ

```
テキスト
  → tokenizer.SegmentWithLexicon  文字種境界で分割（ユーザー辞書の語は最長一致で保持）
  → hmm.SplitSentences            句点・改行で文に分割
  → viterbi.Decode                文ごとに HMM で最適品詞列を探索
  → []Morpheme                    解析結果
```

モデル未学習時はヒューリスティック（文字種・語尾パターン）にフォールバックする。

学習済みモデル・学習カウント・単語エントリは `dict.json` に JSON で永続化される（保存は一時ファイル経由の置換で、途中で失敗しても既存の辞書は壊れない）。

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
| 数詞 | 数字（コーパスに出現しなくても常にモデルに含まれる） |
| 記号 | 句読点など（同上） |
| 未知語 | 判定不能 |

## ライブラリとしての使い方（補足）

基本的な使い方は冒頭の「インストール」を参照。`Analyzer` は複数の goroutine から同時に使える。

### 学習

`Train` の呼び出しは**積み上がる**。辞書を `Save` して次回 `New` で読み込んだ後の `Train` も、前回までのカウントに追加される（学習カウントは辞書ファイルに保存される）。カウントを持たない古い形式の辞書に対する最初の `Train` は、モデルを作り直す。学習をリセットしたい場合は新しい `Analyzer` を作成する。

```go
analyzer.Train("東京は日本の首都です。")   // corpus A でモデル構築
analyzer.Train("今日は良い天気ですね。")   // corpus A+B の累積でモデルを再構築
```

コーパスは1行ずつ処理される。通常の文章の行は文字種ルールで品詞を仮に付けて学習し、`単語/品詞` をスペースで並べた行はその通りに学習する（品詞は下表の10種類）。

```go
analyzer.Train("猿/名詞 も/助詞 木/名詞 から/助詞 落ちる/動詞 。/記号")
```

学習できる文が1つもないコーパスは `ErrInvalidInput` になる。

### ユーザー辞書

```go
analyzer.SaveWord("東京タワー", "名詞", 1) // 空白を含む語・未知の品詞はエラー
```

`SaveWord` で登録した語は、解析で文字種をまたいでもひとまとまりになり、品詞が登録通りに固定される。`NewInMemory()` で作った Analyzer では、メモリ上にのみ反映される。

### 壊れた辞書ファイル

`New` は辞書が JSON として読めないとき `ErrCorruptDictionary` を返す。`OpenOrRecover` は該当ファイルを `<path>.corrupt-<日時>` に退避して空の辞書で開始する。

> **Note:** リポジトリ直下の `dict.json` は、パブリックドメインの短い文章（『吾輩は猫である』『坊っちゃん』『走れメロス』の冒頭）を `cmd/example` で学習したデモ用データであり、本番品質の辞書ではない。利用時は自前のコーパスで `Train` することを推奨する。

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

ローカル用の HTTP サーバー（Electron 版が子プロセスとして使う）。

| メソッド | パス | リクエスト | レスポンス |
|----------|------|-----------|-----------|
| GET | `/health` | — | `{"ok": true}` |
| POST | `/analyze` | `{"text": "..."}` | `{"morphemes": [{...}]}` |
| POST | `/train` | `{"corpus": "..."}` | `{"ok": true}` |
| GET | `/stats` | — | `{"word_count": ..., "is_trained": ..., "pos_tags": [...]}` |
| GET | `/entries` | — | `[{"surface": ..., "pos": ..., "freq": ...}, ...]` |
| POST/PUT | `/word` | `{"surface": ..., "pos": ..., "freq": ...}` | `{"ok": true}` |
| DELETE | `/word?surface=...` | — | `{"ok": true}` |

`/train` は学習後に辞書を自動保存する。POST/PUT は `Content-Type: application/json` が必要。不正な入力（未知の品詞、空のコーパスなど）は 400、本文が 8 MiB を超えると 413。エラーは `{"error": "..."}` で返る。`/entries` の各要素には、ユーザー登録語なら `"user": true` が付く。

**セキュリティ:** 認証はない。既定では `127.0.0.1` のみで待ち受け（`-host` で変更可）、ブラウザからのアクセスは `-cors-origins` で許可したオリジンだけに限る（既定は Vite 開発サーバーと `file://` ページ）。`-host 0.0.0.0` などでネットワークに公開する場合は、自前でリバースプロキシなどによる認証を用意すること。

## Electron アプリ（app/）— レガシー

Electron + Svelte 製の GUI。Go サーバーを子プロセスとして起動し HTTP で通信する。現在の主力は下記の Wails 版（`cmd/desktop/`）で、こちらは新機能の追加やリリースの対象外。Electron や依存パッケージのバージョンも更新していない。

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
