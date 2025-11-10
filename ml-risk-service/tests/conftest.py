"""
Test configuration and fixtures.

This file helps avoid importing heavy dependencies like grpcio-tools during testing.
"""

import sys
from unittest.mock import MagicMock

# Mock grpc modules to avoid needing grpcio-tools for basic ML tests
sys.modules['grpc'] = MagicMock()
sys.modules['grpc_tools'] = MagicMock()
sys.modules['grpcio'] = MagicMock()
sys.modules['grpcio_tools'] = MagicMock()
