import { group } from 'k6';
import http from 'k6/http';

export function reports(host, headers, tags) {
    group('reports', function () {
        const configsResp = http.get(
            `${host}/v2/reports/configurations?pagination.offset=0&pagination.limit=10&pagination.sortOption.field=${encodeURIComponent('Report Name')}&pagination.sortOption.reversed=false`,
            { headers, tags }
        );

        http.get(
            `${host}/v2/reports/configuration-count`,
            { headers, tags }
        );

        try {
            const configs = JSON.parse(configsResp.body);
            const reportConfigs = configs.reportConfigs || [];
            for (const cfg of reportConfigs.slice(0, 3)) {
                http.get(
                    `${host}/v2/reports/configurations/${cfg.id}/my-history?reportParamQuery.pagination.offset=0&reportParamQuery.pagination.limit=1&reportParamQuery.pagination.sortOption.field=${encodeURIComponent('Report Completion Time')}&reportParamQuery.pagination.sortOption.reversed=true`,
                    { headers, tags }
                );
            }
        } catch (_) {}

        http.get(
            `${host}/v2/reports/view-based/history?reportParamQuery.pagination.offset=0&reportParamQuery.pagination.limit=10&reportParamQuery.pagination.sortOption.field=${encodeURIComponent('Report Completion Time')}&reportParamQuery.pagination.sortOption.reversed=true`,
            { headers, tags }
        );
    });
}
