## kanastar
simple, scalable docker orchestrator written in go.

> [!WARNING]
> Highly experimental release in active development, not suitable for production yet.


[![Build and Release](https://github.com/surajsharma/kanastar/actions/workflows/release.yml/badge.svg)](https://github.com/surajsharma/kanastar/actions/workflows/release.yml) [![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)


<img src="./docs/images/kanastar.png" width="250">

>  "I will show you the home of the defiantly disobedient." (Quran 7:145)


## Architecture Overview

![architecture](./docs/images/architecture.svg)


## Usage

```
Kanastar is a dead simple docker orchestrator designed with spot VMs in mind

Usage:
  kanastar [command]

Available Commands:
  help        Help about any command
  manager     Manager command to operate a Kanastar manager node.
  node        Node command to list nodes.
  run         Run a new task.
  status      Status command to list tasks.
  stop        Stop a running task.
  worker      Worker command to operate a Kanastar worker node.

Flags:
  -h, --help   help for kanastar

Use "kanastar [command] --help" for more information about a command.
```

[changelog](./CHANGELOG)
