import { group } from 'k6';
import http from 'k6/http';

export function violations(host, headers, tags) {
    group('violations', function () {
        http.batch([
            [
                'GET',
                `${host}/v1/alerts?query=${encodeURIComponent('Platform Component:false+Entity Type:DEPLOYMENT+Violation State:ACTIVE')}&pagination.offset=0&pagination.limit=50&pagination.sortOption.field=${encodeURIComponent('Violation Time')}&pagination.sortOption.reversed=true`,
                null,
                { headers, tags },
            ],
            [
                'GET',
                `${host}/v1/alertscount?query=${encodeURIComponent('Platform Component:false+Entity Type:DEPLOYMENT+Violation State:ACTIVE')}`,
                null,
                { headers, tags },
            ],
        ]);

        http.post(
            `${host}/api/graphql?opname=autocomplete`,
            JSON.stringify({
                operationName: 'autocomplete',
                variables: {
                    query: 'Platform Component:false+Entity Type:DEPLOYMENT+Policy:',
                    categories: 'ALERTS',
                },
                query: `query autocomplete($query: String!, $categories: [SearchCategory!]) {
  searchAutocomplete(query: $query, categories: $categories)
}`,
            }),
            { headers, tags }
        );

        http.post(
            `${host}/api/graphql?opname=getDeploymentCount`,
            JSON.stringify({
                operationName: 'getDeploymentCount',
                variables: { query: '' },
                query: `query getDeploymentCount($query: String) {
  deploymentCount(query: $query)
}`,
            }),
            { headers, tags }
        );

        http.get(`${host}/v1/sac/clusters?permissions=Deployment`, {
            headers,
            tags,
        });
    });
}
