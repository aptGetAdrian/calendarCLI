#!/usr/bin/env bash
set -e

go build -o calendarCli ./cmd/main.go

./calendarCli 