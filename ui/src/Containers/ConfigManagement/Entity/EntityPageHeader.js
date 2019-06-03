import React from 'react';
import PropTypes from 'prop-types';
import resolvePath from 'object-resolve-path';
import entityTypes from 'constants/entityTypes';
import { SERVICE_ACCOUNT } from 'queries/serviceAccount';

import Query from 'Components/ThrowingQuery';
import PageHeader from 'Components/PageHeader';
import Loader from 'Components/Loader';

const queryMap = {
    [entityTypes.SERVICE_ACCOUNT]: SERVICE_ACCOUNT
};

const nameKeyMap = {
    [entityTypes.SERVICE_ACCOUNT]: 'serviceAccount.name'
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
                return <PageHeader classes="bg-primary-100" header={header} subHeader="Entity" />;
            }}
        </Query>
    );
};

EntityPageHeader.propTypes = {
    entityType: PropTypes.string.isRequired,
    entityId: PropTypes.string.isRequired
};

export default EntityPageHeader;
