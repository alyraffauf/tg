{
  go_1_26,
  fetchurl,
}:
# Pin the crypto/x509 fix for the eMudhra intermediate CA compromise.
go_1_26.overrideAttrs {
  version = "1.26.10";
  src = fetchurl {
    url = "https://go.dev/dl/go1.26.10.src.tar.gz";
    sha256 = "fd63e883ba2f2bfca93c1646b31ebef63e2b1e4d2b317c9789d976a68d81f36f";
  };
}
