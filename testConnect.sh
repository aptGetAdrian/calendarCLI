#!/usr/bin/env bash
set -e

echo "🔨 Building calendarCli..."
go build -o calendarCli ./cmd/main.go

echo "🔐 Running connect..."
./calendarCli 