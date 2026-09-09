# `git-dirstat` コードレビュー報告書

本ドキュメントは、[`docs/git-dirstat-spec.md`](git-dirstat-spec.md) に基づき実施された、`git-dirstat` CLI ツール（v0.1.0）のコードレビュー報告書です。

---

## 1. 仕様書への準拠性確認

仕様書で定義された各要件に対する実装状況と適合性の検証結果です。

| 仕様項目 | 判定 | 検証内容と実装箇所の詳細 |
| :--- | :---: | :--- |
| **セクション 1 & 2: 技術スタック** | **適合** | `go.mod` で Go >= 1.25 を設定。Pure Go 実装の `go-git/v5`、CLI フレームワーク `cobra`、パスパターンマッチ `doublestar/v4` を採用し、CGO に依存しない設計を実現。 |
| **セクション 3: コマンド構文・引数解決** | **適合** | [`cmd/args.go`](../cmd/args.go) にて、0引数（HEAD vs 作業ツリー）、1引数（`<commit>` vs HEAD）、2引数（`<c1> <c2>`）、2点レンジ（`..`）、3点マージベース（`...`）および片側省略記法（`..c`, `c..`）を正確に解決。`-t` と `-- <path>` の競合時は Exit Code 2 で終了。 |
| **セクション 4: オプション・フラグ** | **適合** | [`cmd/root.go`](../cmd/root.go) にて、`--target`, `--depth` (>=1 検証), `--sort` (files, added, deleted, net, path), `--reverse`, `--format` (table, json, csv, tsv), `--exclude`, `--no-color`, `--version`, `--help` を定義。 |
| **セクション 5.1: Git 差分抽出** | **適合** | [`pkg/gitutil`](../pkg/gitutil) にて `.git` 探索、空リポジトリ検知、リビジョン解決、共通祖先（`MergeBase`）算出を実装。コミット間差分と作業ツリー（Staged + Unstaged 合算、Untracked 除外）差分の抽出に対応。バイナリファイルは `Files: 1, Added: 0, Deleted: 0, Net: 0` として計上。移動/名前変更はリネーム検出を行わず削除＋追加として集計。 |
| **セクション 5.2 & 5.3: 集計・深度・フィルタ** | **適合** | [`pkg/aggregator`](../pkg/aggregator) にてターゲット内外判定、`doublestar` による除外処理、深度別グループ化と直下ファイルバケット化（`is_root: true` / `<target>/ (root files)`）、重複なしファイル数と増減行数の集計を実装。 |
| **セクション 5.4: ソート規則** | **適合** | 各フィールドでのソート、`net` の符号付き数値ソート、同値時の `path` 昇順タイブレーク、`--reverse` 反転処理を実装。 |
| **セクション 6: 出力フォーマット** | **適合** | [`pkg/formatter`](../pkg/formatter) にて Table（列揃え、ANSI カラー/TTY判定、TOTAL行）、JSON（スキーマ準拠、0件時の空配列 `[]`）、CSV / TSV（ヘッダー付き・TOTAL行なし）の4形式を実装。 |
| **セクション 7: 終了コード** | **適合** | [`main.go`](../main.go) および [`pkg/model/errors.go`](../pkg/model/errors.go) にて、正常 `0`、Git/実行時エラー `1`、ユーザー入力エラー `2` を厳格に制御し、エラーは `stderr` へ出力。 |
| **セクション 8: ビルド・配布** | **適合** | [`Makefile`](../Makefile) にて、Linux (amd64, arm64), macOS (darwin amd64, arm64), Windows (amd64) 向けの静的リンクバイナリビルド（`CGO_ENABLED=0`, `-trimpath -ldflags="-s -w"`）ターゲットを定義。 |

---

## 2. 発見された問題と対応

### ファイルディスクリプタのリソースリーク

- **検出箇所**: [`pkg/gitutil/diff.go`](../pkg/gitutil/diff.go) の `DiffWorkingTree` 内（作業ツリーのバイナリファイル比較処理）
- **事象**:
  ```go
  diskBytes, _ := os.ReadFile(fullDiskPath)
  headReader, _ := headFile.Reader()
  headBytes, _ := io.ReadAll(headReader)
  // headReader.Close() が呼ばれていなかった
  ```
  `headFile.Reader()` で生成された `io.ReadCloser` が明示的にクローズされておらず、ファイル数の多いリポジトリでファイルディスクリプタが枯渇する恐れがありました。
- **修正内容**:
  読み込み後に即座に `_ = headReader.Close()` を呼び出すよう修正し、リソースリークを解消しました（コミット: `8e1d68f`）。

---

## 3. テスト品質・カバレッジ評価

各パッケージにおいて、境界値や異常系を含む単体テストおよび E2E 統合テストが整備されています。

- **`pkg/aggregator`**: カバレッジ **90.5%**（深度スライス、ターゲット内外判定、除外パターン、ソート規則）
- **`cmd`**: カバレッジ **75.6%**（コマンド構文、フラグバリデーション、バージョン/ヘルプ表示）
- **`pkg/gitutil`**: カバレッジ **72.6%**（リポジトリ探索、リビジョン解決、共通祖先未存在エラー、バイナリ判定、作業ツリー差分）
- **`pkg/formatter`**: カバレッジ **70.1%**（各フォーマット出力、TTY/カラー判定、0件時出力）
- **`test` (E2E)**: 擬似リポジトリを用いた実シナリオ総合テスト（マージベース比較、作業ツリー Staged/Unstaged 比較、Untracked 除外、各種終了コード検証）

全テストが正常に PASS することを確認済みです。
