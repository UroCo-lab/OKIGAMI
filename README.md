# OKIGAMI（オキガミ）

**使わなくなったKindleを、スマホで書き換えられる電子ペーパー掲示板に。**

脱獄済みKindle上で動く、掲示専用アプリです。本体のQRを読み、同じWi-Fiのスマホ・タブレット・PCから文字や写真を編集して送信します。日常の利用にPCサーバー・クラウド・スマホ専用アプリ・Amazonアカウントは不要です。

An on-device, standalone e-paper noticeboard for a jailbroken Kindle. Scan its QR code and edit text or photos from a browser on the same Wi-Fi. No external server, cloud account, or phone app is required.

[最新版をダウンロード](https://github.com/UroCo-lab/OKIGAMI/releases/latest) · [使い方](docs/usage.md) · [ビルド](docs/build.md) · [検証状況](docs/verification.md) · [ライセンスと出典](THIRD_PARTY_NOTICES.md)

## できること

- 文字／画像を切り替え、プレビューを見ながら送信。縦・横表示に対応。
- 画像全体の表示／画面いっぱいの切り抜き。指での位置調整とスライダー・＋／−。
- 標準・ゴシック・明朝・丸ゴシック。本文と写真の文字に、それぞれ太字・文字サイズ・行間・縁取り・白黒反転。
- 写真の文字は複数行入力でき、余白帯に置く／背景なしで写真に重ねるを選択。
- 3×3の1〜9で文字位置を指定。文字はエリア境界で切り取らず、隣のエリアにも跨いで描画。
- 余白帯の上下幅、写真の階調表現を調整。
- 自動スリープをKindle標準／5秒／30秒／1分／3分／5分から選択。
- 起きている間だけQRとWi-Fi・バッテリー・終了アイコンを表示。スリープ中は掲示内容だけを残す。
- Wi-Fi未接続でも前回保存した掲示画像で起動。接続すると編集用QRを自動更新。

## 対応・必要なもの

**実機確認済み：Kindle Paperwhite 4（第10世代）、FW 5.18.1.1.1、1072×1448、ARMv7 KindleHF。**

- 脱獄済みで、ライブラリから `.sh` Scriptletを開ける環境が必要です（SH_Integration対応のhotfix等）。この配布物に脱獄ツールは含みません。
- 書き換えるときはKindleと操作端末が同じWi-Fiに接続され、端末同士の通信が許可されていること。保存済みの掲示表示にはWi-Fiは不要です。スマホのテザリングでも利用できます。
- 他のモデル・ファームウェアは未検証です。現行の起動スクリプトはPW4の解像度を前提にしています。

## インストール

1. [Releases](https://github.com/UroCo-lab/OKIGAMI/releases/latest)から **`OKIGAMI-0.6.1-kindlehf.zip`** をダウンロードして展開します。GitHubの「Source code」ZIPは開発用です。
2. KindleをUSBでPCにつなぎます。
3. 展開した **`Kindle-USB` の中身**をKindleドライブの直下へコピーします。`documents` と `kindle-board` がドライブ直下に並ぶ状態です。
4. 安全に取り外し、USBを抜いてライブラリの **OKIGAMI** を開きます。未接続なら前回の掲示画像を表示します。
5. 書き換えるときはKindleをWi-Fiへ接続し、本体左下のQRをスマホで読んで「Kindleに表示する」を押します。

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
   └─ ss/
```

内部フォルダー名 `kindle-board` は旧版から掲示画像と設定を引き継ぐため、そのまま使っています。フォルダー名を変更しないでください。

## 編集とスリープ

電源ボタンでスリープすると、QR・アイコンが消えて掲示内容だけになります。残像を軽減するため、スリープ移行時は黒→白→掲示画像と全画面を更新します。スリープ中はWi-Fi受信できない場合があるため、編集するときは**本体の電源ボタンで起こします**。開いたブラウザは自動で再接続します。

起動し直すと接続鍵が変わるため、新しいQRを読み直してください。追加書体は初回の選択時にKindleから数MBのフォントを読み込み、以後ブラウザにキャッシュします。フォント読込中は送信を無効にし、編集中としてスリープ時間を延長します。

詳しい操作・更新・削除は[使い方](docs/usage.md)を参照してください。

## 接続とデータ

Kindle自身がローカルHTTPサーバー（TCP 8080）を動かします。QRのランダム鍵で書き換えを認証し、画像・文字・フォントを外部サービスへ送信しません。HTTP通信なので、信頼できる同一ネットワーク内で使う設計です。接続URLを知る同じネットワーク内の人は編集できます。

公開リポジトリ・配布ZIPに実際の接続鍵、個人の掲示画像、動作ログは含めていません。

## 開発・ライセンス

Go製の受信・X11描画アプリと、ブラウザのCanvas編集画面で構成しています。[ビルド方法](docs/build.md)と[実機確認の範囲](docs/verification.md)を公開しています。

自作部分は **[MIT-0](LICENSE)** です。改変・再配布・商用利用ができ、著作権表示も必須ではありません。フォント・Go依存ライブラリ・表示補助ツールは、それぞれの元のライセンスに従います。[THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md)を参照してください。同梱GPLバイナリに対応するソース一式も同じReleaseから配布します。

KindleはAmazonの商標です。本プロジェクトはAmazonの公式製品ではありません。
