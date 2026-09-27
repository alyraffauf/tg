def main() -> None:
    from tgmcp.server import mcp

    mcp.run(transport="stdio")
