import { group } from 'k6';
import http from 'k6/http';

export function clustersAndPolicies(host, headers, tags) {
    group('clusters and policies', function () {
        http.batch([
            ['GET', `${host}/v1/clusters`, null, { headers, tags }],
            ['GET', `${host}/v1/policies`, null, { headers, tags }],
        ]);

        http.post(
            `${host}/api/graphql?opname=autocomplete`,
            JSON.stringify({
                operationName: 'autocomplete',
                variables: { query: '', categories: 'CLUSTERS' },
                query: `query autocomplete($query: String!, $categories: [SearchCategory!]) {
  searchAutocomplete(query: $query, categories: $categories)
}`,
            }),
            { headers, tags }
        );
    });
}
