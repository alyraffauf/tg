{
  buildGoModule,
  lib,
  installShellFiles,
  makeWrapper,
  stdenv,
  git,
}:
buildGoModule {
  pname = "tg";
  version = "dev";
  src = ../.;
  vendorHash = "sha256-vAjC3nyJvXVvsBj+JXPN7dNPAdhwY64lqHLsOhTuVKc=";
  subPackages = ["cmd/tg"];

  nativeBuildInputs = [
    installShellFiles
    makeWrapper
  ];

  postInstall = ''
    wrapProgram $out/bin/tg --prefix PATH : ${lib.makeBinPath [git]}
  ''
  + lib.optionalString (stdenv.buildPlatform.canExecute stdenv.hostPlatform) ''
    # Man page dates stay reproducible because stdenv sets SOURCE_DATE_EPOCH,
    # which cobra/doc reads when GenManHeader.Date is unset.
    manPageDir=$(mktemp -d)
    $out/bin/tg man "$manPageDir"
    installManPage "$manPageDir"/*

    installShellCompletion --cmd tg \
      --bash <($out/bin/tg completion bash) \
      --fish <($out/bin/tg completion fish) \
      --zsh <($out/bin/tg completion zsh)
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
