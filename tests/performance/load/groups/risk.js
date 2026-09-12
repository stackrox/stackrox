import { group } from 'k6';
import http from 'k6/http';

export function risk(host, headers, tags) {
    group('risk', function () {
        http.batch([
            [
                'GET',
                `${host}/v1/deploymentswithprocessinfo?pagination.offset=0&pagination.limit=50&pagination.sortOption.field=${encodeURIComponent('Deployment Risk Priority')}&pagination.sortOption.reversed=false&query=${encodeURIComponent('Platform Component:false')}`,
                null,
                { headers, tags },
            ],
            [
                'GET',
                `${host}/v1/deploymentscount`,
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
