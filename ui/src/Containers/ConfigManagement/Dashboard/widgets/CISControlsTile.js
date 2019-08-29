import React from 'react';
import gql from 'graphql-tag';
import URLService from 'modules/URLService';
import { withRouter } from 'react-router-dom';
import ReactRouterPropTypes from 'react-router-prop-types';
import entityTypes from 'constants/entityTypes';

import Query from 'Components/ThrowingQuery';
import TileLink from 'Components/TileLink';

const QUERY = gql`
    query numCISControls {
        complianceStandards(query: "Standard: CIS") {
            id
            numImplementedChecks
        }
    }
`;

function getNumCISControls(data) {
    return data.complianceStandards.reduce((acc, curr) => {
        return acc + curr.numImplementedChecks;
    }, 0);
}

const CISControlsTile = ({ match, location }) => {
    const controlsLink = URLService.getURL(match, location)
        .base(entityTypes.CONTROL)
        .url();

    return (
        <Query query={QUERY}>
            {({ loading, data }) => {
                let numCISControls = 0;
                if (!loading) numCISControls = getNumCISControls(data);
                return (
                    <TileLink
                        value={numCISControls}
                        caption="CIS Controls"
                        to={controlsLink}
                        loading={loading}
                        className="rounded-none"
                    />
                );
            }}
        </Query>
    );
};

CISControlsTile.propTypes = {
    match: ReactRouterPropTypes.match.isRequired,
    location: ReactRouterPropTypes.location.isRequired
};

export default withRouter(CISControlsTile);
