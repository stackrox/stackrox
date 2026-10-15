"""Real kubectl proxy/HTTP transport checks against a local fake API.

Opt in with NODE_WATCH_TRANSPORT_TESTS=true; these tests bind a local socket
and require kubectl, but never access a real cluster.
"""

from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
import json
import os
from pathlib import Path
import shutil
import sys
import tempfile
import threading
import unittest
from unittest.mock import patch
from urllib.parse import parse_qs, urlsplit
import xml.etree.ElementTree as ET

import node_disruption as monitor


class Handler(BaseHTTPRequestHandler):
    protocol_version = 'HTTP/1.1'

    def log_message(self, *_args):
        pass

    def do_GET(self):  # pylint: disable=invalid-name
        parsed = urlsplit(self.path)
        self.server.requests.append(self.path)
        if parsed.path != '/api/v1/nodes':
            self.send_error(404)
            return
        if 'watch' not in parse_qs(parsed.query):
            metadata = {} if self.server.mode == 'missing-version' else {'resourceVersion': 'list-123'}
            data = {'apiVersion': 'v1', 'kind': 'NodeList', 'metadata': metadata,
                    'items': [{'metadata': {'name': 'worker-1', 'resourceVersion': 'node-99'},
                               'status': {'conditions': [{'type': 'Ready', 'status': 'True'}]}}]}
            body = json.dumps(data).encode()
            self.send_response(200)
            self.send_header('Content-Type', 'application/json')
            self.send_header('Content-Length', str(len(body)))
            self.end_headers()
            self.wfile.write(body)
            return
        if self.server.mode == 'forbidden':
            self.send_error(403, 'Forbidden')
            return
        self.send_response(200)
        self.send_header('Content-Type', 'application/json')
        self.send_header('Transfer-Encoding', 'chunked')
        self.end_headers()
        self.wfile.flush()
        if self.server.mode in ('deletion', 'readiness', 'eof'):
            while not self.server.product_marker.exists():
                if self.server.done.wait(0.01):
                    return
            try:
                if self.server.mode == 'eof':
                    self.wfile.write(b'0\r\n\r\n')
                    self.wfile.flush()
                    return
                event = {'type': 'DELETED' if self.server.mode == 'deletion' else 'MODIFIED',
                         'object': {'metadata': {'name': 'worker-1'}, 'status': {
                             'conditions': [{'type': 'Ready', 'status': 'False'}]}}}
                added = {'type': 'ADDED', 'object': {'metadata': {'name': 'new-worker'}}}
                body = (json.dumps(added) + '\n' + json.dumps(event) + '\n').encode()
                # Split a JSON event across HTTP chunks; combine multiple
                # watch events in the same underlying stream.
                for chunk in (body[:7], body[7:]):
                    self.wfile.write(f'{len(chunk):x}\r\n'.encode() + chunk + b'\r\n')
                    self.wfile.flush()
            except (BrokenPipeError, ConnectionResetError):
                return
        self.server.done.wait(10)


@unittest.skipUnless(os.getenv('NODE_WATCH_TRANSPORT_TESTS') == 'true' and shutil.which('kubectl'),
                     'requires explicit local-socket transport test opt-in and kubectl')
class TransportTest(unittest.TestCase):
    def exercise(self, mode, product_code=0):
        done = threading.Event()

        server = ThreadingHTTPServer(('127.0.0.1', 0), Handler)
        server.daemon_threads = True
        server.mode = mode
        server.done = done
        server.requests = []
        worker = threading.Thread(target=server.serve_forever, daemon=True)
        worker.start()
        try:
            with tempfile.TemporaryDirectory() as directory:
                config = Path(directory) / 'config'
                marker = Path(directory) / 'product-started'
                server.product_marker = marker
                config.write_text(json.dumps({
                    'apiVersion': 'v1', 'kind': 'Config', 'current-context': 'test',
                    'clusters': [{'name': 'test', 'cluster': {'server':
                                 f'http://127.0.0.1:{server.server_port}'}}],
                    'contexts': [{'name': 'test', 'context': {'cluster': 'test', 'user': 'test'}}],
                    'users': [{'name': 'test', 'user': {}}],
                }))
                sleep = 0.4 if mode == 'healthy' else 30
                script = (f'from pathlib import Path; import time; Path({str(marker)!r}).touch(); '
                          f'time.sleep({sleep}); raise SystemExit({product_code})')
                with patch.dict(os.environ, KUBECONFIG=str(config), ARTIFACT_DIR=directory):
                    result = monitor.run([sys.executable, '-c', script])
                report = ET.parse(Path(directory) / 'node-monitor.xml').getroot()
                started = marker.exists()
            if mode != 'missing-version':
                watch = [parse_qs(urlsplit(path).query) for path in server.requests if '?' in path]
                self.assertEqual(watch[0]['resourceVersion'], ['list-123'])
            self.assertEqual(len(report.findall('testcase')), 1)
            self.assertEqual(report.find('testcase').get('classname'), 'ci.gke.node')
            self.assertEqual(report.find('testcase').get('name'), 'NoNodeDisruption')
            return result, report, started
        finally:
            done.set()
            server.shutdown()
            server.server_close()
            worker.join()

    def test_healthy_monitor_preserves_product_failure(self):
        result, report, started = self.exercise('healthy', product_code=42)
        self.assertTrue(started)
        self.assertEqual(result.exit_code, 42)
        self.assertFalse(result.monitor_failed)
        self.assertEqual(report.get('failures'), '0')

    def test_disruptions_and_eof_fail_closed(self):
        for mode in ('deletion', 'readiness', 'eof'):
            with self.subTest(mode=mode):
                result, report, started = self.exercise(mode)
                self.assertTrue(started)
                self.assertTrue(result.monitor_failed)
                self.assertEqual(report.get('failures'), '1')

    def test_establishment_failure_prevents_product_start(self):
        for mode in ('missing-version', 'forbidden'):
            with self.subTest(mode=mode):
                result, report, started = self.exercise(mode)
                self.assertFalse(started)
                self.assertTrue(result.monitor_failed)
                self.assertEqual(report.get('failures'), '1')
