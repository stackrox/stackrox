import { describe, expect, it } from 'vitest';

import { parsePrometheusMetrics } from './prometheusParser';

describe('parsePrometheusMetrics', () => {
    it('should parse empty Prometheus metrics', () => {
        const metrics = parsePrometheusMetrics('');
        expect(metrics).toStrictEqual({
            metricInfoMap: {},
            metrics: {},
            parseErrors: [],
        });
    });
    it('should parse Prometheus metrics', () => {
        const metrics = parsePrometheusMetrics(
            '# TYPE rox_central_api_request_total counter\n' +
                '# HELP rox_central_api_request_total API requests\n' +
                'rox_central_api_request_total{Method="GET",Path="/",Status="200",UserAgent="Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/144.0.0.0 Safari/537.36",UserID=""} 3\n' +
                'rox_central_api_request_total{Method="GET",Path="/",Status="200",UserAgent="Mozilla/5.0 zgrab/0.x",UserID=""} 1 1395066363000\n' +
                'rox_central_api_request_total{Method="GET",Path="/",Status="200",UserAgent="python-requests/2.32.5",UserID=""} 1\n' +
                '# TYPE rox_central_image_vuln_deployment_severity gauge\n' +
                '# HELP rox_central_image_vuln_deployment_severity The total number of image vulnerabilities aggregated by Cluster,Deployment,IsFixable,IsPlatformWorkload,Namespace,Severity and gathered every 16m0s\n' +
                'rox_central_image_vuln_deployment_severity{Cluster="production",Deployment="api-server",IsFixable="true",IsPlatformWorkload="false",Namespace="backend",Severity="CRITICAL_VULNERABILITY_SEVERITY"} 2\n'
        );
        expect(metrics).toStrictEqual({
            metricInfoMap: {
                rox_central_api_request_total: 'API requests',
                rox_central_image_vuln_deployment_severity:
                    'The total number of image vulnerabilities aggregated by Cluster,Deployment,IsFixable,IsPlatformWorkload,Namespace,Severity and gathered every 16m0s',
            },
            metrics: {
                rox_central_api_request_total: [
                    {
                        labels: {
                            Method: 'GET',
                            Path: '/',
                            Status: '200',
                            UserAgent:
                                'Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/144.0.0.0 Safari/537.36',
                            UserID: '',
                        },
                        timestamp: undefined,
                        value: '3',
                    },
                    {
                        labels: {
                            Method: 'GET',
                            Path: '/',
                            Status: '200',
                            UserAgent: 'Mozilla/5.0 zgrab/0.x',
                            UserID: '',
                        },
                        timestamp: 1395066363000,
                        value: '1',
                    },
                    {
                        labels: {
                            Method: 'GET',
                            Path: '/',
                            Status: '200',
                            UserAgent: 'python-requests/2.32.5',
                            UserID: '',
                        },
                        timestamp: undefined,
                        value: '1',
                    },
                ],
                rox_central_image_vuln_deployment_severity: [
                    {
                        labels: {
                            Cluster: 'production',
                            Deployment: 'api-server',
                            IsFixable: 'true',
                            IsPlatformWorkload: 'false',
                            Namespace: 'backend',
                            Severity: 'CRITICAL_VULNERABILITY_SEVERITY',
                        },
                        timestamp: undefined,
                        value: '2',
                    },
                ],
            },
            parseErrors: [],
        });
    });
    it('should parse no metrics', () => {
        const metrics = parsePrometheusMetrics(
            '# TYPE rox_central_api_request_total counter\n' +
                '# HELP rox_central_api_request_total API requests\n' +
                '# TYPE rox_central_image_vuln_deployment_severity gauge\n' +
                '# HELP rox_central_image_vuln_deployment_severity The total number of image vulnerabilities aggregated by Cluster,Deployment,IsFixable,IsPlatformWorkload,Namespace,Severity and gathered every 16m0s\n'
        );
        expect(metrics).toStrictEqual({
            metricInfoMap: {
                rox_central_api_request_total: 'API requests',
                rox_central_image_vuln_deployment_severity:
                    'The total number of image vulnerabilities aggregated by Cluster,Deployment,IsFixable,IsPlatformWorkload,Namespace,Severity and gathered every 16m0s',
            },
            metrics: {
                rox_central_api_request_total: [],
                rox_central_image_vuln_deployment_severity: [],
            },
            parseErrors: [],
        });
    });
    it('should collect parse errors for invalid lines', () => {
        const metrics = parsePrometheusMetrics(
            '# HELP valid_metric A valid metric\n' +
                'valid_metric 42\n' +
                '123invalid_starting_with_number 10\n' +
                '@invalid_special_char 20\n' +
                'valid_metric2 100\n'
        );
        expect(metrics.parseErrors).toStrictEqual([
            { line: '123invalid_starting_with_number 10', lineNumber: 3 },
            { line: '@invalid_special_char 20', lineNumber: 4 },
        ]);
        expect(metrics.metrics).toHaveProperty('valid_metric');
        expect(metrics.metrics).toHaveProperty('valid_metric2');
    });
});
