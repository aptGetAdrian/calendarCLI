#!/usr/bin/env bash
set -e

echo "🔨 Building calendarCli..."
go build -o calendarCli

echo "🔐 Running connect..."
./calendarCli connect