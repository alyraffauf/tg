_: {
  perSystem = _: {
    treefmt.config = {
      settings.global.excludes = [
        "internal/tangledlex/cbor_gen.go"
        "internal/tangledlex/issuecomment.go"
        "internal/tangledlex/issuestate.go"
        "internal/tangledlex/pullcomment.go"
        "internal/tangledlex/pullstatus.go"
        "internal/tangledlex/repogetRepo.go"
        "internal/tangledlex/repoissue.go"
        "internal/tangledlex/repolistIssues.go"
        "internal/tangledlex/repolistPulls.go"
        "internal/tangledlex/repolistRepos.go"
        "internal/tangledlex/repopull.go"
        "internal/tangledlex/tangledpublicKey.go"
        "internal/tangledlex/tangledrepo.go"
        "internal/tangledlex/tangledstring.go"
      ];
      programs = {
        alejandra.enable = true;
        deadnix.enable = true;
        gofmt.enable = true;
        prettier.enable = true;
        shellcheck.enable = true;
        shfmt.enable = true;
        statix.enable = true;
      };
    };
  };
}
