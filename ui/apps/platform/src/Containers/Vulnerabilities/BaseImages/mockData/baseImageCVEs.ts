import { BaseImageCVE } from '../types';

/**
 * Mock CVE data for each tracked base image
 */
export const MOCK_BASE_IMAGE_CVES: Record<string, BaseImageCVE[]> = {
    'base-image-1': [
        // ubuntu:22.04
        {
            cveId: 'CVE-2024-1234',
            severity: 'CRITICAL',
            cvssScore: 9.8,
            summary: 'Buffer overflow in libssl allowing remote code execution',
            fixedBy: '1.2.3-4ubuntu1',
            components: [{ name: 'libssl1.1', version: '1.2.3-3ubuntu1', layerIndex: 2 }],
        },
        {
            cveId: 'CVE-2024-5678',
            severity: 'HIGH',
            cvssScore: 7.5,
            summary: 'SQL injection vulnerability in sqlite3',
            fixedBy: '3.37.2-2ubuntu1',
            components: [{ name: 'sqlite3', version: '3.37.2-1ubuntu1', layerIndex: 1 }],
        },
        {
            cveId: 'CVE-2024-2345',
            severity: 'CRITICAL',
            cvssScore: 9.1,
            summary: 'Remote code execution in glibc DNS resolver',
            fixedBy: '2.35-0ubuntu4',
            components: [{ name: 'libc6', version: '2.35-0ubuntu3', layerIndex: 0 }],
        },
        {
            cveId: 'CVE-2024-3456',
            severity: 'HIGH',
            cvssScore: 8.2,
            summary: 'Privilege escalation in systemd',
            fixedBy: '249.11-0ubuntu4',
            components: [{ name: 'systemd', version: '249.11-0ubuntu3', layerIndex: 3 }],
        },
        {
            cveId: 'CVE-2024-4567',
            severity: 'CRITICAL',
            cvssScore: 9.4,
            summary: 'Authentication bypass in OpenSSH',
            fixedBy: '1:8.9p1-3ubuntu2',
            components: [{ name: 'openssh-server', version: '1:8.9p1-3ubuntu1', layerIndex: 4 }],
        },
        {
            cveId: 'CVE-2024-5679',
            severity: 'HIGH',
            cvssScore: 7.8,
            summary: 'Directory traversal in tar utility',
            fixedBy: '1.34+dfsg-1ubuntu1',
            components: [{ name: 'tar', version: '1.34+dfsg-1ubuntu0', layerIndex: 2 }],
        },
        {
            cveId: 'CVE-2024-6789',
            severity: 'MEDIUM',
            cvssScore: 5.5,
            summary: 'Information disclosure in bash',
            fixedBy: '5.1-6ubuntu2',
            components: [{ name: 'bash', version: '5.1-6ubuntu1', layerIndex: 1 }],
        },
    ],
    'base-image-2': [
        // alpine:3.18
        {
            cveId: 'CVE-2024-9999',
            severity: 'HIGH',
            cvssScore: 8.1,
            summary: 'Privilege escalation in busybox',
            fixedBy: '1.36.1-r2',
            components: [{ name: 'busybox', version: '1.36.1-r1', layerIndex: 0 }],
        },
        {
            cveId: 'CVE-2024-8888',
            severity: 'HIGH',
            cvssScore: 7.3,
            summary: 'Command injection in apk-tools',
            fixedBy: '2.14.0-r3',
            components: [{ name: 'apk-tools', version: '2.14.0-r2', layerIndex: 1 }],
        },
        {
            cveId: 'CVE-2024-7777',
            severity: 'HIGH',
            cvssScore: 7.0,
            summary: 'Buffer overflow in musl libc',
            fixedBy: '1.2.4-r2',
            components: [{ name: 'musl', version: '1.2.4-r1', layerIndex: 0 }],
        },
        {
            cveId: 'CVE-2024-6666',
            severity: 'MEDIUM',
            cvssScore: 6.5,
            summary: 'Denial of service in ssl_client',
            fixedBy: '1.36.1-r2',
            components: [{ name: 'ssl_client', version: '1.36.1-r1', layerIndex: 2 }],
        },
    ],
    'base-image-3': [
        // node:18-alpine (IN_PROGRESS, no CVEs yet)
    ],
    'base-image-4': [
        // nginx:1.25-alpine
        {
            cveId: 'CVE-2024-5555',
            severity: 'CRITICAL',
            cvssScore: 9.0,
            summary: 'HTTP request smuggling in nginx',
            fixedBy: '1.25.3-r0',
            components: [{ name: 'nginx', version: '1.25.2-r0', layerIndex: 3 }],
        },
        {
            cveId: 'CVE-2024-4444',
            severity: 'CRITICAL',
            cvssScore: 8.8,
            summary: 'Path traversal in nginx static file serving',
            fixedBy: '1.25.3-r0',
            components: [{ name: 'nginx', version: '1.25.2-r0', layerIndex: 3 }],
        },
        {
            cveId: 'CVE-2024-3333',
            severity: 'HIGH',
            cvssScore: 7.5,
            summary: 'DoS vulnerability in OpenSSL',
            fixedBy: '3.1.4-r1',
            components: [{ name: 'libssl3', version: '3.1.4-r0', layerIndex: 2 }],
        },
        {
            cveId: 'CVE-2024-2222',
            severity: 'HIGH',
            cvssScore: 7.2,
            summary: 'Memory corruption in zlib',
            fixedBy: '1.3-r1',
            components: [{ name: 'zlib', version: '1.3-r0', layerIndex: 1 }],
        },
    ],
};
