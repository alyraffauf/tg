{
  buildGoModule,
  lib,
}:
buildGoModule {
  pname = "tg";
  version = "dev";
  src = ../.;
  vendorHash = "sha256-4e3RU0z5rh8cDSW7fQSfQM8sqeD53PA0BYGTTjtF23E=";
  subPackages = ["cmd/tg"];
  env.CGO_ENABLED = "0";

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
