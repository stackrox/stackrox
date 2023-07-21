import React, { useContext } from 'react';
import PropTypes from 'prop-types';
import ReactRouterPropTypes from 'react-router-prop-types';
import { withRouter } from 'react-router-dom';
import { useQuery } from '@apollo/client';
import capitalize from 'lodash/capitalize';
import sortBy from 'lodash/sortBy';
import chunk from 'lodash/chunk';

import entityTypes from 'constants/entityTypes';
import URLService from 'utils/URLService';
import Widget from 'Components/Widget';
import Loader from 'Components/Loader';
import NoResultsMessage from 'Components/NoResultsMessage';
import { standardLabels } from 'messages/standards';
import { AGGREGATED_RESULTS_STANDARDS_BY_ENTITY } from 'queries/controls';
import searchContext from 'Containers/searchContext';

import { entityNounOrdinaryCaseSingular } from '../entitiesForCompliance';
import VerticalClusterBar from './VerticalClusterBar';

function processData(match, location, data, entityType, searchParam) {
    if (!data || !data.results.results.length || !data.entityList) {
        return [];
    }
    const standardsGrouping = {};
    const { results, entityList, complianceStandards } = data;
    results.results.forEach((result) => {
        const entity = entityList.find(
            (entityObject) => entityObject.id === result.aggregationKeys[1].id
        );
        if (!entity) {
            return;
        }
        const standard = complianceStandards.find((c) => c.id === result.aggregationKeys[0].id);
        const { numPassing, numFailing } = result;
        if (!standard || (numPassing === 0 && numFailing === 0)) {
            return;
        }
        const percentagePassing = Math.round((numPassing / (numPassing + numFailing)) * 100);

        const link = URLService.getURL(match, location)
            .base(entityTypes.CONTROL)
            .query({
                [searchParam]: {
                    [`${capitalize(entityType)}`]: entity?.name,
                    standard: standardLabels[standard.id] || standard.id,
                },
            })
            .url();
        const dataPoint = {
            x: entity?.name,
            y: percentagePassing,
            link,
        };
        const standardGroup = standardsGrouping[standard.id];
        if (standardGroup) {
            standardGroup.push(dataPoint);
        } else {
            standardsGrouping[standard.id] = [dataPoint];
        }
    });
    const sortedStandardsGrouping = {};
    const GRAPHS_PER_PAGE = 3;
    const pagedStandardsGrouping = [];

    Object.keys(standardsGrouping).forEach((standard) => {
        sortedStandardsGrouping[standard] = sortBy(standardsGrouping[standard], ['x']);
        const pageArray = chunk(sortedStandardsGrouping[standard], GRAPHS_PER_PAGE);
        pageArray.forEach((page, pageIdx) => {
            if (!pagedStandardsGrouping[pageIdx]) {
                pagedStandardsGrouping[pageIdx] = {};
            }
            pagedStandardsGrouping[pageIdx][standard] = pageArray[pageIdx];
        });
    });
    return pagedStandardsGrouping;
}

function getLabelLinks(match, location, data, entityType) {
    if (!data) {
        return null;
    }
    const { entityList } = data;
    const labelLinks = {};
    entityList.forEach((entity) => {
        labelLinks[entity?.name] = URLService.getURL(match, location)
            .base(entityType, entity?.id)
            .url();
    });
    return labelLinks;
}

const StandardsByEntity = ({ match, location, entityType, bodyClassName, className }) => {
    const searchParam = useContext(searchContext);
    const headerText = `Passing standards by ${entityNounOrdinaryCaseSingular[entityType]}`;

    const variables = {
        groupBy: [entityTypes.STANDARD, entityType],
        unit: entityTypes.CHECK,
    };
    const { loading, error, data } = useQuery(AGGREGATED_RESULTS_STANDARDS_BY_ENTITY(entityType), {
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

    let contents = <Loader />;
    let pages;

    if (!loading) {
        if (!data || !data.results || data.results.length === 0) {
            contents = <NoResultsMessage message="No data available. Please run a scan." />;
        } else {
            const formattedData = {
                results: data && data.results,
                controls: data && data.controls,
                complianceStandards: data.complianceStandards,
                entityList: data && data.clusters,
            };
            const results = processData(match, location, formattedData, entityType, searchParam);
            const labelLinks = getLabelLinks(match, location, formattedData, entityType);
            pages = results.length;

            if (pages) {
                const VerticalBarChartPaged = ({ currentPage }) => (
                    <VerticalClusterBar
                        id={`passing-standards-by-${entityType.toLowerCase()}`}
                        data={results[currentPage]}
                        labelLinks={labelLinks}
                    />
                );
                VerticalBarChartPaged.propTypes = { currentPage: PropTypes.number };
                VerticalBarChartPaged.defaultProps = { currentPage: 0 };
                contents = <VerticalBarChartPaged />;
            } else {
                contents = <NoResultsMessage message="No data available. Please run a scan." />;
            }
        }
    }

    return (
        <Widget
            className={`s-2 ${className}`}
            pages={pages}
            header={headerText}
            bodyClassName={`graph-bottom-border ${bodyClassName}`}
        >
            {contents}
        </Widget>
    );
};
StandardsByEntity.propTypes = {
    match: ReactRouterPropTypes.match.isRequired,
    location: ReactRouterPropTypes.location.isRequired,
    entityType: PropTypes.string.isRequired,
    bodyClassName: PropTypes.string,
    className: PropTypes.string,
};

StandardsByEntity.defaultProps = {
    bodyClassName: 'p-4',
    className: '',
};

export default withRouter(StandardsByEntity);
