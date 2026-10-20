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
          (pkgs.callPackage ../nix/go.nix {})
          config.packages.tg
        ]
        ++ lib.attrValues config.treefmt.build.programs;

      shellHook = ''
        echo "👋 Welcome to the tg devShell!"
      '';
    };
  };
}
