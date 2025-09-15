@echo off
REM Traffic Monitor Build Script for Windows
REM This script builds both frontend and backend components

echo 🚀 Starting Traffic Monitor build process...

REM Check if Node.js is installed
where node >nul 2>&1
if %errorlevel% neq 0 (
    echo [ERROR] Node.js is not installed
    pause
    exit /b 1
)

REM Check if npm is installed
where npm >nul 2>&1
if %errorlevel% neq 0 (
    echo [ERROR] npm is not installed
    pause
    exit /b 1
)

REM Check if Go is installed
where go >nul 2>&1
if %errorlevel% neq 0 (
    echo [ERROR] Go is not installed
    pause
    exit /b 1
)

echo [INFO] All dependencies are available

REM Clean previous build
echo [INFO] Cleaning previous build...
if exist build rmdir /s /q build
if exist frontend\dist rmdir /s /q frontend\dist
go clean -cache

REM Build frontend
echo [INFO] Building frontend...
cd frontend
echo [INFO] Installing frontend dependencies...
npm install
if %errorlevel% neq 0 (
    echo [ERROR] Frontend dependency installation failed
    cd ..
    pause
    exit /b 1
)

echo [INFO] Building frontend for production...
npm run build
if %errorlevel% neq 0 (
    echo [ERROR] Frontend build failed
    cd ..
    pause
    exit /b 1
)

cd ..

REM Build backend
echo [INFO] Building backend...
cd backend

echo [INFO] Downloading Go dependencies...
go mod download
go mod tidy

echo [INFO] Building backend...
mkdir ..\build 2>nul
go build -o ..\build\traffic-sniff.exe cmd\server\main.go
if %errorlevel% neq 0 (
    echo [ERROR] Backend build failed
    cd ..
    pause
    exit /b 1
)

cd ..

REM Package distribution
echo [INFO] Packaging distribution...
xcopy frontend\dist build\frontend\ /e /i /q
mkdir build\data 2>nul

REM Create README
echo # Traffic Monitor > build\README.txt
echo. >> build\README.txt
echo ## Quick Start >> build\README.txt
echo. >> build\README.txt
echo 1. Run the server: >> build\README.txt
echo    traffic-sniff.exe -port=8080 -interface=Ethernet >> build\README.txt
echo. >> build\README.txt
echo 2. Open your browser and navigate to: http://localhost:8080 >> build\README.txt
echo. >> build\README.txt
echo ## Command Line Options >> build\README.txt
echo. >> build\README.txt
echo - -port: Server port (default: 8080) >> build\README.txt
echo - -interface: Network interface to monitor >> build\README.txt
echo - -storage: Data storage path (default: .\data) >> build\README.txt

REM Create startup script
echo @echo off > build\start.bat
echo set PORT=8080 >> build\start.bat
echo set INTERFACE= >> build\start.bat
echo set STORAGE=.\data >> build\start.bat
echo. >> build\start.bat
echo echo Starting Traffic Monitor... >> build\start.bat
echo echo Port: %%PORT%% >> build\start.bat
echo echo Interface: %%INTERFACE%% >> build\start.bat
echo echo Storage: %%STORAGE%% >> build\start.bat
echo. >> build\start.bat
echo traffic-sniff.exe -port=%%PORT%% -interface=%%INTERFACE%% -storage=%%STORAGE%% >> build\start.bat
echo pause >> build\start.bat

echo [SUCCESS] Build completed successfully!
echo.
echo 📦 Build Summary:
echo ==================
echo Frontend: ✅ Built
echo Backend:  ✅ Built
echo Package:  ✅ Ready in build\ directory
echo.
echo 📁 Build contents:
dir build
echo.
echo 🎉 You can now run start.bat in the build directory!
pause