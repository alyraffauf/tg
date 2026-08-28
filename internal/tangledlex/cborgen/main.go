package main

import (
	lex "github.com/alyraffauf/tg/internal/tangledlex"
	cbg "github.com/whyrusleeping/cbor-gen"
)

func main() {
	generator := cbg.Gen{MaxStringLength: 1_000_000}
	if err := generator.WriteMapEncodersToFile(
		"cbor_gen.go",
		"tangledlex",
		lex.FeedComment{},
		lex.MarkupMarkdown{},
		lex.PublicKey{},
		lex.Repo{},
		lex.RepoIssue{},
		lex.RepoIssueState{},
		lex.RepoPull{},
		lex.RepoPull_Round{},
		lex.RepoPull_Source{},
		lex.RepoPullStatus{},
		lex.RepoPull_Target{},
		lex.String{},
	); err != nil {
		panic(err)
	}
}
