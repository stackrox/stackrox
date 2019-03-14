import React from 'react';
import PropTypes from 'prop-types';
import { standardLabels } from 'messages/standards';
import URLService from 'modules/URLService';
import pageTypes from 'constants/pageTypes';
import contextTypes from 'constants/contextTypes';
import entityTypes, { standardBaseTypes } from 'constants/entityTypes';
import capitalize from 'lodash/capitalize';

import Widget from 'Components/Widget';
import Sunburst from 'Components/visuals/Sunburst';
import Query from 'Components/ThrowingQuery';
import Loader from 'Components/Loader';
import networkStatuses from 'constants/networkStatuses';
import { COMPLIANCE_STANDARDS as QUERY } from 'queries/standard';
import queryService from 'modules/queryService';
import { Link } from 'react-router-dom';

const colors = [
    'var(--tertiary-400)',
    'var(--warning-400)',
    'var(--caution-400)',
    'var(--alert-400)'
];
const getColor = value => {
    if (value === 100) return colors[0];
    if (value >= 70) return colors[1];
    if (value >= 50) return colors[2];
    return colors[3];
};

const sunburstLegendData = [
    { title: '100%', color: 'var(--tertiary-400)' },
    { title: '> 70%', color: 'var(--warning-400)' },
    { title: '> 50%', color: 'var(--caution-400)' },
    { title: '< 50%', color: 'var(--alert-400)' }
];

const processSunburstData = (data, type) => {
    if (!data || !data.results || !data.results.results.length)
        return { sunburstData: [], totalPassing: 0 };

    const groupMapping = {};
    let controlKeyIndex = 0;
    let categoryKeyIndex = 0;
    data.results.results[0].aggregationKeys.forEach(({ scope }, idx) => {
        if (scope === entityTypes.CONTROL) controlKeyIndex = idx;
        if (scope === entityTypes.CATEGORY) categoryKeyIndex = idx;
    });

    const statsReducer = (statsMapping, { aggregationKeys, numPassing, numFailing, unit }) => {
        const mapping = { ...statsMapping };
        const isGroup = unit === entityTypes.CONTROL;
        const keyIndex = isGroup ? categoryKeyIndex : controlKeyIndex;
        const key = `${aggregationKeys[keyIndex].id}`;
        const group = mapping[key];
        const passing = isGroup && group ? group.passing + numPassing : numPassing;
        const total =
            isGroup && group ? group.total + numPassing + numFailing : numPassing + numFailing;
        mapping[key] = {
            passing,
            total
        };
        return mapping;
    };
    const filterByNonZero = ({ numPassing, numFailing }) => numPassing + numFailing > 0;

    const groupStatsMapping = data.results.results.filter(filterByNonZero).reduce(statsReducer, {});
    const controlStatsMapping = data.checks.results
        .filter(filterByNonZero)
        .reduce(statsReducer, {});

    const { groups, controls } = data.complianceStandards.filter(datum => datum.id === type)[0];

    groups.forEach(datum => {
        const groupStat = groupStatsMapping[datum.id];
        if (groupStat !== undefined) {
            const value = Math.round((groupStat.passing / groupStat.total) * 100);
            groupMapping[datum.id] = {
                name: `${datum.name}. ${datum.description}`,
                color: getColor(value),
                value,
                children: []
            };
        }
    });

    controls
        .filter(control => control.standardId === type)
        .forEach(datum => {
            const group = groupMapping[datum.groupId];
            const controlStat = controlStatsMapping[datum.id];
            const link = URLService.getLinkTo(contextTypes.COMPLIANCE, pageTypes.ENTITY, {
                entityType: entityTypes.CONTROL,
                standardId: datum.standardId,
                controlId: datum.id
            });
            if (group !== undefined && controlStat !== undefined) {
                const value = Math.round((controlStat.passing / controlStat.total) * 100);
                group.children.push({
                    name: `${datum.name} - ${datum.description}`,
                    color: getColor(value),
                    link: link.url,
                    value
                });
            }
        });

    const { passing, total } = Object.values(groupStatsMapping).reduce(
        (acc, currVal) => ({
            passing: acc.passing + currVal.passing,
            total: acc.total + currVal.total
        }),
        { passing: 0, total: 0 }
    );

    const totalPassing = Math.round((passing / total) * 100);

    return {
        sunburstData: Object.values(groupMapping),
        totalPassing
    };
};

const getNumControls = sunburstData =>
    sunburstData.reduce((acc, curr) => acc + curr.children.length, 0);

const getParams = standardType => ({
    entityType: standardType,
    query: {
        groupBy: entityTypes.CATEGORY
    }
});

const createURLLink = (entityType, standardType, entityName) => {
    const linkParams = getParams(standardType);
    if (entityName) {
        const entityKey = capitalize(entityType);
        linkParams.query[entityKey] = entityName;
    }
    const link = URLService.getLinkTo(contextTypes.COMPLIANCE, pageTypes.LIST, linkParams);

    return link;
};

const ComplianceByStandard = ({ standardType, entityName, entityType, entityId, className }) => {
    const groupBy = [
        entityTypes.STANDARD,
        entityTypes.CATEGORY,
        entityTypes.CONTROL,
        ...(entityType ? [entityType] : [])
    ];
    const where = queryService.objectToWhereClause({
        Standard: standardLabels[standardType]
    });
    return (
        <Query
            query={QUERY}
            variables={{
                groupBy,
                where
            }}
        >
            {({ loading, data, networkStatus }) => {
                let contents = <Loader />;
                const headerText = `${standardLabels[standardType]} Compliance`;
                let viewStandardLink = null;

                if (!loading && data && networkStatus === networkStatuses.READY) {
                    const { sunburstData, totalPassing } = processSunburstData(data, standardType);
                    const link = createURLLink(entityType, standardType, entityName);
                    const sunburstRootData = [
                        {
                            text: `${sunburstData.length} Categories`
                        },
                        {
                            text: `${getNumControls(sunburstData)} Controls`,
                            link: link.url
                        }
                    ];
                    const linkToParams = getParams(standardType);
                    const linkTo = URLService.getLinkTo(
                        contextTypes.COMPLIANCE,
                        pageTypes.LIST,
                        linkToParams
                    ).url;

                    viewStandardLink = (
                        <Link to={linkTo} className="no-underline">
                            <button className="btn-sm btn-base" type="button">
                                View Standard
                            </button>
                        </Link>
                    );

                    if (!sunburstData.length) {
                        contents = (
                            <>
                                <div className="flex flex-1 items-center justify-center p-4 leading-loose">
                                    No data available. Please run a scan.
                                </div>
                            </>
                        );
                    } else {
                        contents = (
                            <Sunburst
                                data={sunburstData}
                                rootData={sunburstRootData}
                                legendData={sunburstLegendData}
                                totalValue={totalPassing}
                                key={entityId}
                            />
                        );
                    }
                }
                return (
                    <Widget
                        className={`s-2 ${className}`}
                        header={headerText}
                        headerComponents={viewStandardLink}
                        id={`${standardBaseTypes[standardType]}-compliance`}
                    >
                        {contents}
                    </Widget>
                );
            }}
        </Query>
    );
};

ComplianceByStandard.propTypes = {
    standardType: PropTypes.string.isRequired,
    entityName: PropTypes.string,
    entityType: PropTypes.string,
    entityId: PropTypes.string,
    className: PropTypes.string
};

ComplianceByStandard.defaultProps = {
    entityId: null,
    entityType: null,
    entityName: null,
    className: ''
};

export default ComplianceByStandard;
