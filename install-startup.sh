#!/bin/bash

# Script to install systemd service for auto-starting registry services

echo "Installing systemd service for registry services..."

# Copy the service file to systemd directory
sudo cp /home/pluggedin/registry/registry-services.service /etc/systemd/system/

# Reload systemd daemon
sudo systemctl daemon-reload

# Enable the service to start at boot
sudo systemctl enable registry-services.service

# Start the service now
sudo systemctl start registry-services.service

# Check status
sudo systemctl status registry-services.service

echo "Installation complete!"
echo ""
echo "To manage the services:"
echo "  Start:   sudo systemctl start registry-services"
echo "  Stop:    sudo systemctl stop registry-services"
echo "  Status:  sudo systemctl status registry-services"
echo "  Disable: sudo systemctl disable registry-services"