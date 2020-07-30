import React from 'react';
import URLService from 'utils/URLService';
import { withRouter } from 'react-router-dom';
import ReactRouterPropTypes from 'react-router-prop-types';
import entityTypes from 'constants/entityTypes';
import { gql, useQuery } from '@apollo/client';
import logError from 'utils/logError';

import EntityTileLink from 'Components/EntityTileLink';

const NUM_CIS_CONTROLS = gql`
    query numCISControls {
        executedControlCount(query: "Standard: CIS")
    }
`;

const CISControlsTile = ({ match, location }) => {
    const { loading, error, data } = useQuery(NUM_CIS_CONTROLS);
    if (error) logError(error);

    const controlsURL = URLService.getURL(match, location).base(entityTypes.CONTROL).url();

    const numCISControls = data?.executedControlCount || 0;
    return (
        <EntityTileLink
            count={numCISControls}
            entityType={entityTypes.CONTROL}
            url={controlsURL}
            loading={loading}
            position="middle"
            short
        />
    );
};

CISControlsTile.propTypes = {
    match: ReactRouterPropTypes.match.isRequired,
    location: ReactRouterPropTypes.location.isRequired,
};

export default withRouter(CISControlsTile);
