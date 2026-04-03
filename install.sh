#!/bin/bash

echo "Installing go-dock..."

# Build the binary
go build -o go-dock main.go

# Move to system path so it works from anywhere
sudo mv go-dock /usr/local/bin/

# Create the data directories
mkdir -p ~/.go-dock/images
mkdir -p ~/.go-dock/containers

echo "Done! Try: go-dock pull alpine"
