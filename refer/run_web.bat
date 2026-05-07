@echo off
setlocal enabledelayedexpansion
REM 获取当前日期时间并格式化为yyyyMMddHHmm  %date:~0,4%%date:~5,2%%date:~8,2%%time:~0,2%%time:~3,2%%time:~6,2%
set START_TIME="%date:~0,4%%date:~5,2%%date:~8,2%%time:~0,2%%time:~3,2%%time:~6,2%"

:moudle
echo  please choose:
echo 1. nj-ai-writing-web-pc (default)
echo 2. nj-ai-writing-web
echo 3. nj-ai-tourism-web-pc
set /p dirname="please enter your choose: "
if "%dirname%"=="" ( set dirname=nj-ai-writing-web-pc )
if "%dirname%"=="1" ( set dirname=nj-ai-writing-web-pc )
if "%dirname%"=="2" ( set dirname=nj-ai-writing-web )
if "%dirname%"=="3" ( set dirname=nj-ai-tourism-web-pc )

cd D:\JAVA\nanjinggulou\%dirname%
pwd

:menu
echo  please choose (1-6):
echo 1. dev (default)
echo 2. build-prod
echo 3. build-test
echo 0. open dir
set /p choice="please enter your choose: "

if "%choice%"=="" ( npm run dev )
if "%choice%"=="1" ( npm run dev )
if "%choice%"=="2" (
  call npm run build
  timeout /t 2
  if not exist "dist-zip" (
    mkdir "dist-zip"
  ) 
  call zip -r .\dist-gwzs-client-prod-%START_TIME%.zip .\dist
  move .\dist-gwzs-client-prod-%START_TIME%.zip .\dist-zip
)
if "%choice%"=="3" (
  call npm run build:stage
  timeout /t 2
  if not exist "dist-zip" (
    mkdir "dist-zip"
  ) 
  call zip -r .\dist-gwzs-client-test-%START_TIME%.zip .\dist
  move .\dist-gwzs-client-test-%START_TIME%.zip .\dist-zip
)
if "%choice%"=="0" ( explorer .\dist-zip )
pause

echo done, you can choose again...........
goto menu

endlocal