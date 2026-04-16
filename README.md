# mkepub

青空文庫の作品をEPUBファイルに変換するターミナルツールです。

## 機能

- 青空文庫のCSVカタログから作家・作品を検索
- テキストをダウンロードしてEPUBに変換
- タイトル・作者名入りの表紙を自動生成
- ルビ（振り仮名）をEPUB形式で保持
- 縦書きレイアウトのスタイルシート付き
- 作成したEPUBをメール（Kindle等）に送信

## インストール

```sh
go install mkepub@latest
```

またはリポジトリをクローンしてビルド：

```sh
git clone <repo>
cd mkepub
go build -o mkepub .
```

## セットアップ

1. **CSVカタログを取得**

   青空文庫の人物一覧CSVを取得して配置します：

   ```sh
   # デフォルトの配置先
   ~/.local/share/mkepub/list_person_all_extended_utf8.csv
   ```

   CSVは [青空文庫 図書カードCSV](https://www.aozora.gr.jp/index_pages/person_all.html) からダウンロードできます（`list_person_all_extended_utf8.zip`）。

2. **初回起動**

   初回起動時に設定ファイルが自動生成されます：

   ```sh
   mkepub
   ```

   設定ファイルの場所：`~/.config/mkepub/config.toml`

## 使い方

```sh
mkepub
```

詳しい操作方法は [docs/USER_GUIDE.md](docs/USER_GUIDE.md) を参照してください。

## 設定

`~/.config/mkepub/config.toml`

```toml
output_dir = "~/.local/share/epub"
csv_path   = "~/.local/share/mkepub/list_person_all_extended_utf8.csv"

[mail]
smtp_host = "smtp.gmail.com"
smtp_port = 587
from      = ""
password  = ""
to        = ""
```

メール送信を使わない場合は `[mail]` セクションの `from` と `to` を空のままにしてください。

## ライセンス

MIT
