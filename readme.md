> [!CAUTION]
> Highly experimental release in active development, not suitable for production yet.

# kanastar
simple, scalable docker orchestrator written in go.

[![Build and Release](https://github.com/surajsharma/kanastar/actions/workflows/release.yml/badge.svg)](https://github.com/surajsharma/kanastar/actions/workflows/release.yml) [![Go Report Card](https://goreportcard.com/badge/github.com/surajsharma/kanastar)](https://goreportcard.com/report/github.com/surajsharma/kanastar) [![CodeQL](https://github.com/surajsharma/kanastar/actions/workflows/github-code-scanning/codeql/badge.svg)](https://github.com/surajsharma/kanastar/actions/workflows/github-code-scanning/codeql) [![Scorecard supply-chain security](https://github.com/surajsharma/kanastar/actions/workflows/scorecard.yml/badge.svg)](https://github.com/surajsharma/kanastar/actions/workflows/scorecard.yml) [![OpenSSF Scorecard](https://api.scorecard.dev/projects/github.com/surajsharma/kanastar/badge)](https://scorecard.dev/viewer/?uri=github.com/surajsharma/kanastar) [![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)


![kanastar](./docs/images/kanastar_small.png)

## Architecture Overview

![architecture](./docs/images/architecture.svg)


## Usage

```
Kanastar is a dead simple docker orchestrator designed with spot VMs in mind

Usage:
  kanactl [command]

Available Commands:
  help        Help about any command
  manager     Manager command to operate a Kanastar manager node.
  node        Node command to list nodes.
  run         Run a new task.
  status      Status command to list tasks.
  stop        Stop a running task.
  worker      Worker command to operate a Kanastar worker node.

Flags:
  -h, --help   help for kanactl

Use "kanactl [command] --help" for more information about a command.
```

## Building

- Pull the repo and run `make build` in the dir with the `Makefile`

[changelog](./CHANGELOG)
