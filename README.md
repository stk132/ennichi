# ennichi

fireworq TUI management console application

## これは何？

fireworq管理用のTUIアプリケーションです。

## 何ができるの？

現在以下の機能を実装しています。

- キュー一覧の表示 (queue list)
- キュー設定の表示 (queue info)
- キューのジョブルーティング設定の表示 (job categories)
- キューの新規作成

## 使い方

fireworqのエンドポイントを指定して起動します。

```shell
ennichi --host=http://yourfireworqhost
```

以下のキーバインドが設定されています。

- q: キュー一覧にフォーカス
- l: ログウィンドウにフォーカス
- n: 新規キュー作成フォームに遷移

キュー一覧にフォーカスしている際のキーバインド

- r: キュー一覧を更新
- enter: 選択中のキューの設定情報とジョブルーティング情報がそれぞれqueue infoとjob categoriesに表示
