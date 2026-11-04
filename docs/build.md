# ビルド

## 必要なもの

- Go 1.24以上。使用ツールチェーンは **Go 1.27.3** です。
- PowerShell 5.1以降（配布パッケージ作成用）。
- Node.js（ブラウザ側ロジックの検証用）。
- Python 3とPOSIXシェル（オフライン起動スクリプトの模擬テスト用）。

GoアプリはCGOを使いません。KindleHF向けは `GOOS=linux GOARCH=arm GOARM=7 CGO_ENABLED=0` でビルドします。外部ツールは `vendor/custom-screensaver` の固定済みバイナリを使います。そのソース・再ビルド手順は[SOURCES.md](../SOURCES.md)を参照してください。

## 検査とパッケージ作成

```powershell
go -C src test ./...
go -C src vet ./...
node verify-crop.cjs
node verify-connection.cjs
node verify-caption-anchor.cjs
python scripts/test-offline-launcher.py
./build.ps1
```

成果物は `dist/OKIGAMI-0.6.1` に作られます。`Kindle-USB` が本体へコピーする中身、`okigami-preview.exe` がWindowsの確認用サーバーです。出力先の `Kindle-USB/kindle-board/state` に実データがある場合はパッケージ作成を中止します。

このスクリプトはKindleドライブへ自動インストールしません。個別端末用のインストール記録や接続鍵はソース・配布物に含めません。

## PCプレビュー

```powershell
./dist/OKIGAMI-0.6.1/okigami-preview.exe -listen 127.0.0.1:18080 -ip 127.0.0.1 -data ./preview-state
```

`preview-state/connection.txt` のURLをブラウザで開きます。フック指定なしのPCプレビューでは実機に送信せず、自動スリープもしません。終了は実行中のターミナルでCtrl+Cです。

## 構成

- `src/`：GoのHTTP受信・X11描画・状態管理、組み込みブラウザ画面とフォント。
- `device/`：ライブラリ起動Scriptletと端末上の表示・プロセス管理。
- `vendor/custom-screensaver/`：表示補助部品の固定版、ライセンス、ビルド由来情報。
- `verify-*.cjs`：切り抜き・再接続・文字位置の検証。
- `scripts/test-offline-launcher.py`：Wi-Fi未接続時の起動スクリプトを模擬環境で検証。

編集中の文字や写真はブラウザで画像化し、画面解像度のPNGとして送ります。`state/content.png` は掲示画像の原本、`state/current.png` は起きている間のQR・アイコン付き画像、`state/sleep/board.png` はスリープ用の原本画像です。実行時データをリポジトリや配布ZIPへ入れないでください。

## 起動ラベルの画像

`src/assets/loading-label.png` は起動中の表示用ラスタ画像です。端末の日本語フォントに依存せず起動状態を表示するために使います。編集用のBIZ UD／Zen Maruフォントとは別の画像で、フォントファイルの埋め込みではありません。

`src/assets/offline-label.png` は保存画像がない未接続時の案内です。Pillowをインストールした環境で `python scripts/make-offline-label.py` を実行すると、同梱のBIZ UDゴシックから再生成できます。
