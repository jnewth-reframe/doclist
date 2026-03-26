# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

A single-file Go CLI tool that fetches the Onshape document/folder tree for the Reframe Systems top-level folder and prints it as a tree view to stdout.

## Build and run

```sh
go build         # produces ./doclist binary
```

```sh
./doclist <folder-id>
# e.g.: ./doclist 7bff278ed5b795a1f074acb5
# The folder ID comes from the nodeId= parameter in the Onshape web UI URL.
```

## Credentials

`secrets.json` (gitignored) must contain Onshape API keys:

```json
{
  "accessKey": "<your-access-key>",
  "secretKey": "<your-secret-key>"
}
```

See `secrets.json.template` for the 1Password paths. The pre-commit hook in `hooks/pre-commit` blocks commits that include `secrets.json`. To install: `cp hooks/pre-commit .git/hooks/pre-commit`.

## Architecture

Everything lives in `doclist.go` (single `main` package). No go-client dependency — all API calls use `net/http` directly with basic auth.

**Key functions:**
- `main` — loads credentials, resolves root folder ID (const or CLI arg), prints tree header, calls `printTree`
- `printTree` — recursively fetches folder contents and prints the tree using box-drawing characters
- `getFolderItems` — calls the Onshape folder listing API for a given folder ID, returns `[]item`
- `apiGet` — shared HTTP helper with basic auth, used by all API calls
- `loadSecrets` — reads `secrets.json` and returns access/secret key pair
- `verify` — fatal-error helper used throughout instead of explicit error handling
