# Authentication

`tg` authenticates against your atproto PDS. Log in interactively with OAuth:

```bash
tg auth login alice.example.com
```

For headless use, pass an atproto app password as the second argument:

```bash
tg auth login alice.example.com xxxx-xxxx-xxxx-xxxx
```

To avoid exposing the app password in shell history, pass it on standard input:

```bash
printf '%s\n' "$ATPROTO_APP_PASSWORD" | tg auth login alice.example.com --password-stdin
```

## Multiple accounts

Multiple accounts can be stored at once. Logging in adds or replaces that
account and selects it as the default.

- `tg auth list` — list stored accounts
- `tg auth switch <handle-or-did>` — select the default account
- `tg auth status` — show the active account
- `tg auth logout` — remove the selected account (`--all` removes every account)

Use `--account <handle-or-did>` (or `TG_ACCOUNT`) for a one-command override of
the default account.

## Token storage

`tg` stores OAuth and app-password sessions in the system keyring: macOS
Keychain or the Secret Service on Linux (GNOME Keyring / KWallet). The keyring
unlocks with your login session, so no separate passphrase is needed.

The keyring is accessed on first use (not at startup), so authentication only
fails once you run a command that needs a session. On Linux this requires a
Secret Service provider to be running; on a headless system without a D-Bus
session bus (e.g. a server or container), install and start
`gnome-keyring-daemon` or `kwalletd`, or set `DBUS_SESSION_BUS_ADDRESS`.

See the [command reference](commands/tg_auth.md) for the full `tg auth`
subcommand list.
