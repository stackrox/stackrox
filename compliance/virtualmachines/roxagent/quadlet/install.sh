#!/bin/bash
# Install roxagent Quadlet units on a RHEL VM
#
# Usage:
#   ./install.sh                    # Install locally
#   ./install.sh user@host          # Install on remote host via SSH
#   ./install.sh user@host 2222     # Install on remote host with custom SSH port

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

install_locally() {
    echo "Installing Quadlet units locally..."

    # Quadlet container file
    sudo mkdir -p /etc/containers/systemd/
    sudo cp "${SCRIPT_DIR}/roxagent.container" /etc/containers/systemd/
    sudo restorecon -Rv /etc/containers/systemd/ 2>/dev/null || true

    # Timer and prep service go in standard systemd directory
    sudo cp "${SCRIPT_DIR}/roxagent.timer" /etc/systemd/system/
    sudo cp "${SCRIPT_DIR}/roxagent-prep.service" /etc/systemd/system/
    sudo restorecon -Rv /etc/systemd/system/roxagent.timer /etc/systemd/system/roxagent-prep.service 2>/dev/null || true

    echo "Reloading systemd..."
    sudo systemctl daemon-reload

    echo "Enabling and starting timer..."
    sudo systemctl enable --now roxagent.timer

    echo "Status:"
    sudo systemctl list-timers roxagent.timer
}

install_remote() {
    local REMOTE_HOST="$1"
    local SSH_PORT="${2:-22}"

    echo "Installing Quadlet units on ${REMOTE_HOST} (port ${SSH_PORT})..."

    # Copy files
    scp -P "${SSH_PORT}" "${SCRIPT_DIR}/roxagent.container" "${REMOTE_HOST}:/tmp/"
    scp -P "${SSH_PORT}" "${SCRIPT_DIR}/roxagent.timer" "${REMOTE_HOST}:/tmp/"
    scp -P "${SSH_PORT}" "${SCRIPT_DIR}/roxagent-prep.service" "${REMOTE_HOST}:/tmp/"

    # Install on remote
    ssh -p "${SSH_PORT}" "${REMOTE_HOST}" << 'EOF'
        # Quadlet container file
        sudo mkdir -p /etc/containers/systemd/
        sudo mv /tmp/roxagent.container /etc/containers/systemd/
        sudo restorecon -Rv /etc/containers/systemd/ 2>/dev/null || true

        # Timer and prep service go in standard systemd directory
        sudo mv /tmp/roxagent.timer /etc/systemd/system/
        sudo mv /tmp/roxagent-prep.service /etc/systemd/system/
        sudo restorecon -Rv /etc/systemd/system/roxagent.timer /etc/systemd/system/roxagent-prep.service 2>/dev/null || true

        echo "Reloading systemd..."
        sudo systemctl daemon-reload

        echo "Enabling and starting timer..."
        sudo systemctl enable --now roxagent.timer

        echo "Status:"
        sudo systemctl list-timers roxagent.timer
EOF
}

# Main
if [ $# -eq 0 ]; then
    install_locally
else
    install_remote "$1" "${2:-22}"
fi

echo ""
echo "Done! The roxagent will run hourly."
echo ""
echo "To run immediately:  sudo systemctl start roxagent.service"
echo "To view logs:        sudo journalctl -u roxagent.service -f"
echo "To check timer:      sudo systemctl list-timers roxagent.timer"
