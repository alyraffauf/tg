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
        ])
        ++ lib.attrValues config.treefmt.build.programs;

      shellHook = ''
        echo "👋 Welcome to the tg devShell!"
      '';
    };
  };
}
