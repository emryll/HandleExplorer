# HandleExplorer
Interactive command-line tool for capturing, searching and analyzing handle data.

https://github.com/user-attachments/assets/f405c4cd-e00a-4a25-95c5-0b2fcc7a9c91

## Features
> New features and upgraded UI coming very soon...

- Search for processes with filters
- View objects accessed by a process
- Search for handles by describing object access
- Perform statistical analysis on captured handle data
- View statistics about object access system-wide
- Find overlapping object access
  
## Requirements
- Windows 10 or 11 (x64)

If you want to build it yourself:
- C compiler
- [Go](https://go.dev/doc/install)

## Usage
It is recommended to simply [download the latest release](https://github.com/emryll/HandleExplorer/releases).

If you want, you can also build it yourself:
```
git clone https://github.com/emryll/HandleExplorer
cd .\HandleExplorer\

go mod init HandleExplorer
go mod tidy
go build
.\HandleExplorer.exe
```
*Run the help command if needed*
