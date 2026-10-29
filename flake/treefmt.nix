_: {
  perSystem = _: {
    treefmt.config = {
      settings.formatter.prettier.excludes = ["mcp/uv.lock"];
      settings.global.excludes = ["internal/tangledlex/cbor_gen.go"];
      programs = {
        alejandra.enable = true;
        deadnix.enable = true;
        gofmt.enable = true;
        prettier.enable = true;
        ruff-format.enable = true;
        shellcheck.enable = true;
        shfmt.enable = true;
        statix.enable = true;
      };
    };
  };
}
