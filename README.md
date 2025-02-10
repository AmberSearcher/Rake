# Rake

Web crawler for the amber search engine!

## Building Instructions

### Building for Windows on Windows

1. Open a terminal.
2. Run the following command:

```
go build -o builds/windows/RakeCrawler.exe
cp urls.txt blacklist.txt builds/windows`
```

### Building for Linux on Windows

1. Install a cross-compilation toolchain like `mingw-w64`.
2. Open a terminal.
3. Run the following command:

```
set GOOS=linux
go build -o builds/windows/RakeCrawler
cp urls.txt blacklist.txt builds/windows
```

### Building for Linux on Linux

1. Open a terminal.
2. Run the following command:

```
go build -o builds/linux/RakeCrawler.exe
cp urls.txt blacklist.txt builds/linux
```

### Building for Windows on Linux

1. Install a cross-compilation toolchain like `mingw-w64`.
2. Open a terminal.
3. Run the following command:

```
GOOS=windows GOARCH=amd64 go build -o builds/windows/RakeCrawler.exe
cp urls.txt blacklist.txt builds/windows
```
