import React from 'react';
import gql from 'graphql-tag';
import URLService from 'utils/URLService';
import { withRouter } from 'react-router-dom';
import ReactRouterPropTypes from 'react-router-prop-types';
import entityTypes from 'constants/entityTypes';
import Query from 'Components/ThrowingQuery';
import EntityTileLink from 'Components/EntityTileLink';
import queryService from 'utils/queryService';

const policiesQuery = gql`
    query numPolicies($query: String) {
        policies(query: $query) {
            id
            lifecycleStages
            policyStatus
        }
    }
`;

function getTotalNumPolicies(data) {
    if (!data || !data.policies) return 0;
    const totalPolicies = data.policies.length;
    return totalPolicies;
}

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
                const totalNumPolicies = getTotalNumPolicies(data);
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
