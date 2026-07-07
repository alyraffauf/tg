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
        ])
        ++ lib.attrValues config.treefmt.build.programs;

      shellHook = ''
        echo "👋 Welcome to the tg devShell!"
      '';
    };
  };
}
