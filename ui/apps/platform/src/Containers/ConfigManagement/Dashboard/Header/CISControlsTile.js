import React from 'react';
import URLService from 'utils/URLService';
import { useLocation, useMatch } from 'react-router-dom';
import entityTypes from 'constants/entityTypes';
import { gql, useQuery } from '@apollo/client';
import logError from 'utils/logError';
import { workflowPaths } from 'routePaths';

import EntityTileLink from 'Components/EntityTileLink';

const NUM_CIS_CONTROLS = gql`
    query numCISControls {
        executedControlCount(query: "Standard: CIS")
    }
`;

const CISControlsTile = () => {
    const { loading, error, data } = useQuery(NUM_CIS_CONTROLS);
    if (error) {
        logError(error);
    }

    const match = useMatch(workflowPaths.DASHBOARD);
    const location = useLocation();
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

export default CISControlsTile;
