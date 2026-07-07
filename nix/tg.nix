{
  buildGoModule,
  lib,
}:
buildGoModule {
  pname = "tg";
  version = "dev";
  src = ../.;
  vendorHash = "sha256-wv8rvPaQH4iY7KpnWrI59dmUVhxvNki8EIhklY4rMPU=";
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
