# `chapel` - mp3ファイルのメタデータ編集ツール

## 概要

このツールは、mp3ファイルのメタデータを編集するためのコマンドラインツールです。サポートするのはmp3ファイルのみに限ります。

## メタデータ形式

メタデータはYAMLで表現され、chapelはそれを入出力として使用します。すべてのフィールドはオプショナルで、必要なものだけ記述できます。

```yaml
# 基本メタデータ (ID3v2タグ)
title: "曲名 / ポッドキャストエピソードタイトル"  # TIT2
artist: "アーティスト名 / ポッドキャスター名"  # TPE1
album: "アルバム名 / ポッドキャスト名"  # TALB
albumArtist: "アルバムアーティスト名"  # TPE2 (コンピレーションアルバム用)
date: "2024-08-15"  # TDRC (録音時刻) - ID3v2タイムスタンプフォーマット
track: "3/10"  # TRCK (トラック番号/総数) - ID3v2フォーマット
disc: "1/2"  # TPOS (ディスク番号/総数) - ID3v2フォーマット
genre: "ジャンル / Podcast"  # TCON
comment: "コメント"  # COMM
composer: "作曲者"  # TCOM
publisher: "パブリッシャー"  # TPUB (レーベル名)
bpm: 120  # TBPM (テンポ)

# チャプター情報 (CHAP)
chapters:
  - "0:00 イントロ"
  - "1:30.500 Aメロ"  # ミリ秒精度対応
  - "3:00 サビ"
  - "4:30 ブリッジ"
  - "5:45 アウトロ"

# カバーアート (APIC) - データURIまたはファイルパス
artwork: "data:image/jpeg;base64,/9j/4AAQSkZJRgABAQEA..."  # データURI
# artwork: "/absolute/path/to/cover.jpg"  # 絶対パス
# artwork: "cover.jpg"  # 相対パス
# artwork: "./images/cover.jpg"  # 相対パス

# 歌詞 (USLT)
lyrics: |
  ここに歌詞を
  複数行で
  記述できます
```

### フォーマット仕様

#### 基本メタデータ
- **ID3v2.4標準タグ対応**: 一般的に使用される主要なタグフィールドを網羅
- **UTF-8エンコーディング**: 多バイト文字対応

#### 日付フィールド (date)
- **ID3v2タイムスタンプフォーマット**: ID3v2.4.0-structure仕様準拠
- **ISO 8601サブセット**: UTCタイムスタンプ
- **対応精度**: `2024`, `2024-08`, `2024-08-15`, `2024-08-15T14`, `2024-08-15T14:30`, `2024-08-15T14:30:45`

#### トラック/ディスク番号
- **ID3v2フォーマット**: `"現在/総数"` または `"現在"` のみ
- **例**: `"3/10"` (10曲中3曲目), `"1/2"` (2枚組の1枚目), `"5"` (番号のみ)

#### チャプター
- **WebVTTフォーマット**: `"時間 タイトル"` 形式
- **時間精度**: `M:SS`, `H:MM:SS`, ミリ秒付き `M:SS.mmm`, `H:MM:SS.mmm`
- **例**: `"0:00 Introduction"`, `"1:30.500 Main Topic"`, `"1:02:30.123 Conclusion"`

#### アートワーク
- **データURI形式**: `data:image/jpeg;base64,<base64データ>`
- **ファイルパス形式**:
  - 絶対パス: `/absolute/path/to/image.jpg`
  - 相対パス: `cover.jpg`, `./images/cover.jpg` (作業ディレクトリからの相対)
- **対応形式**: JPEG, PNG, GIF, BMP, WebP

- **すべてのフィールドはオプショナル**: 必要なものだけ記述可能

## 利用方法

### メタデータのダンプ (`dump`)

mp3を指定して、メタデータを標準出力にダンプします。

```console
% chapel dump {{mp3ファイル}} > dump.yaml
```

### メタデータの適用 (`apply`)

標準入力からYAML形式のメタデータを読み込み、指定したmp3ファイルに適用します。適用する前に、適用差分を表示しながらプロンプトで確認を求めます。

```console
% chapel apply {{mp3ファイル}} < dump.yaml
```

### メタデータの直接編集

mp3ファイルを直接指定してメタデータを編集します。

```console
% chapel {{mp3ファイル}}
```

処理としては、一時ファイルにメタデータをダンプし、それを "EDITOR" 環境変数に指定されたエディタで開きます。ファイル編集後、その内容がmp3ファイルに反映されます。`apply` コマンドと同様に、適用前に差分を確認するプロンプトが表示されます。

## 技術仕様

### 内部実装
- **カスタム型**:
  - `Timestamp`: ID3v2.4.0-structure準拠のタイムスタンプ型 (time.Time + Precision)
  - `NumberInSet`: ID3v2のTRCK/TPOSフォーマット対応型 (Current/Total)
  - `Chapter`: WebVTTフォーマット対応のチャプター型 (time.Duration + Title)

- **YAML対応**: 各カスタム型にMarshalYAML/UnmarshalYAMLメソッドを実装
- **ID3v2.4準拠**: TDRC、TRCK、TPOS、CHAPフレームの仕様に完全準拠
- **クロスプラットフォーム**: Windows/macOS/Linux対応

## 利用ライブラリ
- github.com/goccy/go-yaml
    - YAML設定ファイルの読み書きのために利用。YAMLライブラリは色々ありますがこれを使います
- github.com/Songmu/prompter
    - プロンプトの表示と確認のために利用。ユーザに適用前の差分を確認させるために使用します
- github.com/sergi/go-diff/diffmatchpatch
    - 差分の計算と表示のために利用。YAMLファイルとmp3ファイルのメタデータの差分を計算するために使用します
- github.com/bogem/id3v2/v2
    - mp3ファイルのID3v2メタデータの読み書きのために利用
- github.com/tcolgate/mp3
    - MP3ファイルの音声時間長計算のために利用

## 参考
- <https://github.com/Songmu/podbard>
    - ポッドキャスト配信ツール
    - <https://github.com/Songmu/podbard/blob/main/internal/cast/chapter.go> 辺りにチャプターに関する処理が書かれているので参考にして下さい
