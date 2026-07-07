_: {
  perSystem = {pkgs, ...}: {
    packages = {
      tg = pkgs.callPackage ../nix/tg.nix {};
    };
  };
}
