import componentTypes from 'constants/componentTypes';
import entityTypes from 'constants/entityTypes';
import contextTypes from 'constants/contextTypes';
import pageTypes from 'constants/pageTypes';
import { CLUSTER_QUERY } from 'queries/cluster';
import { NAMESPACE_QUERY, RELATED_DEPLOYMENTS } from 'queries/namespace';
import { NODE_QUERY } from 'queries/node';

export default [
    {
        context: [contextTypes.COMPLIANCE],
        pageType: [pageTypes.ENTITY],
        entityType: [entityTypes.CLUSTERS],
        component: [componentTypes.HEADER],
        query: CLUSTER_QUERY,
        variables: [{ graphQLParam: 'id', queryParam: 'entityId' }]
    },
    {
        context: [contextTypes.COMPLIANCE],
        pageType: [pageTypes.ENTITY],
        entityType: [entityTypes.NAMESPACES],
        component: [componentTypes.HEADER],
        query: NAMESPACE_QUERY,
        variables: [{ graphQLParam: 'id', queryParam: 'entityId' }]
    },
    {
        context: [contextTypes.COMPLIANCE],
        pageType: [pageTypes.ENTITY],
        entityType: [entityTypes.NODES],
        component: [componentTypes.HEADER],
        query: NODE_QUERY,
        variables: [{ graphQLParam: 'id', queryParam: 'entityId' }]
    },
    {
        context: [contextTypes.COMPLIANCE],
        pageType: [pageTypes.ENTITY],
        entityType: [entityTypes.CLUSTERS],
        component: [componentTypes.RELATED_ENTITIES_LIST],
        query: RELATED_DEPLOYMENTS,
        metadata: { entityType: entityTypes.NAMESPACES },
        variables: [{ graphQLParam: 'id', queryParam: 'entityId' }]
    }
];
