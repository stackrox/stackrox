import componentTypes from 'constants/componentTypes';
import entityTypes from 'constants/entityTypes';
import contextTypes from 'constants/contextTypes';
import pageTypes from 'constants/pageTypes';
import { CLUSTER_QUERY, CLUSTER_COMPLIANCE } from 'queries/cluster';
import { NAMESPACE_QUERY, RELATED_DEPLOYMENTS } from 'queries/namespace';
import { CLUSTERS_QUERY, NAMESPACES_QUERY, NODES_QUERY } from 'queries/table';
import { NODE_QUERY } from 'queries/node';
import AGGREGATED_RESULTS from 'queries/controls';

/**
 * context:     Array of contextTypes to match
 * pageType:    Array of pageTypes to match
 * entityType:  Array of entityTypes to match
 * config:      Contains information about the query
 *      query:      GraphQL query text
 *      variables:  A mapping of GraphQL parameter names to URL/Query parameter names used byApp Query to pass url params to the query.
 *                  e.g.:
 *                      URL       =   /:entityType/:entityId
 *                      GraphQL   =   query getCluster($id: ID!)
 *                      variables =   [{ graphQLParam: 'id', queryParam: 'entityId' }]
 *      format:     A function run on the result set before returning.
 */

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
        entityType: [entityTypes.CLUSTERS],
        component: [componentTypes.ENTITY_COMPLIANCE],
        config: {
            query: CLUSTER_COMPLIANCE,
            variables: [{ graphQLParam: 'id', queryParam: 'entityId' }],
            format(data) {
                const formattedData = {
                    ...data
                };
                formattedData.results = getSubField(data, 'aggregatedResults.results');
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
    }
];
