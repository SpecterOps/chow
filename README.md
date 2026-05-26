# chow (confirm hound output worthiness)

chow is a command line tool to validate BloodHound payloads

# Usage

```bash
chow -output errors.txt test.json
```

`-output` will redirect errors to an output file. Otherwise errors will be written to stdout

# Installation

```bash
go install github.com/specterops/chow@latest
```

Or you can clone the repo and run the following command from the top level:

```bash
go install .
```

# Benchmarking

Use `chowbench` when you want to measure validation performance across one or more files without changing the normal `chow` CLI.

```bash
go run ./cmd/chowbench -runs 5 -warmup 1 payload-one.json payload-two.json
```

The harness loads the JSON schemas once, validates each file for the requested number of runs, and prints a table with byte size, status, average duration, min/max duration, and error counts.

By default, invalid payloads are still measured and reported. Add `-strict` if invalid payloads should make the command exit non-zero:

```bash
go run ./cmd/chowbench -runs 5 -strict payload-one.json payload-two.json
```

# JSON Schema

Want to add the OpenGraph schema to your JSON document?

```json
{
  "$schema": "https://raw.githubusercontent.com/SpecterOps/chow/refs/heads/main/pkg/payload/jsonschema/schema.json"
}
```

Most editors will ask you to trust the schema's source. Be sure to add the following URL to your trusted domains

```text
https://raw.githubusercontent.com/SpecterOps/chow/refs/heads/main/pkg/payload/jsonschema/
```
