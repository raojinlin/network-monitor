#!/bin/bash

# Traffic Monitor Build Script
# This script builds both frontend and backend components

set -e  # Exit on any error

echo "🚀 Starting Traffic Monitor build process..."

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
FRONTEND_DIR="frontend"
BACKEND_DIR="backend"
BUILD_DIR="build"
DIST_DIR="dist"

# Function to print colored output
print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if required tools are installed
check_dependencies() {
    print_status "Checking dependencies..."
    
    if ! command -v node &> /dev/null; then
        print_error "Node.js is not installed"
        exit 1
    fi
    
    if ! command -v npm &> /dev/null; then
        print_error "npm is not installed"
        exit 1
    fi
    
    if ! command -v go &> /dev/null; then
        print_error "Go is not installed"
        exit 1
    fi
    
    print_success "All dependencies are available"
}

# Clean previous build
clean_build() {
    print_status "Cleaning previous build..."
    
    if [ -d "$BUILD_DIR" ]; then
        rm -rf "$BUILD_DIR"
    fi
    
    if [ -d "$FRONTEND_DIR/dist" ]; then
        rm -rf "$FRONTEND_DIR/dist"
    fi
    
    # Clean Go build cache
    go clean -cache
    
    print_success "Cleaned previous build"
}

# Build frontend
build_frontend() {
    print_status "Building frontend..."
    
    cd "$FRONTEND_DIR"
    
    # Install dependencies
    print_status "Installing frontend dependencies..."
    npm install
    
    # Build for production
    print_status "Building frontend for production..."
    npm run build
    
    if [ ! -d "dist" ]; then
        print_error "Frontend build failed - dist directory not found"
        exit 1
    fi
    
    cd ..
    print_success "Frontend build completed"
}

# Build backend
build_backend() {
    print_status "Building backend..."
    
    cd "$BACKEND_DIR"
    
    # Download Go dependencies
    print_status "Downloading Go dependencies..."
    go mod download
    go mod tidy
    
    # Build for current platform only (pcap requires CGO)
    print_status "Building backend for current platform..."
    print_warning "Cross-platform builds are disabled due to pcap CGO requirements"
    print_warning "To build for other platforms, run this script on the target platform"
    
    # Get current OS and architecture
    CURRENT_OS=$(go env GOOS)
    CURRENT_ARCH=$(go env GOARCH)
    
    print_status "Building for $CURRENT_OS/$CURRENT_ARCH..."
    
    if [ "$CURRENT_OS" = "windows" ]; then
        go build -o "../$BUILD_DIR/traffic-sniff.exe" cmd/server/main.go
    else
        go build -o "../$BUILD_DIR/traffic-sniff" cmd/server/main.go
    fi
    
    # Create platform-specific binary with full name
    if [ "$CURRENT_OS" = "windows" ]; then
        cp "../$BUILD_DIR/traffic-sniff.exe" "../$BUILD_DIR/traffic-sniff-$CURRENT_OS-$CURRENT_ARCH.exe"
    else
        cp "../$BUILD_DIR/traffic-sniff" "../$BUILD_DIR/traffic-sniff-$CURRENT_OS-$CURRENT_ARCH"
    fi
    
    cd ..
    print_success "Backend build completed"
}

# Package distribution
package_dist() {
    print_status "Packaging distribution..."
    
    # Create build directory
    mkdir -p "$BUILD_DIR"
    
    # Copy frontend dist to build directory
    cp -r "$FRONTEND_DIR/dist" "$BUILD_DIR/frontend"
    
    # Create data directory
    mkdir -p "$BUILD_DIR/data"
    
    # Create configuration files
    cat > "$BUILD_DIR/README.md" << EOF
# Traffic Monitor

## Quick Start

1. Run the server:
   \`\`\`bash
   ./traffic-sniff -port=8080 -interface=en0
   \`\`\`

2. Open your browser and navigate to: http://localhost:8080

## Command Line Options

- \`-port\`: Server port (default: 8080)
- \`-interface\`: Network interface to monitor (default: auto-detect)
- \`-storage\`: Data storage path (default: ./data)

## Platform-specific Binaries

- \`traffic-sniff\`: Current platform
- \`traffic-sniff-linux-amd64\`: Linux x64
- \`traffic-sniff-linux-arm64\`: Linux ARM64
- \`traffic-sniff-darwin-amd64\`: macOS Intel
- \`traffic-sniff-darwin-arm64\`: macOS Apple Silicon
- \`traffic-sniff-windows-amd64.exe\`: Windows x64

## Requirements

- Root/Administrator privileges for packet capture
- Network interface access

EOF

    # Create startup scripts
    cat > "$BUILD_DIR/start.sh" << 'EOF'
#!/bin/bash
# Default startup script for Traffic Monitor

# Detect platform
ARCH=$(uname -m)
OS=$(uname -s)

BINARY="traffic-sniff"

case "$OS" in
    Linux)
        case "$ARCH" in
            x86_64) BINARY="traffic-sniff-linux-amd64" ;;
            aarch64|arm64) BINARY="traffic-sniff-linux-arm64" ;;
        esac
        ;;
    Darwin)
        case "$ARCH" in
            x86_64) BINARY="traffic-sniff-darwin-amd64" ;;
            arm64) BINARY="traffic-sniff-darwin-arm64" ;;
        esac
        ;;
esac

# Check if binary exists
if [ ! -f "$BINARY" ]; then
    echo "Error: Binary $BINARY not found"
    echo "Available binaries:"
    ls -1 traffic-sniff*
    exit 1
fi

# Make executable
chmod +x "$BINARY"

# Default configuration
PORT=${PORT:-8080}
INTERFACE=${INTERFACE:-""}
STORAGE=${STORAGE:-"./data"}

echo "Starting Traffic Monitor..."
echo "Port: $PORT"
echo "Interface: $INTERFACE"
echo "Storage: $STORAGE"
echo "Binary: $BINARY"

# Start server
./"$BINARY" -port="$PORT" -interface="$INTERFACE" -storage="$STORAGE"
EOF

    cat > "$BUILD_DIR/start.bat" << 'EOF'
@echo off
REM Windows startup script for Traffic Monitor

set BINARY=traffic-sniff-windows-amd64.exe
set PORT=8080
set INTERFACE=
set STORAGE=.\data

if not exist "%BINARY%" (
    echo Error: Binary %BINARY% not found
    echo Available binaries:
    dir traffic-sniff*
    pause
    exit /b 1
)

echo Starting Traffic Monitor...
echo Port: %PORT%
echo Interface: %INTERFACE%
echo Storage: %STORAGE%
echo Binary: %BINARY%

REM Start server
"%BINARY%" -port=%PORT% -interface=%INTERFACE% -storage=%STORAGE%
pause
EOF

    # Make startup scripts executable
    chmod +x "$BUILD_DIR/start.sh"
    
    print_success "Distribution packaged"
}

# Create release archive
create_archive() {
    print_status "Creating release archive..."
    
    VERSION=$(date +"%Y%m%d-%H%M%S")
    ARCHIVE_NAME="traffic-monitor-$VERSION.tar.gz"
    
    tar -czf "$ARCHIVE_NAME" -C "$BUILD_DIR" .
    
    print_success "Release archive created: $ARCHIVE_NAME"
    
    # Show build summary
    echo ""
    echo "📦 Build Summary:"
    echo "=================="
    echo "Frontend: ✅ Built"
    echo "Backend:  ✅ Built (multiple platforms)"
    echo "Package:  ✅ $ARCHIVE_NAME"
    echo ""
    echo "📁 Build contents:"
    ls -la "$BUILD_DIR"
    echo ""
    echo "🎉 Build completed successfully!"
}

# Main build process
main() {
    echo "Traffic Monitor Build Script"
    echo "============================"
    
    check_dependencies
    clean_build
    build_frontend
    build_backend
    package_dist
    create_archive
}

# Handle script arguments
case "$1" in
    "clean")
        clean_build
        ;;
    "frontend")
        build_frontend
        ;;
    "backend")
        build_backend
        ;;
    "package")
        package_dist
        ;;
    *)
        main
        ;;
esac