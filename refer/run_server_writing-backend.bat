@echo off
chcp 65001 >nul
setlocal enabledelayedexpansion

@REM 选择是admin模块还是client模块
echo ==============================
echo  1: admin(默认)
echo  2: client
echo ==============================
set "MODULE=admin"
set /p "MODULE=请选择(1/2, 直接回车默认1): "
if "%MODULE%"=="" set "MODULE=admin"
if "%MODULE%"=="1" set "MODULE=admin"
if "%MODULE%"=="2" set "MODULE=client"

REM ========== 配置区 ==========
set "PROJECT_DIR=D:\JAVA\nanjinggulou\nj-ai-writing-backend"
set "BUILD_ADMIN_DIR=D:\JAVA\nanjinggulou\build\writing-backend-%MODULE%"

set "JAR_NAME=sm-service-1.0.0.jar"
if "%MODULE%"=="client" set "JAR_NAME=agent-client-1.0.0.jar"
set "JAR_BUILD_REL=ai-agent-%MODULE%\target\%JAR_NAME%"

set "JAVA_HOME=E:\Program\JAVA\jdk-17.0.12"
set "JASYPT_PWD=t5zQeeYczDZgXQTjPtQjiatjeVfLeRu"
set "LOG_FILE=app.log"
REM ===========================

:menu
echo.
echo ==============================
echo  1: 直接启动 %BUILD_ADMIN_DIR%\%JAR_NAME%
echo  2: 构建(含 git pull + mvn) 然后启动
echo ==============================
set "SEL="
set /p "SEL=请选择(1/2, 直接回车默认1): "
if "%SEL%"=="" set "SEL=1"

if "%SEL%"=="1" goto option1
if "%SEL%"=="2" goto option2

echo.
echo [WARN] 输入无效: "%SEL%"，请重新选择
goto menu

:option1
if not exist "%BUILD_ADMIN_DIR%\%JAR_NAME%" (
  echo.
  echo [WARN] 未发现 "%BUILD_ADMIN_DIR%\%JAR_NAME%"
  set "YN="
  set /p "YN=是否执行构建然后启动? (Y/N, 回车默认Y): "
  if "%YN%"=="" set "YN=Y"
  if /i "%YN%"=="Y" goto option2
  goto menu
)
call :run
goto end

:option2
call :build
if errorlevel 1 goto end
call :run
goto end

:build
echo.
echo [INFO] 进入项目目录: %PROJECT_DIR%
pushd "%PROJECT_DIR%" || (echo [ERROR] 无法进入目录 & exit /b 1)

REM pull 前 HEAD
for /f "delims=" %%i in ('git rev-parse HEAD 2^>nul') do set "OLD_HEAD=%%i"

echo [INFO] git pull ...
call git pull
if errorlevel 1 (
  echo [ERROR] git pull 失败
  popd
  exit /b 1
)

REM pull 后 HEAD
for /f "delims=" %%i in ('git rev-parse HEAD 2^>nul') do set "NEW_HEAD=%%i"

REM 根据是否有更新决定是否 clean
if "%OLD_HEAD%"=="%NEW_HEAD%" (
  echo [INFO] 仓库无变化 mvn package
  call mvn package
) else (
  echo [INFO] 仓库有更新 mvn clean package
  call mvn clean package
)

if errorlevel 1 (
  echo [ERROR] mvn 构建失败
  popd
  exit /b 1
)

REM 确保 build/admin 目录存在（绝对路径）
if not exist "%BUILD_ADMIN_DIR%" mkdir "%BUILD_ADMIN_DIR%"

REM 复制 jar 到 build/admin（绝对路径）
if not exist "%PROJECT_DIR%\%JAR_BUILD_REL%" (
  echo [ERROR] 未找到构建产物: "%PROJECT_DIR%\%JAR_BUILD_REL%"
  popd
  exit /b 1
)

echo [INFO] 复制 JAR -> "%BUILD_ADMIN_DIR%\"
copy /y "%PROJECT_DIR%\%JAR_BUILD_REL%" "%BUILD_ADMIN_DIR%\" >nul
if errorlevel 1 (
  echo [ERROR] 复制 JAR 失败
  popd
  exit /b 1
)

popd
exit /b 0

:run
echo.
echo [INFO] 进入运行目录: %BUILD_ADMIN_DIR%
pushd "%BUILD_ADMIN_DIR%" || (echo [ERROR] 无法进入运行目录 & exit /b 1)

if not exist "%JAR_NAME%" (
  echo [ERROR] 运行目录下未找到: "%cd%\%JAR_NAME%"
  popd
  exit /b 1
)
"%JAVA_HOME%\bin\java.exe" -XshowSettings:properties -version 2>&1 | findstr /i "file.encoding sun.jnu.encoding"
echo [INFO] 启动中... 日志输出到: "%cd%\%LOG_FILE%"  秘钥: "%JASYPT_PWD%"
:: start "" /b   additional-
:: "%JAVA_HOME%\bin\java.exe" -Xmx512m -Xms512m -jar "%JAR_NAME%" --jasypt.encryptor.password=%JASYPT_PWD% > "%LOG_FILE%" 2>&1
@REM "%JAVA_HOME%\bin\java.exe" -Dfile.encoding=UTF-8 -Dsun.jnu.encoding=UTF-8 -Xmx300m -Xms300m -jar "%JAR_NAME%" --jasypt.encryptor.password=%JASYPT_PWD% --spring.config.location=file:./application.yml
"%JAVA_HOME%\bin\java.exe" -Dfile.encoding=UTF-8 -Dsun.jnu.encoding=UTF-8 -Xmx300m -Xms300m -jar "%JAR_NAME%" --spring.profiles.active=dev

popd
exit /b 0

:end
echo.
echo [INFO] 脚本结束
endlocal