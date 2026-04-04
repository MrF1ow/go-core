@echo off
REM Database Migration Script for Windows
REM Interactive tool for managing database migrations

setlocal EnableDelayedExpansion

echo ======================================
echo Database Migration Tool
echo ======================================
echo.
echo 1. Apply pending migrations
echo 2. Rollback last migration
echo 3. List available migrations
echo 4. Show migration status
echo 5. Backup database
echo 6. Test database connection
echo 0. Exit
echo.

set /p choice="Enter your choice (0-6): "

if "%choice%"=="1" goto apply_migrations
if "%choice%"=="2" goto rollback_migration
if "%choice%"=="3" goto list_migrations
if "%choice%"=="4" goto show_status
if "%choice%"=="5" goto backup_database
if "%choice%"=="6" goto test_connection
if "%choice%"=="0" goto exit_script
goto invalid_choice

:apply_migrations
call "%~dp0apply_pending_migrations.sh"
goto end

:rollback_migration
call "%~dp0rollback_last_migration.sh"
goto end

:list_migrations
echo.
echo Available Migrations:
echo.
if exist "migrations" (
    dir /b migrations\*.sql 2>nul | findstr /v "_rollback.sql"
    if %ERRORLEVEL% NEQ 0 (
        echo No migration files found
    )
) else (
    echo Migrations directory not found!
)
goto end

:show_status
echo.
echo Tracked Migrations:
echo.
docker exec -i auth_db psql -U postgres -d auth_db -c "SELECT version, applied_at FROM schema_migrations ORDER BY applied_at;" 2>nul
if %ERRORLEVEL% NEQ 0 (
    echo Could not query migration status. Is the database running?
)
goto end

:backup_database
call "%~dp0backup_db.bat"
goto end

:test_connection
echo.
echo Testing Database Connection
echo.
docker exec auth_db psql -U postgres -d auth_db -c "SELECT version();" >nul 2>nul
if %ERRORLEVEL% EQU 0 (
    echo [OK] Connection successful!
    echo.
    docker exec auth_db psql -U postgres -d auth_db -c "SELECT version();"
) else (
    echo [FAIL] Connection failed!
    echo Start the database with: make docker-dev
)
goto end

:invalid_choice
echo Invalid choice.
exit /b 1

:exit_script
echo Exiting...
exit /b 0

:end
echo.
echo ======================================
echo Done.
echo ======================================
pause
