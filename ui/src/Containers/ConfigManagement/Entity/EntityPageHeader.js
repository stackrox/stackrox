import React from 'react';
import PropTypes from 'prop-types';
import resolvePath from 'object-resolve-path';
import entityTypes from 'constants/entityTypes';
import { SERVICE_ACCOUNT } from 'queries/serviceAccount';
import { SECRET } from 'queries/secret';
import { CLUSTER_QUERY as CLUSTER } from 'queries/cluster';
import { DEPLOYMENT_QUERY as DEPLOYMENT } from 'queries/deployment';
import { NAMESPACE_QUERY as NAMESPACE } from 'queries/namespace';

import Query from 'Components/ThrowingQuery';
import PageHeader from 'Components/PageHeader';
import Loader from 'Components/Loader';

const queryMap = {
    [entityTypes.SERVICE_ACCOUNT]: SERVICE_ACCOUNT,
    [entityTypes.SECRET]: SECRET,
    [entityTypes.CLUSTER]: CLUSTER,
    [entityTypes.DEPLOYMENT]: DEPLOYMENT,
    [entityTypes.NAMESPACE]: NAMESPACE
};

const nameKeyMap = {
    [entityTypes.SERVICE_ACCOUNT]: 'serviceAccount.name',
    [entityTypes.SECRET]: 'secret.name',
    [entityTypes.CLUSTER]: 'results.name',
    [entityTypes.DEPLOYMENT]: 'deployment.name',
    [entityTypes.NAMESPACE]: 'results.metadata.name'
};

const getQueryAndVariables = (entityType, entityId) => {
    const query = queryMap[entityType] || null;
    return {
        query,
        variables: {
            id: entityId
        }
    };
};

const processHeader = (entityType, data) => {
    const key = nameKeyMap[entityType];
    return resolvePath(data, key);
};

const EntityPageHeader = ({ entityType, entityId }) => {
    const { query, variables } = getQueryAndVariables(entityType, entityId);
    if (!query) return null;
    return (
        <Query query={query} variables={variables}>
            {({ loading, data }) => {
                if (loading) return <Loader />;
                const header = processHeader(entityType, data);
                if (!header) return null;
                return (
                    <PageHeader classes="bg-primary-100" header={header} subHeader={entityType} />
                );
            }}
        </Query>
    );
};

EntityPageHeader.propTypes = {
    entityType: PropTypes.string.isRequired,
    entityId: PropTypes.string.isRequired
};

export default EntityPageHeader;
