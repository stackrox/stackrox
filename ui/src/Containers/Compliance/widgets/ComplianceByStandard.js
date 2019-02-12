import React from 'react';
import PropTypes from 'prop-types';
import componentTypes from 'constants/componentTypes';
import standardLabels from 'messages/standards';
import capitalize from 'lodash/capitalize';
import URLService from 'modules/URLService';
import pageTypes from 'constants/pageTypes';
import contextTypes from 'constants/contextTypes';

import Widget from 'Components/Widget';
import Sunburst from 'Components/visuals/Sunburst';
import Query from 'Components/AppQuery';
import Loader from 'Components/Loader';

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

    const statsReducer = (statsMapping, { aggregationKeys, numPassing, numFailing }) => {
        const groupsMapping = { ...statsMapping.groupsMapping };
        const controlsMapping = { ...statsMapping.controlsMapping };
        const groupKey = `${aggregationKeys[1].id}`;
        const controlKey = `${aggregationKeys[2].id}`;
        const group = groupsMapping[groupKey];
        const passing = group ? group.passing : 0;
        const total = group ? group.total : 0;
        groupsMapping[groupKey] = {
            passing: passing + numPassing,
            total: total + numPassing + numFailing
        };
        controlsMapping[controlKey] = {
            passing: numPassing,
            total: numPassing + numFailing
        };
        return {
            groupsMapping,
            controlsMapping
        };
    };

    const {
        groupsMapping: groupStatsMapping,
        controlsMapping: controlStatsMapping
    } = data.results.results
        .filter(result => result.numPassing + result.numFailing > 0)
        .reduce(statsReducer, { groupsMapping: {}, controlsMapping: {} });

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
                entityType: datum.standardId,
                entityId: datum.id
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

const constructURLWithQuery = (params, type, entityName) => {
    const newParams = { ...params };
    if (type) {
        newParams.query.Standard = standardLabels[type];
    }
    if (entityName) {
        const entityKey = capitalize(newParams.entityType);
        newParams.query[entityKey] = entityName;
    }
    return newParams;
};

const createURLLink = (params, type, entityName) => {
    const linkParams = {
        entityType: type,
        query: {}
    };
    if (entityName) {
        const entityKey = capitalize(params.entityType);
        linkParams.query[entityKey] = entityName;
    }
    const link = URLService.getLinkTo(params.context, pageTypes.LIST, linkParams);
    return link;
};

const ComplianceByStandard = ({ type, entityName, params, pollInterval }) => {
    const newParams = constructURLWithQuery(params, type, entityName);
    return (
        <Query
            params={newParams}
            componentType={componentTypes.COMPLIANCE_BY_STANDARD}
            pollInterval={pollInterval}
        >
            {({ loading, data }) => {
                let contents = <Loader />;
                const headerText = `${standardLabels[type]} Compliance`;
                if (!loading || data) {
                    const { sunburstData, totalPassing } = processSunburstData(data, type);
                    const link = createURLLink(params, type, entityName);
                    const sunburstRootData = [
                        {
                            text: `${sunburstData.length} Categories`
                        },
                        {
                            text: `${getNumControls(sunburstData)} Controls`,
                            link: link.url
                        }
                    ];

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
                            />
                        );
                    }
                }
                return (
                    <Widget className="s-2" header={headerText}>
                        {contents}
                    </Widget>
                );
            }}
        </Query>
    );
};

ComplianceByStandard.propTypes = {
    type: PropTypes.string.isRequired,
    entityName: PropTypes.string,
    params: PropTypes.shape({}),
    pollInterval: PropTypes.number
};

ComplianceByStandard.defaultProps = {
    params: null,
    entityName: null,
    pollInterval: 0
};

export default ComplianceByStandard;
