import React, { useContext } from 'react';
import PropTypes from 'prop-types';
import ReactRouterPropTypes from 'react-router-prop-types';
import { withRouter } from 'react-router-dom';
import { useQuery } from '@apollo/client';
import merge from 'lodash/merge';

import entityTypes, { standardBaseTypes } from 'constants/entityTypes';
import { standardLabels } from 'messages/standards';
import URLService from 'utils/URLService';
import Widget from 'Components/Widget';
import Loader from 'Components/Loader';
import NoResultsMessage from 'Components/NoResultsMessage';
import { AGGREGATED_RESULTS_ACROSS_ENTITY } from 'queries/controls';
import searchContext from 'Containers/searchContext';

import { entityNounOrdinaryCasePlural } from '../entitiesForCompliance';
import HorizontalBarChart from './HorizontalBarChart';

function formatAsPercent(x) {
    return `${x}%`;
}

function setStandardsMapping(data, key, type) {
    const mapping = {};
    data[key].results.forEach((result) => {
        const standardId = result.aggregationKeys[0].id;
        if (!data.complianceStandards.some(({ id }) => id === standardId)) {
            return; // because it implies standardId has hideScanResults: true
        }
        const { numPassing, numFailing } = result;
        if (numPassing === 0 && numFailing === 0) {
            return;
        }
        if (!mapping[standardId]) {
            mapping[standardId] = {};
            mapping[standardId][type] = {
                passing: numPassing,
                total: numPassing + numFailing,
            };
        } else {
            const { passing, total } = mapping[standardId][type];
            mapping[standardId][type] = {
                passing: passing + numPassing,
                total: total + (numPassing + numFailing),
            };
        }
    });

    return mapping;
}

const StandardsAcrossEntity = ({ match, location, entityType, bodyClassName, className }) => {
    const searchParam = useContext(searchContext);
    const headerText = `Passing standards across ${entityNounOrdinaryCasePlural[entityType]}`;

    function processData(data, type) {
        if (!data || !data.results || !data.results.results.length) {
            return [];
        }
        const standardsMapping = merge(
            {},
            setStandardsMapping(data, 'results', 'checks'),
            setStandardsMapping(data, 'controls', 'controls')
        );

        const barData = Object.keys(standardsMapping).map((standardId) => {
            const { checks } = standardsMapping[standardId];
            const { passing: passingChecks, total: totalChecks } = checks;
            const percentagePassing = Math.round((passingChecks / totalChecks) * 100) || 0;
            const link = URLService.getURL(match, location)
                .base(entityTypes.CONTROL)
                .query({
                    [searchParam]: {
                        groupBy: type,
                        standard: standardLabels[standardId] || standardId,
                    },
                })
                .url();
            const dataPoint = {
                y: standardBaseTypes[standardId] || standardId,
                x: percentagePassing,
                link,
            };
            return dataPoint;
        });
        return barData;
    }

    const variables = {
        groupBy: [entityTypes.STANDARD, entityType],
        unit: entityTypes.CHECK,
    };
    const { loading, error, data } = useQuery(AGGREGATED_RESULTS_ACROSS_ENTITY(entityType), {
        variables,
    });

    if (error) {
        return (
            <Widget
                className={`s-2 ${className}`}
                header={headerText}
                bodyClassName={`graph-bottom-border ${bodyClassName}`}
            >
                <div>
                    A database error has occurred. Please check that you have the correct
                    permissions to view this information.
                </div>
            </Widget>
        );
    }

    let contents;
    if (!loading && data && data.complianceStandards) {
        const results = processData(data, entityType);
        if (!results.length) {
            contents = <NoResultsMessage message="No data available. Please run a scan." />;
        } else {
            contents = <HorizontalBarChart data={results} valueFormat={formatAsPercent} />;
        }
    } else {
        contents = <Loader />;
    }
    return (
        <Widget
            className={`s-2 ${className}`}
            header={headerText}
            bodyClassName={`graph-bottom-border ${bodyClassName}`}
        >
            {contents}
        </Widget>
    );
};

StandardsAcrossEntity.propTypes = {
    match: ReactRouterPropTypes.match.isRequired,
    location: ReactRouterPropTypes.location.isRequired,
    entityType: PropTypes.string.isRequired,
    bodyClassName: PropTypes.string,
    className: PropTypes.string,
};

StandardsAcrossEntity.defaultProps = {
    bodyClassName: 'px-4 pt-1',
    className: '',
};

export default withRouter(StandardsAcrossEntity);
