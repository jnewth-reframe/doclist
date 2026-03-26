# doclist

A single-file Go CLI tool that walks the Onshape folder tree starting from a given folder ID and produces an interactive HTML page and a Graphviz DOT file.

## Build

```sh
go build
```

## Usage

```sh
./doclist [--dump] [--depth=N] [--root-type=<type>] [--secrets=<path>] <folder-id> [output-base]
```

| Argument | Description |
|---|---|
| `folder-id` | Onshape folder ID — the `nodeId=` value in the web UI URL |
| `output-base` | Base name for output files (default: `doclist`) |
| `--depth=N` | Max folder recursion depth; 0 = unlimited (default) |
| `--root-type` | Node type for the root folder: `folder` (default) or `resourcecompanyowner` |
| `--secrets` | Path to credentials file (default: `secrets.json`) |
| `--dump` | Write raw API responses to `<output-base>_dump/` for inspection |

**Regular folder:**
```sh
./doclist --depth=3 7bff278ed5b795a1f074acb5 reframe
# produces reframe.html and reframe.dot
```

**Company root folder:**
```sh
./doclist --root-type=resourcecompanyowner --secrets=personal/secrets.json 5a998a2b5618f7127b13db54 furious
```

The folder ID comes from the `nodeId=` parameter in the Onshape web UI URL. The `resourceType=` value in the URL indicates which `--root-type` to use:

| Web UI `resourceType` | `--root-type` |
|---|---|
| `folder` | `folder` (default) |
| `resourcecompanyowner` | `resourcecompanyowner` |

## Output

- **`<output-base>.html`** — Interactive tree with collapsible folders, clickable links for all nodes and documents, and doc IDs shown inline.
- **`<output-base>.dot`** — Graphviz DOT file. Render with:
  ```sh
  dot -Tsvg output.dot -o output.svg   # SVG with clickable links
  dot -Tpng output.dot -o output.png
  ```

## Credentials

Copy `secrets.json.template` to `secrets.json` and fill in your Onshape API keys:

```json
{
  "accessKey": "<your-access-key>",
  "secretKey": "<your-secret-key>"
}
```

`secrets.json` is gitignored. Install the pre-commit hook to prevent accidental commits:

```sh
cp hooks/pre-commit .git/hooks/pre-commit
```

Use `--secrets` to point to a different credentials file when targeting a different Onshape account.

## Architecture

Everything lives in `doclist.go` (single `main` package, no external dependencies). API calls use `net/http` directly with HTTP Basic Auth against `GET /api/globaltreenodes/{type}/{fid}`.

Key functions:
- `buildTree` — recursively fetches folder contents and assembles a `node` tree
- `fetchItems` — calls the Onshape folder listing API, follows pagination via `next` URL
- `writeHTML` — renders the tree as a collapsible HTML page
- `writeDOT` — renders the tree as a Graphviz DOT file
- `apiGetRaw` — shared HTTP helper with Basic Auth
