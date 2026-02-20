@echo off
setlocal

:: Set variables
set BIN_NAME=naivepanel
set OUTPUT_DIR=bin
mkdir %OUTPUT_DIR% 2>nul

echo Building for Linux (amd64)...
set GOOS=linux
set GOARCH=amd64
go build -ldflags="-s -w" -o %OUTPUT_DIR%/%BIN_NAME%-linux-amd64 main.go

echo Building for Linux (arm64)...
set GOOS=linux
set GOARCH=arm64
go build -ldflags="-s -w" -o %OUTPUT_DIR%/%BIN_NAME%-linux-arm64 main.go

echo Build complete! Binaries are in the %OUTPUT_DIR% directory.
pause
