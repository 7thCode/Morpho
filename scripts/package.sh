#!/bin/bash
set -e

# ルートディレクトリの絶対パスを取得
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

echo "=== Morpho Wails アプリのパッケージングを開始します ==="

# 1. ビルド
echo "1. Wails アプリをビルド中..."
cd "$ROOT_DIR/cmd/desktop"
wails build -clean

# 2. dist ディレクトリの作成
echo "2. 配布用ディレクトリの準備中..."
DIST_DIR="$ROOT_DIR/dist"
rm -rf "$DIST_DIR"
mkdir -p "$DIST_DIR"

# 3. 成果物のコピー
echo "3. ビルド成果物をコピー中..."
BUILT_APP="$ROOT_DIR/cmd/desktop/build/bin/Morpho.app"

if [ -d "$BUILT_APP" ]; then
    cp -R "$BUILT_APP" "$DIST_DIR/"
    echo "Morpho.app を $DIST_DIR にコピーしました。"
else
    echo "エラー: ビルド成果物 ($BUILT_APP) が見つかりません。"
    exit 1
fi

# 4. ZIP 圧縮
echo "4. アプリケーションを ZIP 圧縮中..."
cd "$DIST_DIR"
zip -r "Morpho_darwin_arm64.zip" "Morpho.app" > /dev/null

echo "=== パッケージングが完了しました ==="
echo "成果物:"
echo " - $DIST_DIR/Morpho.app (アプリパッケージ)"
echo " - $DIST_DIR/Morpho_darwin_arm64.zip (配布用ZIPファイル)"
