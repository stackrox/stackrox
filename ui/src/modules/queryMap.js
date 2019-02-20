import orderBy from 'lodash/orderBy';

import componentTypes from 'constants/componentTypes';
import entityTypes from 'constants/entityTypes';
import standardLabels from 'messages/standards';
import contextTypes from 'constants/contextTypes';
import pageTypes from 'constants/pageTypes';
import { CLUSTER_QUERY } from 'queries/cluster';
import { NAMESPACE_QUERY } from 'queries/namespace';
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

const complianceRate = (numPassing, numFailing) =>
    numPassing + numFailing > 0
        ? `${Math.round((numPassing / (numPassing + numFailing)) * 100)}%`
        : 'N/A';

const formatComplianceTableData = (data, entityType) => {
    if (!data.results || data.results.results.length === 0) return null;
    const formattedData = { results: [] };
    const entityMap = {};
    let standardKeyIndex = 0;
    let entityKeyIndex = 0;
    data.results.results[0].aggregationKeys.forEach(({ scope }, idx) => {
        if (scope === 'STANDARD') standardKeyIndex = idx;
        if (scope === entityType) entityKeyIndex = idx;
    });
    data.results.results.forEach(({ aggregationKeys, keys, numPassing, numFailing }) => {
        const curEntity = aggregationKeys[entityKeyIndex].id;
        const curStandard = aggregationKeys[standardKeyIndex].id;
        if (!entityMap[curEntity]) {
            const entity = keys[entityKeyIndex];
            // the check below is to address ROX-1420
            // eslint-disable-next-line no-underscore-dangle
            if (entity.__typename !== '') {
                entityMap[curEntity] = {
                    name: entity.name || entity.metadata.name,
                    id: curEntity,
                    overall: {
                        numPassing: 0,
                        numFailing: 0,
                        average: 0
                    }
                };
                if (entityType !== entityTypes.CLUSTER) {
                    entityMap[curEntity].cluster =
                        entity.clusterName || entity.metadata.clusterName;
                }
                if (numPassing + numFailing > 0)
                    entityMap[curEntity][curStandard] = complianceRate(numPassing, numFailing);
                entityMap[curEntity].overall.numPassing += numPassing;
                entityMap[curEntity].overall.numFailing += numFailing;
            }
        }
    });
    Object.keys(entityMap).forEach(cluster => {
        const overallCluster = Object.assign({}, entityMap[cluster]);
        const { numPassing, numFailing } = overallCluster.overall;
        overallCluster.overall.average = complianceRate(numPassing, numFailing);
        formattedData.results.push(overallCluster);
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
                const formattedData = { results: [], totalRows: 0 };
                const groups = {};
                let controlKeyIndex = null;
                let categoryKeyIndex = null;
                let groupByKeyIndex = null;
                data.results.results[0].aggregationKeys.forEach(({ scope }, idx) => {
                    if (scope === 'CONTROL') controlKeyIndex = idx;
                    if (scope === 'CATEGORY') categoryKeyIndex = idx;
                    if (scope !== 'CATEGORY' && scope !== 'CONTROL') groupByKeyIndex = idx;
                });
                data.results.results.forEach(({ keys, numPassing, numFailing }) => {
                    const groupKey = groupByKeyIndex === null ? categoryKeyIndex : groupByKeyIndex;
                    const { name, description: groupDescription, metadata, __typename } = keys[
                        groupKey
                    ];
                    // the check below is to address ROX-1420
                    if (__typename !== '') {
                        const groupName = name || `${metadata.clusterName}--${metadata.name}`;
                        if (!groups[groupName]) {
                            const groupId = parseInt(groupName, 10) || groupName;
                            groups[groupName] = {
                                groupId,
                                name: `${groupName} ${
                                    groupDescription ? `- ${groupDescription}` : ''
                                }`,
                                rows: []
                            };
                        }
                        if (controlKeyIndex) {
                            const { id, name: controlName, description } = keys[controlKeyIndex];
                            groups[groupName].rows.push({
                                id,
                                name: controlName,
                                control: `${controlName} - ${description}`,
                                compliance: complianceRate(numPassing, numFailing),
                                group: groupName
                            });
                        }
                    }
                });
                Object.keys(groups).forEach(group => {
                    formattedData.results.push(groups[group]);
                    formattedData.totalRows += groups[group].rows.length;
                });
                formattedData.results = orderBy(
                    formattedData.results,
                    ['groupId', 'name'],
                    ['asc', 'asc']
                );
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
                { graphQLParam: 'unit', graphQLValue: 'CONTROL' }
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
                { graphQLParam: 'unit', graphQLValue: 'CONTROL' }
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
                { graphQLParam: 'unit', graphQLValue: 'CONTROL' }
            ]
        }
    },
    {
        context: [contextTypes.COMPLIANCE],
        pageType: [pageTypes.DASHBOARD, pageTypes.ENTITY],
        entityType: [],
        component: [componentTypes.COMPLIANCE_BY_STANDARD],
        config: {
            query: COMPLIANCE_STANDARDS,
            variables: [
                {
                    graphQLParam: 'groupBy',
                    paramsFunc: params => {
                        const groupByArray = ['STANDARD', 'CATEGORY', 'CONTROL'];
                        if (params.pageType === pageTypes.ENTITY)
                            groupByArray.push(params.entityType);
                        return groupByArray;
                    }
                }
            ]
        }
    }
];
