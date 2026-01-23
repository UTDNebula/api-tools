@echo off
::vars
set EXEC_NAME=api-tools.exe

::simulate makefile-like branching
set skip=1
if "%1"=="setup" goto %1
if "%1"=="check" goto %1
if "%1"=="build" goto %1
if "%1"=="test" goto %1
set skip=0

:setup
echo Performing setup...
go install honnef.co/go/tools/cmd/staticcheck@latest && ^
go install golang.org/x/tools/cmd/goimports@latest
if ERRORLEVEL 1 exit /b %ERRORLEVEL% :: fail if error occurred
echo Setup done!
if %skip%==1 exit
echo[

:check
echo Performing checks...
go mod tidy && ^
go vet ./... && ^
staticcheck ./... && ^
gofmt -w . && ^
goimports -w .
if ERRORLEVEL 1 exit /b %ERRORLEVEL% :: fail if error occurred
echo Checks done!
if %skip%==1 exit
echo[

:build
echo Building...
go build -o %EXEC_NAME% ./main.go
if ERRORLEVEL 1 exit /b %ERRORLEVEL% :: fail if error occurred
echo Build complete!
if %skip%==1 exit
echo[

:test
echo Testing...
go test ./...
if ERRORLEVEL 1 exit /b %ERRORLEVEL% :: fail if error occurred
echo Testing complete!
if %skip%==1 exit
echo[