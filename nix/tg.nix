{
  buildGoModule,
  lib,
  makeWrapper,
  git,
}:
buildGoModule {
  pname = "tg";
  version = "dev";
  src = ../.;
  vendorHash = "sha256-WDM1O622yKsP6qifLSh796qph5HzrJR42F8OpJwNzJQ=";
  subPackages = ["cmd/tg"];
  env.CGO_ENABLED = "0";

  nativeBuildInputs = [makeWrapper];

  postInstall = ''
    wrapProgram $out/bin/tg --prefix PATH : ${lib.makeBinPath [git]}
  '';

  ldflags = [
    "-s"
    "-w"
  ];

  meta = with lib; {
    description = "Terminal client for Tangled";
    homepage = "https://github.com/alyraffauf/tg";
    license = licenses.gpl3Plus;
    platforms = platforms.unix;
    mainProgram = "tg";
  };
}
