import { group } from 'k6';
import http from 'k6/http';

export function configManagement(host, headers, tags) {
    group('configuration management', function () {
        http.batch([
            [
                'POST',
                `${host}/api/graphql?opname=numPolicies`,
                JSON.stringify({
                    operationName: 'numPolicies',
                    variables: { query: 'Lifecycle Stage:DEPLOY' },
                    query: `query numPolicies($query: String) { policyCount(query: $query) }`,
                }),
                { headers, tags },
            ],
            [
                'POST',
                `${host}/api/graphql?opname=policyViolationsBySeverity`,
                JSON.stringify({
                    operationName: 'policyViolationsBySeverity',
                    variables: { query: 'LifeCycle Stage:DEPLOY' },
                    query: `query policyViolationsBySeverity($query: String) {
  LOW_SEVERITY: violationCount(query: $query)
  MEDIUM_SEVERITY: violationCount(query: $query)
  HIGH_SEVERITY: violationCount(query: $query)
  CRITICAL_SEVERITY: violationCount(query: $query)
}`,
                }),
                { headers, tags },
            ],
            [
                'POST',
                `${host}/api/graphql?opname=secrets`,
                JSON.stringify({
                    operationName: 'secrets',
                    variables: {},
                    query: `query secrets { secretCount }`,
                }),
                { headers, tags },
            ],
        ]);

        http.post(
            `${host}/api/graphql?opname=policies`,
            JSON.stringify({
                operationName: 'policies',
                variables: { query: 'Lifecycle Stage:DEPLOY' },
                query: `query policies($query: String) {
  policies(query: $query) { id name severity __typename }
}`,
            }),
            { headers, tags }
        );

        http.post(
            `${host}/api/graphql?opname=serviceaccounts`,
            JSON.stringify({
                operationName: 'serviceaccounts',
                variables: {
                    pagination: {
                        offset: 0,
                        limit: 25,
                        sortOption: { field: 'Service Account', reversed: false },
                    },
                },
                query: `query serviceaccounts($pagination: Pagination) {
  serviceAccountCount
  results: serviceAccounts(pagination: $pagination) {
    id
    name
    namespace
    clusterName
    __typename
  }
}`,
            }),
            { headers, tags }
        );

        http.post(
            `${host}/api/graphql?opname=subjects`,
            JSON.stringify({
                operationName: 'subjects',
                variables: {
                    pagination: {
                        offset: 0,
                        limit: 25,
                        sortOption: { field: 'Subject', reversed: false },
                    },
                },
                query: `query subjects($pagination: Pagination) {
  subjectCount
  results: subjects(pagination: $pagination) {
    name
    type
    __typename
  }
}`,
            }),
            { headers, tags }
        );

        http.post(
            `${host}/api/graphql?opname=nodes`,
            JSON.stringify({
                operationName: 'nodes',
                variables: {
                    pagination: {
                        offset: 0,
                        limit: 25,
                        sortOption: { field: 'Node', reversed: false },
                    },
                },
                query: `query nodes($pagination: Pagination) {
  nodeCount
  results: nodes(pagination: $pagination) {
    id
    name
    clusterName
    __typename
  }
}`,
            }),
            { headers, tags }
        );
    });
}
