{
  lib,
  runCommand,
  makeWrapper,
  virtualenv,
  tg,
  git,
  openssh,
  version,
}:
runCommand "tgmcp-${version}" {
  nativeBuildInputs = [makeWrapper];
  meta = {
    description = "A typed MCP server for the tg Tangled client";
    mainProgram = "tgmcp";
    platforms = lib.platforms.unix;
  };
} ''
  mkdir -p "$out/bin"
  makeWrapper ${virtualenv}/bin/tgmcp "$out/bin/tgmcp" \
    --prefix PATH : ${lib.makeBinPath [tg git openssh]}
''
