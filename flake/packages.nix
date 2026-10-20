{inputs, ...}: {
  perSystem = {pkgs, ...}: let
    go = pkgs.callPackage ../nix/go.nix {};
    tg = pkgs.callPackage ../nix/tg.nix {
      buildGoModule = pkgs.buildGoModule.override {inherit go;};
    };
    workspace = inputs.uv2nix.lib.workspace.loadWorkspace {workspaceRoot = ../mcp;};
    overlay = workspace.mkPyprojectOverlay {sourcePreference = "wheel";};
    pythonSet =
      (pkgs.callPackage inputs.pyproject-nix.build.packages {
        python = pkgs.python314;
      }).overrideScope (pkgs.lib.composeManyExtensions [
        inputs.pyproject-build-systems.overlays.wheel
        overlay
      ]);
    tgmcp = pkgs.callPackage ../nix/tgmcp.nix {
      inherit tg;
      virtualenv = pythonSet.mkVirtualEnv "tgmcp-env" workspace.deps.default;
      version = (builtins.fromTOML (builtins.readFile ../mcp/pyproject.toml)).project.version;
    };
  in {
    packages = {
      inherit tg tgmcp;
      default = tg;
    };
  };
}
