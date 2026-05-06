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

`Train` を呼ぶたびにそのセッションのコーパスのみでモデルを再構築する（累積ではなく上書き）。継続学習が必要な場合はコーパスをまとめて渡す。

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
