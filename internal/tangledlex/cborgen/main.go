package main

import (
	lex "github.com/alyraffauf/tg/internal/tangledlex"
	cbg "github.com/whyrusleeping/cbor-gen"
)

func main() {
	g := cbg.Gen{MaxStringLength: 1_000_000}
	if err := g.WriteMapEncodersToFile("cbor_gen.go", "tangledlex", lex.ActorDefs_ProfileViewBasic{}, lex.ActorDefs_ProfileViewDetailed{}, lex.ActorDefs_ViewerState{}, lex.EmbedCommit{}, lex.EmbedCommit_View{}, lex.MarkupEmbedImage{}, lex.MarkupEmbedVideo{}, lex.FeedComment{}, lex.GitOid{}, lex.GitRefs{}, lex.GraphAttestation{}, lex.KeySshKey{}, lex.LabelDefs_LabelToken{}, lex.MarkupMarkdown{}, lex.MarkupMarkdown_Inline{}, lex.RepoDeclaration{}, lex.RepoDefs_Language{}, lex.RepoDefs_RepoViewBasic{}, lex.RepoDefs_SourceExternal{}, lex.RepoDefs_SourceTangled{}, lex.RepoDefs_ViewerState{}, lex.RepoManifest{}, lex.RepoManifest_Declaration{}, lex.ReviewPatch{}, lex.ReviewPatch_Source{}, lex.SnippetString{}, lex.SnippetString_TextFile{}, lex.TrackDefs_TicketViewDetailed{}, lex.TrackDefs_TicketViewDetailed_Body{}, lex.TrackDefs_TicketViewDetailed_TangledLabels_Elem{}, lex.TrackDefs_TicketViewDetailed_Title{}, lex.TrackTicket{}); err != nil {
		panic(err)
	}
}
