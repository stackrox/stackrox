import React from 'react';
import URLService from 'modules/URLService';
import { withRouter } from 'react-router-dom';
import ReactRouterPropTypes from 'react-router-prop-types';
import entityTypes from 'constants/entityTypes';
import Query from 'Components/ThrowingQuery';
import TileLink from 'Components/TileLink';
import { AGGREGATED_RESULTS_WITH_CONTROLS as CISControlsQuery } from 'queries/controls';
import queryService from 'modules/queryService';

function processControlsData(data) {
    let totalControls = 0;
    let hasViolations = false;

    if (!data || !data.results || !data.results.results || !data.results.results.length)
        return { totalControls, hasViolations };

    const { results } = data.results;
    totalControls = data.complianceStandards
        .filter(standard => standard.name.includes('CIS'))
        .reduce((total, standard) => {
            return total + standard.controls.length;
        }, 0);

    hasViolations = !!results.find(({ numFailing }) => {
        return numFailing > 0;
    });

    return { totalControls, hasViolations };
}

const CISControlsHeaderTile = ({ match, location }) => {
    const controlsLink = URLService.getURL(match, location)
        .base(entityTypes.control)
        .url();

    return (
        <Query
            query={CISControlsQuery}
            variables={{
                groupBy: entityTypes.CONTROL,
                unit: entityTypes.CONTROL,
                where: queryService.objectToWhereClause({ standard: 'CIS' })
            }}
        >
            {({ loading, data }) => {
                const { totalControls, hasViolations } = processControlsData(data);
                return (
                    <TileLink
                        value={totalControls}
                        isError={hasViolations}
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

CISControlsHeaderTile.propTypes = {
    match: ReactRouterPropTypes.match.isRequired,
    location: ReactRouterPropTypes.location.isRequired
};

export default withRouter(CISControlsHeaderTile);
