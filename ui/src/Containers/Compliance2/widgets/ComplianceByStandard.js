import React from 'react';
import PropTypes from 'prop-types';
import componentTypes from 'constants/componentTypes';
import standardLabels from 'messages/standards';
import capitalize from 'lodash/capitalize';
import URLService from 'modules/URLService';
import pageTypes from 'constants/pageTypes';

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
    if (!data || !data.groupResults || !data.groupResults.results.length) return [];

    const groupMapping = {};

    const statsReducer = (statsMapping, { aggregationKeys, numPassing, numFailing }) => {
        const newMapping = { ...statsMapping };
        newMapping[`${aggregationKeys[1].id}`] = Math.round(
            (numPassing / (numFailing + numPassing)) * 100
        );
        return newMapping;
    };

    const groupStatsMapping = data.groupResults.results
        .filter(result => result.numPassing + result.numFailing > 0)
        .reduce(statsReducer, {});

    const controlStatsMapping = data.controlResults.results
        .filter(result => result.numPassing + result.numFailing > 0)
        .reduce(statsReducer, {});

    const { groups, controls } = data.complianceStandards.filter(datum => datum.id === type)[0];

    groups.forEach(datum => {
        const group = groupStatsMapping[datum.id];
        if (group !== undefined) {
            const value = group;
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
            const control = controlStatsMapping[datum.id];
            if (group !== undefined && control !== undefined) {
                const value = control;
                group.children.push({
                    name: `${datum.name} - ${datum.description}`,
                    color: getColor(value),
                    link: `/main/compliance2/${datum.standardId}/${datum.id}`,
                    value
                });
            }
        });

    return Object.values(groupMapping);
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
                    const sunburstData = processSunburstData(data, type);
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
