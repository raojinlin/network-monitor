# Build Notes for Traffic Monitor

## Cross-Platform Compilation Limitations

### The CGO Challenge

Traffic Monitor uses the `gopacket` library with `pcap` for packet capture. This requires CGO (C bindings) because:

1. **libpcap dependency**: The packet capture functionality relies on libpcap (Linux/Mac) or WinPcap/Npcap (Windows)
2. **System-specific APIs**: Network interface access requires platform-specific system calls
3. **CGO requirement**: Go's pcap bindings need CGO enabled to link with the C libraries

### Why Cross-Compilation Doesn't Work

When cross-compiling (building for a different OS/architecture), you would need:
- Target platform's C compiler
- Target platform's libpcap headers and libraries
- Proper CGO environment configuration

This is complex and error-prone, so we've simplified the build process.

## Building for Different Platforms

### Option 1: Build on Target Platform (Recommended)

Build directly on the platform you want to run on:

```bash
# On Linux
./build.sh

# On macOS
./build.sh

# On Windows
build.bat
```

### Option 2: Use Docker for Linux Builds

Docker can build Linux binaries on any platform:

```bash
# Build Linux binary using Docker
docker build -t traffic-monitor .

# Extract the binary
docker create --name temp traffic-monitor
docker cp temp:/app/traffic-sniff ./traffic-sniff-linux
docker rm temp
```

### Option 3: GitHub Actions

Use CI/CD to build on multiple platforms:

1. Push code to GitHub
2. GitHub Actions will build on Linux, macOS, and Windows runners
3. Download artifacts from the Actions tab

## Platform-Specific Requirements

### Linux
```bash
# Debian/Ubuntu
sudo apt-get install libpcap-dev

# RHEL/CentOS/Fedora
sudo yum install libpcap-devel

# Alpine
sudo apk add libpcap-dev
```

### macOS
- libpcap is included with macOS
- May need to run with `sudo` for packet capture

### Windows
- Install [Npcap](https://npcap.com/) or WinPcap
- Run as Administrator for packet capture

## Build Troubleshooting

### "undefined: pcapErrorNotActivated" Error
This occurs when trying to cross-compile. Solution: Build on the target platform.

### "permission denied" During Packet Capture
Run with elevated privileges:
```bash
sudo ./traffic-sniff -port=8080
```

### Missing libpcap
Install the appropriate libpcap development package for your platform (see Platform-Specific Requirements).

## Alternative: Remove Packet Capture Feature

If you need cross-platform builds without packet capture, you could:

1. Create a build tag for packet capture features
2. Build without packet capture for easy distribution
3. Add packet capture as an optional plugin

This would require refactoring the code to make packet capture optional.

## Docker Multi-Architecture Builds

For Linux targets, you can use Docker buildx for multi-arch builds:

```bash
# Setup buildx
docker buildx create --name multiarch --use

# Build for multiple architectures
docker buildx build --platform linux/amd64,linux/arm64 -t traffic-monitor:latest .
```

## Summary

- **Current approach**: Build on each target platform
- **Best practice**: Use CI/CD (GitHub Actions) for automated multi-platform builds
- **For development**: Use the build scripts on your development machine
- **For production**: Use Docker or platform-specific builds

The limitation is due to the low-level network access requirements, which is common for network monitoring tools.