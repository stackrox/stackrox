import React, { useContext } from 'react';
import PropTypes from 'prop-types';
import capitalize from 'lodash/capitalize';
import { Link, useLocation, useRouteMatch } from 'react-router-dom';

import URLService from 'utils/URLService';
import entityTypes, { standardBaseTypes } from 'constants/entityTypes';
import Widget from 'Components/Widget';
import Sunburst from 'Components/visuals/Sunburst';
import Query from 'Components/CacheFirstQuery';
import Loader from 'Components/Loader';
import { COMPLIANCE_STANDARDS } from 'queries/standard';
import queryService from 'utils/queryService';
import searchContext from 'Containers/searchContext';
import isGQLLoading from 'utils/gqlLoading';

const passColor = 'var(--tertiary-400)';
const warningColor = 'var(--warning-400)';
const cautionColor = 'var(--caution-400)';
const skippedColor = 'var(--base-300)';
const alertColor = 'var(--alert-400)';

const passTextColor = 'var(--tertiary-500)';
const warningTextColor = 'var(--warning-500)';
const cautionTextColor = 'var(--caution-500)';
const skippedTextColor = 'var(--base-500)';
const alertTextColor = 'var(--alert-500)';

const getColor = (value) => {
    if (value === 100) {
        return passColor;
    }
    if (value >= 70) {
        return warningColor;
    }
    if (value >= 50) {
        return cautionColor;
    }
    if (Number.isNaN(value)) {
        return skippedColor;
    }
    return alertColor;
};

const getTextColor = (value) => {
    if (value === 100) {
        return passTextColor;
    }
    if (value >= 70) {
        return warningTextColor;
    }
    if (value >= 50) {
        return cautionTextColor;
    }
    if (Number.isNaN(value)) {
        return skippedTextColor;
    }
    return alertTextColor;
};

const sunburstLegendData = [
    { title: '100%', color: passColor },
    { title: '> 70%', color: warningColor },
    { title: '> 50%', color: cautionColor },
    { title: '< 50%', color: alertColor },
    { title: 'Skipped', color: skippedColor },
];

const processSunburstData = (match, location, data, standard) => {
    const groupMapping = {};
    let controlKeyIndex = 0;
    let categoryKeyIndex = 0;
    data.results.results[0].aggregationKeys.forEach(({ scope }, idx) => {
        if (scope === entityTypes.CONTROL) {
            controlKeyIndex = idx;
        }
        if (scope === entityTypes.CATEGORY) {
            categoryKeyIndex = idx;
        }
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
            total,
        };
        return mapping;
    };

    const groupStatsMapping = data.results.results.reduce(statsReducer, {});
    const controlStatsMapping = data.checks.results.reduce(statsReducer, {});

    const { groups, controls } = data.complianceStandards.find((datum) => datum.id === standard);

    groups.forEach((datum) => {
        const groupStat = groupStatsMapping[datum.id];
        if (groupStat !== undefined) {
            const value = Math.round((groupStat.passing / groupStat.total) * 100);
            groupMapping[datum.id] = {
                name: `${datum?.name}. ${datum?.description}`,
                color: getColor(value),
                textColor: getTextColor(value),
                value,
                children: [],
            };
        }
    });

    controls
        .filter((control) => control.standardId === standard)
        .forEach((datum) => {
            const group = groupMapping[datum.groupId];
            const controlStat = controlStatsMapping[datum.id];

            const url = URLService.getURL(match, location)
                .base(entityTypes.CONTROL, datum.id)
                .url();

            if (group !== undefined && controlStat !== undefined) {
                const value = Math.round((controlStat.passing / controlStat.total) * 100);
                group.children.push({
                    name: `${datum?.name} - ${datum?.description}`,
                    color: getColor(value),
                    textColor: getTextColor(value),
                    link: url,
                    value,
                });
            }
        });

    const { passing, total } = Object.values(controlStatsMapping).reduce(
        (acc, currVal) => ({
            passing: acc.passing + currVal.passing,
            total: acc.total + currVal.total,
        }),
        { passing: 0, total: 0 }
    );

    const totalPassing = Math.round((passing / total) * 100);

    return {
        sunburstData: Object.values(groupMapping),
        totalPassing,
    };
};

const getNumControls = (sunburstData) =>
    sunburstData.reduce((acc, curr) => acc + curr.children.length, 0);

const createURLLink = (match, location, entityType, standardName, entityName, searchParam) => {
    const query = { groupBy: entityTypes.CATEGORY };
    if (entityName) {
        const entityKey = capitalize(entityType);
        query[entityKey] = entityName;
    }
    return URLService.getURL(match, location)
        .base(entityTypes.CONTROL)
        .query({ [searchParam]: { standard: standardName } })
        .url();
};

const ComplianceByStandard = ({
    standardName,
    standardId,
    entityName,
    entityType,
    entityId,
    className,
}) => {
    const location = useLocation();
    const match = useRouteMatch();
    const searchParam = useContext(searchContext);
    const groupBy = [
        entityTypes.STANDARD,
        entityTypes.CATEGORY,
        entityTypes.CONTROL,
        ...(entityType ? [entityType] : []),
    ];
    const where = {
        Standard: standardName,
    };
    if (entityType && entityId) {
        where[`${entityType} ID`] = entityId;
    }
    const variables = {
        groupBy,
        where: queryService.objectToWhereClause(where),
    };

    return (
        <Query query={COMPLIANCE_STANDARDS(standardId)} variables={variables}>
            {({ loading, data }) => {
                let contents = null;
                let viewStandardLink = null;
                if (isGQLLoading(loading, data)) {
                    contents = <Loader />;
                } else if (data?.checks?.results?.length && data?.results?.results?.length) {
                    const { sunburstData, totalPassing } = processSunburstData(
                        match,
                        location,
                        data,
                        standardId
                    );

                    const url = createURLLink(
                        match,
                        location,
                        entityType,
                        standardName,
                        entityName,
                        searchParam
                    );
                    const sunburstRootData = [
                        {
                            text: `${sunburstData.length} Categories`,
                        },
                        {
                            text: `${getNumControls(sunburstData)} Controls`,
                            link: url,
                            color: 'var(--tertiary-700)',
                        },
                    ];

                    const linkTo = URLService.getURL(match, location)
                        .base(entityTypes.CONTROL)
                        .query({
                            [searchParam]: {
                                standard: standardName,
                                groupBy: entityTypes.CATEGORY,
                            },
                        })
                        .url();

                    viewStandardLink = (
                        <Link to={linkTo} className="no-underline">
                            <button className="btn-sm btn-base" type="button">
                                View Standard
                            </button>
                        </Link>
                    );

                    contents = (
                        <Sunburst
                            data={sunburstData}
                            rootData={sunburstRootData}
                            legendData={sunburstLegendData}
                            totalValue={totalPassing}
                            key={entityId}
                        />
                    );
                } else if (
                    data?.checks?.results?.length === 0 &&
                    data?.results?.results?.length === 0
                ) {
                    contents = (
                        <div className="flex flex-1 items-center justify-center p-4 leading-loose">
                            No data available. Please run a scan.
                        </div>
                    );
                }

                return (
                    <Widget
                        className={`s-2 ${className}`}
                        header={`${standardName} Compliance`}
                        headerComponents={viewStandardLink}
                        id={`${standardBaseTypes[standardId]}-compliance`}
                    >
                        {contents}
                    </Widget>
                );
            }}
        </Query>
    );
};

ComplianceByStandard.propTypes = {
    standardName: PropTypes.string.isRequired,
    standardId: PropTypes.string.isRequired,
    entityName: PropTypes.string,
    entityType: PropTypes.string,
    entityId: PropTypes.string,
    className: PropTypes.string,
};

ComplianceByStandard.defaultProps = {
    entityId: null,
    entityType: null,
    entityName: null,
    className: '',
};

export default ComplianceByStandard;
