@echo off
setlocal
cd /d "%~dp0"

set LOG_FILE=app.log
set EXE_FILE=drun_debug.exe

set "COMMAND=cmd /c %EXE_FILE% > %LOG_FILE% 2>&1"

echo CreateObject("Wscript.Shell").Run "%COMMAND%", 0, False > temp_run.vbs
wscript temp_run.vbs
del temp_run.vbs

echo "%EXE_FILE% started..."