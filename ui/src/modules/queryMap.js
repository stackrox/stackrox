import componentTypes from 'constants/componentTypes';
import entityTypes, { standardTypes } from 'constants/entityTypes';
import standardLabels from 'messages/standards';
import contextTypes from 'constants/contextTypes';
import pageTypes from 'constants/pageTypes';
import { CLUSTER_QUERY } from 'queries/cluster';
import { NAMESPACE_QUERY, RELATED_DEPLOYMENTS } from 'queries/namespace';
import { CLUSTERS_LIST_QUERY, NAMESPACES_LIST_QUERY, NODES_QUERY } from 'queries/table';
import { NODE_QUERY } from 'queries/node';
import { AGGREGATED_RESULTS } from 'queries/controls';
import { LIST_STANDARD, COMPLIANCE_STANDARDS } from 'queries/standard';

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
const complianceRate = (numPassing, numFailing) =>
    numPassing + numFailing === 0
        ? '0%'
        : `${((numPassing / (numPassing + numFailing)) * 100).toFixed(2)}%`;

const formatComplianceTableData = (data, entityType) => {
    if (!data.results || data.results.results.length === 0) return null;
    const formattedData = { results: [] };
    const entityMap = {};
    let standardKeyIndex = 0;
    let entityKeyIndex = 0;
    data.results.results[0].aggregationKeys.forEach((key, idx) => {
        if (key.scope === 'STANDARD') standardKeyIndex = idx;
        if (key.scope === entityType) entityKeyIndex = idx;
    });
    data.results.results.forEach(({ aggregationKeys, keys, numPassing, numFailing }) => {
        const curEntity = aggregationKeys[entityKeyIndex].id;
        const curStandard = aggregationKeys[standardKeyIndex].id;
        if (!entityMap[curEntity]) {
            entityMap[curEntity] = {
                name: keys[entityKeyIndex].name || keys[entityKeyIndex].metadata.name,
                id: curEntity
            };
        }
        entityMap[curEntity][curStandard] = complianceRate(numPassing, numFailing);
    });
    Object.keys(entityMap).forEach(cluster => {
        formattedData.results.push(entityMap[cluster]);
    });
    return formattedData;
};

export default [
    {
        context: [contextTypes.COMPLIANCE],
        pageType: [pageTypes.ENTITY],
        entityType: [entityTypes.CLUSTER],
        component: [componentTypes.HEADER],
        config: {
            query: CLUSTER_QUERY,
            variables: [{ graphQLParam: 'id', queryParam: 'entityId' }]
        }
    },
    {
        context: [contextTypes.COMPLIANCE],
        pageType: [pageTypes.ENTITY],
        entityType: [entityTypes.NAMESPACE],
        component: [componentTypes.HEADER],
        config: {
            query: NAMESPACE_QUERY,
            variables: [{ graphQLParam: 'id', queryParam: 'entityId' }]
        }
    },
    {
        context: [contextTypes.COMPLIANCE],
        pageType: [pageTypes.ENTITY],
        entityType: [entityTypes.NODE],
        component: [componentTypes.HEADER],
        config: {
            query: NODE_QUERY,
            variables: [{ graphQLParam: 'id', queryParam: 'entityId' }]
        }
    },
    {
        context: [contextTypes.COMPLIANCE],
        pageType: [pageTypes.ENTITY],
        entityType: [entityTypes.CLUSTER],
        component: [componentTypes.RELATED_ENTITIES_LIST],
        config: {
            query: RELATED_DEPLOYMENTS,
            variables: [{ graphQLParam: 'id', queryParam: 'entityId' }]
        }
    },
    {
        context: [contextTypes.COMPLIANCE],
        pageType: [pageTypes.LIST],
        entityType: [entityTypes.CLUSTER],
        component: [componentTypes.LIST_TABLE],
        config: {
            query: CLUSTERS_LIST_QUERY,
            variables: [{ graphQLParam: 'id', queryParam: 'entityId' }],
            format(data) {
                return formatComplianceTableData(data, 'CLUSTER');
            }
        }
    },
    {
        context: [contextTypes.COMPLIANCE],
        pageType: [pageTypes.LIST],
        entityType: [entityTypes.NAMESPACE],
        component: [componentTypes.LIST_TABLE],
        config: {
            query: NAMESPACES_LIST_QUERY,
            variables: [{ graphQLParam: 'id', queryParam: 'entityId' }],
            format(data) {
                return formatComplianceTableData(data, 'NAMESPACE');
            }
        }
    },
    {
        context: [contextTypes.COMPLIANCE],
        pageType: [pageTypes.LIST],
        entityType: [entityTypes.NODE],
        component: [componentTypes.LIST_TABLE],
        config: {
            query: NODES_QUERY,
            variables: [{ graphQLParam: 'id', queryParam: 'entityId' }],
            format(data) {
                return formatComplianceTableData(data, 'NODE');
            }
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
            variables: [
                {
                    graphQLParam: 'where',
                    paramsFunc: params => `Standard=${standardLabels[params.entityType]}`
                },
                {
                    graphQLParam: 'groupBy',
                    paramsFunc: params => {
                        const groupByArray = ['CONTROL', 'CATEGORY'];
                        if (params.query.groupBy)
                            groupByArray.push(`${params.query.groupBy.toUpperCase()}`);
                        return groupByArray;
                    }
                }
            ],
            format(data) {
                if (!data.results || data.results.results.length === 0) return null;
                const formattedData = { results: [], totalControls: 0 };
                const groups = {};
                let controlKeyIndex = null;
                let controlGroupKeyIndex = null;
                let groupByKeyIndex = null;
                data.results.results[0].aggregationKeys.forEach((key, idx) => {
                    if (key.scope === 'CONTROL') controlKeyIndex = idx;
                    if (key.scope === 'CATEGORY') controlGroupKeyIndex = idx;
                    if (key.scope !== 'CATEGORY' && key.scope !== 'CONTROL') groupByKeyIndex = idx;
                });
                data.results.results.forEach(({ keys, numPassing, numFailing }) => {
                    const groupKey =
                        groupByKeyIndex === null ? controlGroupKeyIndex : groupByKeyIndex;
                    const { name: groupName, description: groupDescription } = keys[groupKey];
                    if (!groups[groupName]) {
                        groups[groupName] = {
                            name: `${groupName} ${groupDescription ? `- ${groupDescription}` : ''}`,
                            rows: []
                        };
                    }
                    if (controlKeyIndex) {
                        const { id, name, description } = keys[controlKeyIndex];
                        groups[groupName].rows.push({
                            id,
                            name,
                            control: `${name} - ${description}`,
                            compliance: complianceRate(numPassing, numFailing),
                            group: groupName
                        });
                    }
                });
                Object.keys(groups).forEach(group => {
                    formattedData.results.push(groups[group]);
                    formattedData.totalControls += groups[group].rows.length;
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
                { graphQLParam: 'groupBy', graphQLValue: ['STANDARD', 'CLUSTER'] },
                { graphQLParam: 'unit', graphQLValue: 'CHECK' }
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
                { graphQLParam: 'groupBy', graphQLValue: ['STANDARD', 'NAMESPACE'] },
                { graphQLParam: 'unit', graphQLValue: 'CHECK' }
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
                { graphQLParam: 'groupBy', graphQLValue: ['STANDARD', 'NODE'] },
                { graphQLParam: 'unit', graphQLValue: 'CHECK' }
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
                        return entityType;
                    }
                }
            ],
            bypassCache: true
        }
    },
    {
        context: [contextTypes.COMPLIANCE],
        pageType: [pageTypes.DASHBOARD, pageTypes.ENTITY],
        entityType: [],
        component: [componentTypes.COMPLIANCE_BY_STANDARD],
        config: {
            query: COMPLIANCE_STANDARDS,
            variables: []
        }
    }
];
