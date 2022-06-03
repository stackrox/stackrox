#!/usr/bin/env python3

import sys
import socket

if len(sys.argv) != 2:
    print("Usage: get-hostname.py <IP>")
    sys.exit(1)

print(socket.getnameinfo((sys.argv[1], 0), 0)[0])
