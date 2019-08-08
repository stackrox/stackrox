import React from 'react';
import PropTypes from 'prop-types';
import getEntityName from 'modules/getEntityName';
import { entityNameQueryMap } from 'modules/queryMap';

import Query from 'Components/ThrowingQuery';
import PageHeader from 'Components/PageHeader';

const getQueryAndVariables = (entityType, entityId) => {
    const query = entityNameQueryMap[entityType] || null;
    return {
        query,
        variables: {
            id: entityId
        }
    };
};

const EntityPageHeader = ({ entityType, entityId, children }) => {
    const { query, variables } = getQueryAndVariables(entityType, entityId);
    if (!query) return null;
    return (
        <Query query={query} variables={variables}>
            {({ data }) => {
                const header = getEntityName(entityType, data) || '-';
                return (
                    <PageHeader classes="bg-primary-100" header={header} subHeader={entityType}>
                        {children}
                    </PageHeader>
                );
            }}
        </Query>
    );
};

EntityPageHeader.propTypes = {
    entityType: PropTypes.string.isRequired,
    entityId: PropTypes.string.isRequired,
    children: PropTypes.node.isRequired
};

export default EntityPageHeader;
