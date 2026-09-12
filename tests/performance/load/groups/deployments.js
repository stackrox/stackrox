import { group } from 'k6';
import http from 'k6/http';

export function deployments(host, headers, tags) {
    group('deployments', function () {
        http.batch([
            [
                'GET',
                `${host}/v1/deploymentswithprocessinfo?pagination.offset=0&pagination.limit=50&pagination.sortOption.field=${encodeURIComponent('Deployment')}&pagination.sortOption.reversed=false&query=${encodeURIComponent('Platform Component:false')}`,
                null,
                { headers, tags },
            ],
            [
                'GET',
                `${host}/v1/deploymentscount?query=${encodeURIComponent('Platform Component:false')}`,
                null,
                { headers, tags },
            ],
        ]);

        http.post(
            `${host}/api/graphql?opname=searchOptions`,
            JSON.stringify({
                operationName: 'searchOptions',
                variables: { categories: ['DEPLOYMENTS'] },
                query: `query searchOptions($categories: [SearchCategory!]) {
  searchOptions(categories: $categories)
}`,
            }),
            { headers, tags }
        );
    });
}
