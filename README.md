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

# JSON Schema

Want to add the OpenGraph schema to your JSON document?

```json
{
  "$schema": "https://raw.githubusercontent.com/SpecterOps/chow/refs/heads/main/pkg/validator/jsonschema/payload-schema.json"
}
```

Most editors will ask you to trust the schema's source. Be sure to add the following URL to your trusted domains

```text
https://raw.githubusercontent.com/SpecterOps/chow/refs/heads/main/pkg/validator/jsonschema/
```
