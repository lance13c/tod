#!/bin/bash

# Start a simple HTTP server on port 5000 for testing
echo "Starting test server on http://localhost:5000"
echo "Serving test_server.html"

python3 -c "
import http.server
import socketserver
import os

os.chdir('$(dirname "$0")')

class MyHTTPRequestHandler(http.server.SimpleHTTPRequestHandler):
    def do_GET(self):
        if self.path == '/' or self.path == '/index.html':
            self.path = '/test_server.html'
        elif self.path in ['/login', '/dashboard']:
            self.path = '/test_server.html'
        return super().do_GET()

PORT = 5000
with socketserver.TCPServer(('', PORT), MyHTTPRequestHandler) as httpd:
    print(f'Server running at http://localhost:{PORT}')
    httpd.serve_forever()
"