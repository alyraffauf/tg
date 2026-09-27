_: {
  perSystem = {
    config,
    lib,
    pkgs,
    ...
  }: {
    devShells.default = pkgs.mkShell {
      packages =
        (with pkgs; [
          go
          just
          python314
          uv
        ])
        ++ [config.packages.tg]
        ++ lib.attrValues config.treefmt.build.programs;

      env = {
        UV_PYTHON = pkgs.python314.interpreter;
        UV_PYTHON_DOWNLOADS = "never";
      };

      shellHook = ''
        echo "👋 Welcome to the tg devShell!"
      '';
    };
  };
}
