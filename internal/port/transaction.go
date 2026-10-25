package port

import "context"

// GrantRepositories はカード配布のトランザクション内で使うリポジトリの組です。
type GrantRepositories struct {
	ProcessedEvents ProcessedEventRepository
	CardPacks       CardPackRepository
	PlayerCards     PlayerCardRepository
}

// GrantTransactionRunner はカード配布を 1 つのトランザクションで実行するインターフェースです。
type GrantTransactionRunner interface {
	// RunInTransaction は fn を 1 つのトランザクションで実行し、fn に渡したリポジトリの変更をまとめて確定します。
	// fn がエラーを返したときは、リポジトリの変更をすべて取り消してそのエラーを返します。
	RunInTransaction(ctx context.Context, fn func(repositories GrantRepositories) error) error
}
