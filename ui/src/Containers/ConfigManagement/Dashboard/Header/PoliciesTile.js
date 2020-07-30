import React from 'react';
import { gql } from '@apollo/client';
import URLService from 'utils/URLService';
import { withRouter } from 'react-router-dom';
import ReactRouterPropTypes from 'react-router-prop-types';
import entityTypes from 'constants/entityTypes';
import Query from 'Components/ThrowingQuery';
import EntityTileLink from 'Components/EntityTileLink';
import queryService from 'utils/queryService';

const policiesQuery = gql`
    query numPolicies($query: String) {
        policyCount(query: $query)
    }
`;

const PoliciesTile = ({ match, location }) => {
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

PoliciesTile.propTypes = {
    match: ReactRouterPropTypes.match.isRequired,
    location: ReactRouterPropTypes.location.isRequired,
};

export default withRouter(PoliciesTile);
