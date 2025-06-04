import threading
import time
from flask import Flask, request
import socket

app = Flask(__name__)

# Set of active targets (tuple of IP and port)
targets = set()
targets_lock = threading.Lock()

def ping_target(ip, port, timeout=1):
    """Ping target via TCP socket"""
    try:
        with socket.create_connection((ip, int(port)), timeout=timeout):
            print(f"[✓] Pinged {ip}:{port}")
    except Exception as e:
        print(f"[✗] Failed to ping {ip}:{port} - {e}")

def pinger():
    """Background thread to continuously ping targets"""
    while True:
        with targets_lock:
            current_targets = list(targets)
        for ip, port in current_targets:
            ping_target(ip, port)
        time.sleep(2)  # Interval between pings

@app.route('/')
def handle_request():
    action = request.args.get('action')
    ip = request.args.get('ip')
    port = request.args.get('port')

    if not all([action, ip, port]):
        return "Missing required parameters: action, ip, port", 400

    target = (ip, int(port))

    with targets_lock:
        if action == 'open':
            targets.add(target)
            return f"Opened {ip}:{port}", 200
        elif action == 'close':
            targets.discard(target)
            return f"Closed {ip}:{port}", 200
        else:
            return f"Invalid action '{action}'", 400

if __name__ == '__main__':
    # Start the background ping loop
    threading.Thread(target=pinger, daemon=True).start()
    # Start the Flask server
    app.run(host='127.0.0.1', port=8181)

