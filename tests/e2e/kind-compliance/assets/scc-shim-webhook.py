#!/usr/bin/env python3
"""Minimal mutating webhook that emulates OpenShift SCC UID injection.

Only needed for the high-fidelity "content" mode: with PLATFORM=OpenShift the
Compliance Operator spawns profileparser/scanner pods from content/openscap images
that run as root and set runAsNonRoot without a runAsUser (OpenShift's SCC would
assign one). On kind there is no SCC, so the kubelet rejects them. This webhook
injects a non-root runAsUser/runAsGroup/fsGroup into pods in the watched namespace.
"""
import base64
import json
import ssl
from http.server import BaseHTTPRequestHandler, HTTPServer

UID = 1001


class Handler(BaseHTTPRequestHandler):
    def log_message(self, *a):
        pass

    def do_GET(self):
        self.send_response(200)
        self.end_headers()
        self.wfile.write(b"ok")

    def do_POST(self):
        review = json.loads(self.rfile.read(int(self.headers.get("Content-Length", 0))))
        req = review.get("request", {})
        spec = (req.get("object", {}) or {}).get("spec", {}) or {}
        sc = spec.get("securityContext", {}) or {}
        patch = []
        if sc.get("runAsUser") is None:
            if "securityContext" not in spec:
                patch.append({"op": "add", "path": "/spec/securityContext", "value": {}})
            patch.append({"op": "add", "path": "/spec/securityContext/runAsUser", "value": UID})
            if sc.get("runAsGroup") is None:
                patch.append({"op": "add", "path": "/spec/securityContext/runAsGroup", "value": UID})
            if sc.get("fsGroup") is None:
                patch.append({"op": "add", "path": "/spec/securityContext/fsGroup", "value": UID})
        resp = {"uid": req.get("uid", ""), "allowed": True}
        if patch:
            resp["patchType"] = "JSONPatch"
            resp["patch"] = base64.b64encode(json.dumps(patch).encode()).decode()
        body = json.dumps({
            "apiVersion": review.get("apiVersion", "admission.k8s.io/v1"),
            "kind": "AdmissionReview",
            "response": resp,
        }).encode()
        self.send_response(200)
        self.send_header("Content-Type", "application/json")
        self.end_headers()
        self.wfile.write(body)


if __name__ == "__main__":
    ctx = ssl.SSLContext(ssl.PROTOCOL_TLS_SERVER)
    ctx.load_cert_chain("/tls/tls.crt", "/tls/tls.key")
    srv = HTTPServer(("0.0.0.0", 8443), Handler)
    srv.socket = ctx.wrap_socket(srv.socket, server_side=True)
    srv.serve_forever()
