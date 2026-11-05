# tgmcp

A Tangled MCP server using FastMCP, Pydantic, and the `tg` CLI.

## Run

`nix run .#tgmcp` from the repository root starts the stdio server with `tg`, Git, and SSH included.
From `mcp/`, run `uv run tgmcp` with `tg` on your PATH.
Log in with `tg auth login <handle>` before using authenticated tools.

Point your MCP client at the checkout:

```json
{
  "mcpServers": {
    "tangled": {
      "command": "nix",
      "args": ["run", "/path/to/tg#tgmcp"]
    }
  }
}
```

## Configuration

| Variable                 | Purpose                                                                                      |
| ------------------------ | -------------------------------------------------------------------------------------------- |
| `TGMCP_EXECUTABLE`       | Override the `tg` executable                                                                 |
| `TGMCP_ACCOUNT`          | Override tg's active account                                                                 |
| `TGMCP_CONFIG`           | Override tg's config path                                                                    |
| `TGMCP_TIMEOUT_SECONDS`  | Command timeout, defaults to 30 seconds                                                      |
| `TGMCP_MAX_OUTPUT_BYTES` | Maximum stdout per command, defaults to 128 MiB. Stderr diagnostics are truncated at 64 KiB. |

The server inherits tg's environment and credentials. Local Git tools need
absolute paths on the server's filesystem.

## Development

Enter `nix develop` from the repository root, then `cd mcp` and run `uv sync`.

- `nix build .#tgmcp` builds the package from the repository root.
- `nix fmt` formats the code from the repository root.
- `uv run pytest` runs local tests with a fake `tg`, without an account or network.
- `uv run fastmcp dev inspector` opens MCP Inspector.
