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
