# OKIGAMI 導入と改編のプロジェクトガイド

このファイルをAIアシスタントのプロジェクトやチャットに添付し、下の依頼文とGitHub URLまたはダウンロードファイルを渡すと、OKIGAMIの導入・改編を相談できます。別の会話の履歴がなくても使えるよう、入手先、構成、検証方法をまとめています。

このファイルは案内文です。ソースコードや実行ファイルは含みません。実際の取得・ビルド・端末へのコピー・GitHubへの反映には、それぞれの操作ができる環境が必要です。操作できない環境では、利用者が実行する手順と必要なファイルを用意してください。

基準は **OKIGAMI 0.6.1**、文書更新日は **2026年11月7日**です。導入・改編時には取得した版のREADMEと実際のコードを確認してください。

## 最初に送る依頼文

次をコピーし、分かる項目だけ埋めて送ってください。ファイルを使う場合は一緒に添付します。

```text
添付した「OKIGAMI 導入と改編のプロジェクトガイド」を読んで、OKIGAMIを導入してください。
導入後は、私の好みに合わせて改編しやすい状態にしてください。

入手元：GitHub URL、ReleaseのURL、または添付したZIP
使う端末：Kindleの機種とファームウェア（分からなければ調査してください）
作業するPC：OS（分からなければ調査してください）
変えたいこと：未定／文字や写真の見せ方／操作画面／その他
GitHubへの反映先：自分のリポジトリURL、または未作成

まず接続状況と既存ファイルを確認し、可能な作業を進めてください。
導入だけならアプリの仕様は変えず、変更の希望があれば実装してください。
改編用のソース、起動・ビルド手順、変更箇所の案内を作業フォルダーに残してください。
変更後は検証し、私のGitHubリポジトリに反映してください。
実施できたことと、私の操作や実機確認が必要なことを区別して報告してください。
```

## アシスタントが進める順序

1. 指定されたURL・ZIPと、その中のREADME・ライセンス・作業指示を確認する。既存の作業フォルダー、Gitの変更、端末接続を調べ、利用者の編集を保つ。
2. Kindleの機種、ファームウェア、Scriptlet対応、USBドライブを確認する。端末がなければソース取得とPCプレビューの準備まで進める。不明点は調査し、調べられない必須情報だけ利用者に尋ねる。
3. 導入用ZIPと改編用ソースを区別し、作業フォルダーに整理する。配布バイナリだけをソースコードとして扱わない。
4. 実機へ配置する前に、既存のOKIGAMI本体・起動ファイル・保存データをPCへバックアップする。端末上の対象パスを確認し、下記の導入手順に沿って配置する。
5. 希望が未定なら、現在の機能と変更箇所を案内する。希望が決まっていれば小さな単位で改編し、関係する検証とプレビューを行う。
6. 利用者のリポジトリで変更をコミット・プッシュし、反映先URLと検証結果を渡す。認証や端末接続が足りなければ、可能な準備を完了して、残る操作を具体的に示す。

## 入手先とファイルの選び方

- 公式リポジトリ：https://github.com/UroCo-lab/OKIGAMI
- 現在のRelease：https://github.com/UroCo-lab/OKIGAMI/releases/latest
- このガイドの最新版：https://raw.githubusercontent.com/UroCo-lab/OKIGAMI/main/docs/OKIGAMI-PROJECT.md
- 0.6.1のRelease：https://github.com/UroCo-lab/OKIGAMI/releases/tag/v0.6.1
- 0.6.1の導入用ZIP：https://github.com/UroCo-lab/OKIGAMI/releases/download/v0.6.1/OKIGAMI-0.6.1-kindlehf.zip
- 0.6.1の改編用ソースZIP：https://github.com/UroCo-lab/OKIGAMI/archive/refs/tags/v0.6.1.zip
- 同梱外部部品のソース：https://github.com/UroCo-lab/OKIGAMI/releases/download/v0.6.1/OKIGAMI-0.6.1-third-party-sources.zip

**導入用ZIP**には、Kindleへコピーする `Kindle-USB` とWindowsのプレビュー用実行ファイルがあります。**ソースZIP**またはGit cloneは、コードの改編と再ビルドに使います。**外部部品のソースZIP**は表示補助ツール等のソースであり、OKIGAMI本体のソースの代わりにはなりません。

再現可能な基準から始める場合は `v0.6.1` を選びます。基準コミットは `0a285a715dbe064ee624a251c0cc829f13537805` です。`main` の後続変更も利用する場合は、取得時のコミットを作業記録に残します。

Gitで改編を始める例です。

```sh
git clone https://github.com/UroCo-lab/OKIGAMI.git
cd OKIGAMI
git switch -c customize/my-kindle v0.6.1
```

ブランチ名は任意です。ZIPを使う場合は展開先を確認して作業し、GitHubに反映するときはGit管理を初期化するか、利用者のリポジトリに取り込んでください。

## OKIGAMIの役割と対応範囲

OKIGAMIは、脱獄済みKindleを文字・写真の掲示板にするアプリです。Kindle自身がHTTPサーバーを動かし、同じWi-Fiのスマホ・タブレット・PCからQR経由で編集します。通常の利用にクラウドやPCサーバーは不要です。

0.6.1の実機確認は **Kindle Paperwhite 4 第10世代、FW 5.18.1.1.1、1072×1448、ARMv7 KindleHF** で行われています。他機種・他ファームウェアへの適合は個別に確認してください。ライブラリから `.sh` Scriptletを起動できる環境が必要で、この配布物に脱獄ツールは含みません。

文字・写真の縦横表示、切り抜き、複数書体、太字、文字サイズ、行間、縁取り、白黒反転、写真の余白帯・文字重ね、9か所の文字位置、自動スリープを設定できます。保存済みの掲示画像はWi-Fi未接続でも起動時に復元します。編集時は本体を起こして同じWi-Fiにつなぎます。

## Kindleへ導入する手順

1. 導入用ZIPを展開し、`Kindle-USB` の存在を確認する。
2. USB接続したKindleのドライブを特定する。既存の `kindle-board`、`documents/OKIGAMI.sh`、`documents/OKIGAMI Stop.sh` があればバックアップする。
3. `Kindle-USB` **の中身**をKindleドライブ直下へコピーする。更新時は既存の `kindle-board/state` を保持する。
4. 安全に取り外してUSBを抜き、ライブラリから「OKIGAMI」を起動する。
5. 同じWi-Fiの操作端末で本体のQRを読み、文字や写真を送る。再起動後は新しいQRを使う。

配置後は次の形になります。他の書籍やアプリのファイルは操作対象に含めません。

```text
Kindleドライブ/
├─ documents/
│  ├─ OKIGAMI.sh
│  └─ OKIGAMI Stop.sh
└─ kindle-board/
   ├─ bin/kindle-board
   ├─ run.sh
   ├─ display.sh
   ├─ process.sh
   ├─ licenses/
   ├─ ss/
   └─ state/                 保存画像と設定
```

内部フォルダー名 `kindle-board` は保存データの引き継ぎに使います。端末の接続URL・鍵、実際の掲示画像、ログ、バックアップは公開物へ入れないでください。通信はローカルHTTPで、信頼できる同一ネットワーク内で使う設計です。

## 好みに合わせて変える場所

まずブラウザの設定で希望を実現できるか確認し、設定の範囲を超える場合にコードを変更します。

| 変えたい内容 | 主なファイル |
|---|---|
| 操作画面、文言、文字・写真の描画、切り抜き、文字位置 | `src/web/index.html` |
| 書体の配信とフォント | `src/fonts.go`、`src/web/fonts/` |
| HTTP API、アップロード、起動引数、基本解像度 | `src/main.go` |
| 保存画像、向き、オフライン復元 | `src/frame.go`、`src/startup.go`、`src/assets/` |
| QRと接続状態 | `src/connection_qr.go`、`src/web/index.html` |
| バッテリー等のアイコンとタップ操作 | `src/toolbar.go` |
| 自動スリープと復帰 | `src/auto_sleep.go`、`src/power.go`、`src/sleep_refresh.go` |
| Kindle上での起動・終了・表示連携 | `device/`、`src/native.go`、`src/ximage.go` |
| 配布パッケージの作成 | `build.ps1` |

画面のHTML・JavaScript・フォント等はGo実行ファイルに埋め込まれます。ソースを変えたら再ビルドが必要です。機種対応では解像度の値だけでなく、起動スクリプト、描画・タッチ・スリープ動作も確認してください。

改編を始めた作業フォルダーには、利用者の希望、変更したファイル、採用版とコミット、起動方法、検証結果、未確認事項を `CUSTOMIZATION.md` などに残します。次の会話でも、このガイドとその記録を渡すことで作業を引き継げます。

## ビルドとPCでの確認

基準版の必要環境はGo 1.24以上、パッケージ作成用のPowerShell 5.1以降、ブラウザ側検証用のNode.jsです。オフライン起動の模擬検証にはPython 3とPOSIXシェルも使います。既存CIのGo指定は1.27.3です。

リポジトリ直下で実行します。必要なツールの有無を先に調べてください。

```powershell
go -C src test ./...
go -C src vet ./...
node verify-crop.cjs
node verify-connection.cjs
node verify-caption-anchor.cjs
python scripts/test-offline-launcher.py
./build.ps1
```

0.6.1の成果物は `dist/OKIGAMI-0.6.1/` です。`build.ps1` はPC用とKindle用をビルドしますが、端末へ自動コピーはしません。別の版では `VERSION` とビルドスクリプトに従って出力先を確認してください。

Windowsのプレビューは次のように起動し、`preview-state/connection.txt` のURLをブラウザで開きます。終了はターミナルでCtrl+Cです。

```powershell
./dist/OKIGAMI-0.6.1/okigami-preview.exe -listen 127.0.0.1:18080 -ip 127.0.0.1 -data ./preview-state
```

PCプレビューはKindleへ送信しません。文字・写真の編集や表示を確認した後、実機ではQR接続、縦横表示、終了・再起動、スリープ・復帰、Wi-Fi未接続時の保存画像復元を確認します。PCでのPASSと実機での確認を区別して記録してください。

## 改編をGitHubへ反映する

利用者自身のForkまたは指定された書き込み可能なリポジトリを使います。配布元の `UroCo-lab/OKIGAMI` へ直接書き込むことを前提にしないでください。clone直後の `origin` は配布元なので、利用者のリポジトリを追加するかremoteを整理します。

1. 反映先URL、ブランチ、認証と権限を確認する。
2. 差分と検証結果を確認し、変更したソースと必要な文書を選んでコミットする。
3. 指定先へプッシュし、GitHub上のコミットを確認する。PR運用ならPRを作成する。
4. 配布も依頼された場合は、版番号を整え、ビルドした導入用ZIPとSHA-256、必要なライセンス・外部部品のソースをReleaseで提供する。

`state/`、`preview-state/`、個人の画像、接続鍵、認証情報、端末バックアップはコミットしません。自作部分のライセンスはMIT-0です。フォントや表示補助部品等には別のライセンスがあるため、改変・再配布時は `LICENSE`、`THIRD_PARTY_NOTICES.md`、`SOURCES.md` を確認してください。

## 詳しい原典

- README：https://github.com/UroCo-lab/OKIGAMI/blob/main/README.md
- 操作・更新・削除：https://github.com/UroCo-lab/OKIGAMI/blob/main/docs/usage.md
- ビルド：https://github.com/UroCo-lab/OKIGAMI/blob/main/docs/build.md
- 実機確認の範囲：https://github.com/UroCo-lab/OKIGAMI/blob/main/docs/verification.md
- 外部部品と再ビルド：https://github.com/UroCo-lab/OKIGAMI/blob/main/SOURCES.md
- ライセンスと出典：https://github.com/UroCo-lab/OKIGAMI/blob/main/THIRD_PARTY_NOTICES.md
