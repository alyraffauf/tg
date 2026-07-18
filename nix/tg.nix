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
  vendorHash = "sha256-NKfVkBe863hXwjY1Jic+0rTnRD6v25pbzDx9W4W5OqU=";
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
