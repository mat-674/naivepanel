@echo off
setlocal

:: go-sqlite3 requires CGO at runtime. A plain GOOS/GOARCH cross-build turns
:: CGO off and produces a binary that starts with the driver's stub error.
:: Install the two Linux cross-compilers below, or build on the target Linux
:: host with scripts/install.sh instead.
where x86_64-linux-gnu-gcc >nul 2>&1
if errorlevel 1 (
    echo ERROR: x86_64-linux-gnu-gcc is required for the linux/amd64 build.
    echo Build on the target Linux server, or install a Linux cross-compiler toolchain.
    exit /b 1
)

where aarch64-linux-gnu-gcc >nul 2>&1
if errorlevel 1 (
    echo ERROR: aarch64-linux-gnu-gcc is required for the linux/arm64 build.
    echo Build on the target Linux server, or install a Linux cross-compiler toolchain.
    exit /b 1
)

set BIN_NAME=naivepanel
set OUTPUT_DIR=bin
mkdir %OUTPUT_DIR% 2>nul

echo Building for Linux (amd64)...
set GOOS=linux
set GOARCH=amd64
set CGO_ENABLED=1
set CC=x86_64-linux-gnu-gcc
go build -ldflags="-s -w" -o %OUTPUT_DIR%/%BIN_NAME%-linux-amd64 main.go
if errorlevel 1 exit /b 1

echo Building for Linux (arm64)...
set GOOS=linux
set GOARCH=arm64
set CGO_ENABLED=1
set CC=aarch64-linux-gnu-gcc
go build -ldflags="-s -w" -o %OUTPUT_DIR%/%BIN_NAME%-linux-arm64 main.go
if errorlevel 1 exit /b 1

echo Build complete! Binaries are in the %OUTPUT_DIR% directory.
pause
