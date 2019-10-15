import React from 'react';
import gql from 'graphql-tag';
import URLService from 'modules/URLService';
import { withRouter } from 'react-router-dom';
import ReactRouterPropTypes from 'react-router-prop-types';
import entityTypes from 'constants/entityTypes';
import { useQuery } from 'react-apollo';
import logError from 'modules/logError';

import EntityTileLink from 'Components/EntityTileLink';

const NUM_CIS_CONTROLS = gql`
    query numCISControls {
        executedControlCount(query: "Standard: CIS")
    }
`;

function getNumCISControls(data) {
    if (!data || !data.executedControlCount) return 0;
    return data.executedControlCount;
}

const CISControlsTile = ({ match, location }) => {
    const { loading, error, data } = useQuery(NUM_CIS_CONTROLS);
    if (error) logError(error);

    const controlsURL = URLService.getURL(match, location)
        .base(entityTypes.CONTROL)
        .url();

    const numCISControls = !loading ? getNumCISControls(data) : 0;
    return (
        <EntityTileLink
            count={numCISControls}
            entityType={entityTypes.CONTROL}
            url={controlsURL}
            loading={loading}
            position="middle"
        />
    );
};

CISControlsTile.propTypes = {
    match: ReactRouterPropTypes.match.isRequired,
    location: ReactRouterPropTypes.location.isRequired
};

export default withRouter(CISControlsTile);
