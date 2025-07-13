@echo off
REM setup-env.bat - Interactive environment configuration for rate-limiter (Windows)

echo === Rate Limiter Environment Setup ===
echo This script will create a .env file with your configuration.
echo.

REM Clear existing .env file
if exist .env (
    echo Found existing .env file. Creating backup as .env.backup
    copy .env .env.backup >nul
)

REM Create new .env file
echo # Rate Limiter Environment Configuration > .env
echo # Generated on %date% %time% >> .env
echo. >> .env

echo === Redis Configuration ===
set /p redis_password_input="Redis password [redis_pass_123]: "
if "%redis_password_input%"=="" set redis_password_input=redis_pass_123
echo redis_password=%redis_password_input% >> .env

echo.
echo === JWT Configuration ===
set /p jwt_secret_input="JWT secret key [your-super-secure-jwt-secret-key-that-is-at-least-32-characters-long]: "
if "%jwt_secret_input%"=="" set jwt_secret_input=your-super-secure-jwt-secret-key-that-is-at-least-32-characters-long
echo jwt_secret=%jwt_secret_input% >> .env

echo.
echo === Database Configuration ===
set /p mongo_url_input="MongoDB connection URL [mongodb://username:password@localhost:27017/ratelimiter]: "
if "%mongo_url_input%"=="" set mongo_url_input=mongodb://username:password@localhost:27017/ratelimiter
echo mongo_url=%mongo_url_input% >> .env

set /p backend_url_input="Backend service URL [http://localhost:3000/api]: "
if "%backend_url_input%"=="" set backend_url_input=http://localhost:3000/api
echo backend_url=%backend_url_input% >> .env

echo.
echo === Server Configuration ===
set /p port_input="Server port [8080]: "
if "%port_input%"=="" set port_input=8080
echo port=%port_input% >> .env

set /p limit_count_input="Default rate limit requests [100]: "
if "%limit_count_input%"=="" set limit_count_input=100
echo default_limit_count=%limit_count_input% >> .env

set /p period_input="Default time period [1h]: "
if "%period_input%"=="" set period_input=1h
echo default_period=%period_input% >> .env

echo.
echo === Optional Development Settings ===
set /p log_level_input="Log level DEBUG/INFO/WARN/ERROR [INFO]: "
if "%log_level_input%"=="" set log_level_input=INFO
echo log_level=%log_level_input% >> .env

echo.
echo Environment configuration complete!
echo.
echo Configuration saved to: .env
if exist .env.backup echo Backup saved to: .env.backup
echo.
echo Next steps:
echo   1. Review the .env file: type .env
echo   2. Start services: docker-compose up -d
echo   3. Run the application: go run main.go
echo.
pause