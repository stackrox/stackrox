import React from 'react';
import PropTypes from 'prop-types';
import { standardLabels } from 'messages/standards';
import URLService from 'modules/URLService';
import entityTypes, { standardBaseTypes } from 'constants/entityTypes';
import capitalize from 'lodash/capitalize';
import ReactRouterPropTypes from 'react-router-prop-types';
import Widget from 'Components/Widget';
import Sunburst from 'Components/visuals/Sunburst';
import Query from 'Components/ThrowingQuery';
import Loader from 'Components/Loader';
import networkStatuses from 'constants/networkStatuses';
import { COMPLIANCE_STANDARDS as QUERY } from 'queries/standard';
import queryService from 'modules/queryService';
import { Link, withRouter } from 'react-router-dom';

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

const processSunburstData = (match, location, data, type) => {
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

    const groupStatsMapping = data.results.results.reduce(statsReducer, {});
    const controlStatsMapping = data.checks.results.reduce(statsReducer, {});

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

            const url = URLService.getURL(match, location)
                .base(entityTypes.CONTROL, datum.id)
                .url();

            if (group !== undefined && controlStat !== undefined) {
                const value = Math.round((controlStat.passing / controlStat.total) * 100);
                group.children.push({
                    name: `${datum.name} - ${datum.description}`,
                    color: getColor(value),
                    link: url,
                    value
                });
            }
        });

    const { passing, total } = Object.values(controlStatsMapping).reduce(
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

const createURLLink = (match, location, entityType, standardId, entityName) => {
    const query = { groupBy: entityTypes.CATEGORY };
    if (entityName) {
        const entityKey = capitalize(entityType);
        query[entityKey] = entityName;
    }
    return URLService.getURL(match, location)
        .base(entityTypes.CONTROL)
        .query({ standard: standardLabels[standardId], ...query })
        .url();
};

const ComplianceByStandard = ({
    match,
    location,
    standardType,
    entityName,
    entityType,
    entityId,
    className
}) => {
    const groupBy = [
        entityTypes.STANDARD,
        entityTypes.CATEGORY,
        entityTypes.CONTROL,
        ...(entityType ? [entityType] : [])
    ];
    const where = {
        Standard: standardLabels[standardType]
    };
    if (entityType) where[`${entityType} ID`] = entityId;
    const variables = {
        groupBy,
        where: queryService.objectToWhereClause(where)
    };
    return (
        <Query query={QUERY} variables={variables}>
            {({ loading, data, networkStatus }) => {
                let contents = <Loader />;
                const headerText = `${standardLabels[standardType]} Compliance`;
                let viewStandardLink = null;

                if (!loading && data && networkStatus === networkStatuses.READY) {
                    const { sunburstData, totalPassing } = processSunburstData(
                        match,
                        location,
                        data,
                        standardType
                    );
                    const url = createURLLink(
                        match,
                        location,
                        entityType,
                        standardType,
                        entityName
                    );
                    const sunburstRootData = [
                        {
                            text: `${sunburstData.length} Categories`
                        },
                        {
                            text: `${getNumControls(sunburstData)} Controls`,
                            link: url
                        }
                    ];

                    const linkTo = URLService.getURL(match, location)
                        .base(entityTypes.CONTROL)
                        .query({
                            standard: standardLabels[standardType],
                            groupBy: entityTypes.CATEGORY
                        })
                        .url();

                    viewStandardLink = (
                        <Link to={linkTo} className="no-underline">
                            <button className="btn-sm btn-base" type="button">
                                View Standard
                            </button>
                        </Link>
                    );

                    if (!sunburstData.length) {
                        contents = (
                            <div className="flex flex-1 items-center justify-center p-4 leading-loose">
                                No data available. Please run a scan.
                            </div>
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
    match: ReactRouterPropTypes.match.isRequired,
    location: ReactRouterPropTypes.location.isRequired,
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

export default withRouter(ComplianceByStandard);
