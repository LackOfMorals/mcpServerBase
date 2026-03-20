# A Foundational MCP Server
The MCP Server which can be build off to do things

## Features






## Prerequisites

- Go 1.25+ (see `go.mod`)


## Installation
### Clone the repository 

```bash
git clone https://github.com/LackOfMorals/mcpServerBase
```

### Install Dependencies

```bash
cd mcpServerBase
go mod download
```


### Build

MCP Server will need to be compiled before use.   Do this with

```Bash
go build -o ./bin/mcp-base ./cmd/mcp-base

```

### Testing

You can test before using with LLMs by using MCP Inspector. 

```bash
npx @modelcontextprotocol/inspector go run ./cmd/mcp-base

```


## Using with Claude Desktop

```json
{
  "mcpServers": {
    "mcp-base": {
      "command": "<FULL PATH TO MCP BINARY>",
      "args": [],
      "env": {
      }
    }
  }
}
```

