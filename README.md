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

You should run the program from an elevated PowerShell window, or the LNK shortcut.
*Running directly from File Explorer GUI also works if you run as administrator.*

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

## Future
Features that will be in the next few versions:
- Better object tracking (any object)
- Automatic alerts from statistics and set rules
- Statistical outlier identification
- More advanced statistical analysis
- Process lineage tracking
- Upgraded UI