@echo off
REM start-local.bat - One-command local startup for rate-limiter (Windows)

echo === Rate Limiter Local Startup ===

REM Check if .env already exists
if exist docker\.env (
    echo Found existing docker\.env file - preserving credentials to maintain Redis data access
    echo.
    set /p regenerate="Regenerate new credentials? This will lose existing Redis data [y/N]: "
    if /i not "!regenerate!"=="y" (
        echo Using existing credentials...
        goto start_services
    )
    echo Creating backup of existing .env...
    copy docker\.env docker\.env.backup >nul
    echo.
)

echo Generating new secure credentials...
echo.

REM Generate secure random passwords
setlocal enabledelayedexpansion

REM Generate Redis password (24 characters)
set "chars=ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
set "redis_password="
for /L %%i in (1,1,24) do (
    set /a "rand=!random! %% 62"
    for %%j in (!rand!) do set "redis_password=!redis_password!!chars:~%%j,1!"
)

REM Generate JWT secret (48 characters)
set "jwt_secret="
for /L %%i in (1,1,48) do (
    set /a "rand=!random! %% 62"
    for %%j in (!rand!) do set "jwt_secret=!jwt_secret!!chars:~%%j,1!"
)

REM Create .env file in docker directory
echo # Rate Limiter Environment Configuration > docker\.env
echo # Generated on %date% %time% >> docker\.env
echo. >> docker\.env
echo # Redis Configuration >> docker\.env
echo redis_password=!redis_password! >> docker\.env
echo. >> docker\.env
echo # JWT Configuration >> docker\.env
echo jwt_secret=!jwt_secret! >> docker\.env

echo New credentials generated and saved to docker\.env
echo WARNING: New Redis password means existing Redis data will be inaccessible
echo.

:start_services

REM Start Docker services
echo Starting Docker services...
cd docker
docker-compose up -d

if %ERRORLEVEL% EQU 0 (
    echo.
    echo === Services Started Successfully! ===
    echo.
    echo Rate Limiter: http://localhost:8080
    echo Demo API: http://localhost:9080/hello
    echo Redis Commander: http://localhost:8081
    echo.
    echo To stop services: docker-compose down
    echo To view logs: docker-compose logs -f
) else (
    echo.
    echo ERROR: Failed to start services
    echo Check Docker installation and try again
)

cd ..
pause