import componentTypes from 'constants/componentTypes';
import entityTypes, { standardTypes } from 'constants/entityTypes';
import contextTypes from 'constants/contextTypes';
import pageTypes from 'constants/pageTypes';
import { CLUSTER_QUERY } from 'queries/cluster';
import { NAMESPACE_QUERY, RELATED_DEPLOYMENTS } from 'queries/namespace';
import { CLUSTERS_QUERY, NAMESPACES_QUERY, NODES_QUERY } from 'queries/table';
import { NODE_QUERY } from 'queries/node';
import AGGREGATED_RESULTS from 'queries/controls';
import { LIST_STANDARD, ENTITY_COMPLIANCE } from 'queries/standard';

import pluralize from 'pluralize';

/**
 * context:     Array of contextTypes to match
 * pageType:    Array of pageTypes to match
 * entityType:  Array of entityTypes to match
 * config:      Contains information about the query
 *      query:              GraphQL query text
 *      variables:          A mapping of GraphQL parameter names to URL/Query parameter names used byApp Query to pass url params to the query.
 *                          e.g.:
 *                              URL       =   /:entityType/:entityId
 *                              GraphQL   =   query getCluster($id: ID!)
 *                              variables =   [{ graphQLParam: 'id', queryParam: 'entityId' }]
 *                          or  variables =   [{ graphQLParam: 'groupBy', graphQLValue: ['STANDARD', 'CLUSTER'] }]
 *                          or  variables =   [{ graphQLParam: 'groupBy', paramsFunc: params => ['STANDARD', params.entityType] }]
 *      format:             A function run on the result set before returning.
 *      bypassCache:        A boolean that tells the Query whether to bypass the cache
 */

const isStandard = type => Object.keys(standardTypes).includes(type);

function getSubField(data, path) {
    const fields = path.split('.');
    if (!data) return null;
    let subfield = data;
    for (let i = 0; i < fields.length; i += 1) {
        subfield = subfield[fields[i]];
        if (!subfield) return data;
    }
    return subfield;
}
export default [
    {
        context: [contextTypes.COMPLIANCE],
        pageType: [pageTypes.ENTITY],
        entityType: [entityTypes.CLUSTERS],
        component: [componentTypes.HEADER],
        config: {
            query: CLUSTER_QUERY,
            variables: [{ graphQLParam: 'id', queryParam: 'entityId' }]
        }
    },
    {
        context: [contextTypes.COMPLIANCE],
        pageType: [pageTypes.ENTITY],
        entityType: [entityTypes.NAMESPACES],
        component: [componentTypes.HEADER],
        config: {
            query: NAMESPACE_QUERY,
            variables: [{ graphQLParam: 'id', queryParam: 'entityId' }]
        }
    },
    {
        context: [contextTypes.COMPLIANCE],
        pageType: [pageTypes.ENTITY],
        entityType: [entityTypes.NODES],
        component: [componentTypes.HEADER],
        config: {
            query: NODE_QUERY,
            variables: [{ graphQLParam: 'id', queryParam: 'entityId' }]
        }
    },
    {
        context: [contextTypes.COMPLIANCE],
        pageType: [pageTypes.ENTITY],
        entityType: [entityTypes.CLUSTERS],
        component: [componentTypes.RELATED_ENTITIES_LIST],
        config: {
            query: RELATED_DEPLOYMENTS,
            variables: [{ graphQLParam: 'id', queryParam: 'entityId' }]
        }
    },
    {
        context: [contextTypes.COMPLIANCE],
        pageType: [pageTypes.ENTITY],
        entityType: [entityTypes.CLUSTERS, entityTypes.NODES, entityTypes.NAMESPACES],
        component: [componentTypes.ENTITY_COMPLIANCE],
        config: {
            query: ENTITY_COMPLIANCE,
            variables: [
                { graphQLParam: 'id', queryParam: 'entityId' },
                { graphQLParam: 'entityType', queryParam: 'entityType' }
            ],
            format(data) {
                const formattedData = {
                    ...data
                };
                formattedData.results = getSubField(data, 'aggregatedResults.results.results');
                return formattedData;
            }
        }
    },
    {
        context: [contextTypes.COMPLIANCE],
        pageType: [pageTypes.LIST],
        entityType: [entityTypes.CLUSTERS],
        component: [componentTypes.LIST_TABLE],
        config: {
            query: CLUSTERS_QUERY,
            variables: [{ graphQLParam: 'id', queryParam: 'entityId' }]
        }
    },
    {
        context: [contextTypes.COMPLIANCE],
        pageType: [pageTypes.LIST],
        entityType: [entityTypes.NAMESPACES],
        component: [componentTypes.LIST_TABLE],
        config: {
            query: NAMESPACES_QUERY,
            variables: [{ graphQLParam: 'id', queryParam: 'entityId' }]
        }
    },
    {
        context: [contextTypes.COMPLIANCE],
        pageType: [pageTypes.LIST],
        entityType: [entityTypes.NODES],
        component: [componentTypes.LIST_TABLE],
        config: {
            query: NODES_QUERY,
            variables: [{ graphQLParam: 'id', queryParam: 'entityId' }]
        }
    },
    {
        context: [contextTypes.COMPLIANCE],
        pageType: [pageTypes.LIST],
        entityType: [
            entityTypes.PCI_DSS_3_2,
            entityTypes.NIST_800_190,
            entityTypes.HIPAA_164,
            entityTypes.CIS_DOCKER_V1_1_0,
            entityTypes.CIS_KUBERENETES_V1_2_0
        ],
        component: [componentTypes.LIST_TABLE],
        config: {
            query: LIST_STANDARD,
            variables: [{ graphQLParam: 'where', queryParam: 'entityType' }],
            format(data) {
                if (!data.results) return null;
                const formattedData = { results: [] };
                const groups = {};
                data.results.forEach(({ keys, numPassing, numFailing }) => {
                    const { name, groupId, description } = keys[1];
                    if (!groups[groupId]) {
                        groups[groupId] = {
                            name: `${name} ${description}`,
                            rows: []
                        };
                    }
                    const compliance =
                        numPassing + numFailing === 0
                            ? '0%'
                            : `${(numPassing / (numPassing + numFailing)).toFixed(2) * 100}%`;
                    groups[groupId].rows.push({
                        control: `${name} - ${description}`,
                        compliance,
                        group: groupId
                    });
                });
                Object.keys(groups).forEach(groupId => {
                    formattedData.results.push(groups[groupId]);
                });
                return formattedData;
            }
        }
    },
    {
        context: [contextTypes.COMPLIANCE],
        pageType: [pageTypes.DASHBOARD],
        entityType: [],
        component: [componentTypes.STANDARDS_BY_CLUSTER],
        config: {
            query: AGGREGATED_RESULTS,
            variables: [
                { graphQLParam: 'groupBy', graphQLValue: ['STANDARD', 'CLUSTER'] },
                { graphQLParam: 'unit', graphQLValue: 'CONTROL' }
            ],
            format(data) {
                const formattedData = {
                    results: data.results,
                    complianceStandards: data.complianceStandards,
                    entityList: data.clusters
                };
                return formattedData;
            }
        }
    },
    {
        context: [contextTypes.COMPLIANCE],
        pageType: [pageTypes.DASHBOARD],
        entityType: [],
        component: [componentTypes.STANDARDS_ACROSS_CLUSTERS],
        config: {
            query: AGGREGATED_RESULTS,
            variables: [
                { graphQLParam: 'groupBy', graphQLValue: ['STANDARD'] },
                { graphQLParam: 'unit', graphQLValue: 'CLUSTER' }
            ]
        }
    },
    {
        context: [contextTypes.COMPLIANCE],
        pageType: [pageTypes.DASHBOARD],
        entityType: [],
        component: [componentTypes.STANDARDS_ACROSS_NAMESPACES],
        config: {
            query: AGGREGATED_RESULTS,
            variables: [
                { graphQLParam: 'groupBy', graphQLValue: ['STANDARD'] },
                { graphQLParam: 'unit', graphQLValue: 'NAMESPACE' }
            ]
        }
    },
    {
        context: [contextTypes.COMPLIANCE],
        pageType: [pageTypes.DASHBOARD],
        entityType: [],
        component: [componentTypes.STANDARDS_ACROSS_NODES],
        config: {
            query: AGGREGATED_RESULTS,
            variables: [
                { graphQLParam: 'groupBy', graphQLValue: ['STANDARD'] },
                { graphQLParam: 'unit', graphQLValue: 'NODE' }
            ]
        }
    },
    {
        context: [contextTypes.COMPLIANCE],
        pageType: [pageTypes.LIST],
        entityType: [],
        component: [
            componentTypes.COMPLIANCE_ACROSS_RESOURCES,
            componentTypes.COMPLIANCE_ACROSS_STANDARDS
        ],
        config: {
            query: AGGREGATED_RESULTS,
            variables: [
                { graphQLParam: 'groupBy', graphQLValue: ['STANDARD'] },
                {
                    graphQLParam: 'unit',
                    paramsFunc: ({ entityType }) => {
                        if (isStandard(entityType)) return 'CONTROL';
                        return pluralize.singular(entityType).toUpperCase();
                    }
                }
            ],
            bypassCache: true
        }
    }
];
