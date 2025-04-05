import React from 'react';
import { gql } from '@apollo/client';
import { useLocation } from 'react-router-dom';
import entityTypes from 'constants/entityTypes';
import Query from 'Components/ThrowingQuery';
import EntityTileLink from 'Components/EntityTileLink';
import useWorkflowMatch from 'hooks/useWorkflowMatch';
import queryService from 'utils/queryService';
import URLService from 'utils/URLService';

const policiesQuery = gql`
    query numPolicies($query: String) {
        policyCount(query: $query)
    }
`;

const PoliciesTile = () => {
    const match = useWorkflowMatch();
    const location = useLocation();
    const policiesURL = URLService.getURL(match, location).base(entityTypes.POLICY).url();

    return (
        <Query
            query={policiesQuery}
            variables={{
                query: queryService.objectToWhereClause({ 'Lifecycle Stage': 'DEPLOY' }),
            }}
        >
            {({ loading, data }) => {
                const totalNumPolicies = data?.policyCount || 0;
                return (
                    <EntityTileLink
                        count={totalNumPolicies}
                        entityType={entityTypes.POLICY}
                        url={policiesURL}
                        loading={loading}
                        position="first"
                        short
                    />
                );
            }}
        </Query>
    );
};

export default PoliciesTile;
