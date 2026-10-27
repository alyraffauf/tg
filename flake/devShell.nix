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
          just
          python314
          uv
        ])
        ++ [
          pkgs.go_1_27
          config.packages.tg
        ]
        ++ lib.attrValues config.treefmt.build.programs;

      shellHook = ''
        echo "👋 Welcome to the tg devShell!"
      '';
    };
  };
}
