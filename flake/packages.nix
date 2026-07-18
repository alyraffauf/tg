_: {
  perSystem = {pkgs, ...}: let
    tg = pkgs.callPackage ../nix/tg.nix {};
  in {
    packages = {
      inherit tg;
      default = tg;
    };
  };
}
