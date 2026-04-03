#!/bin/bash

echo "Installing go-dock..."

# 1. Install Go if not already installed
if ! command -v go &> /dev/null; then
    echo "Go not found, installing..."
    curl -OL https://go.dev/dl/go1.21.0.linux-amd64.tar.gz
    sudo tar -C /usr/local -xzf go1.21.0.linux-amd64.tar.gz
    export PATH=$PATH:/usr/local/go/bin
    echo "Go installed!"
fi

# 2. Build go-dock
go build -o go-dock main.go

# 3. Install system-wide
sudo mv go-dock /usr/local/bin/

# 4. Create data directories
mkdir -p ~/.go-dock/images
mkdir -p ~/.go-dock/containers

echo "Done! Try: go-dock pull alpine"
