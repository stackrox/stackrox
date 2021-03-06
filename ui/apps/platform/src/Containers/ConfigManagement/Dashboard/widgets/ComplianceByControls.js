import React, { useState, useContext } from 'react';
import PropTypes from 'prop-types';
import ReactRouterPropTypes from 'react-router-prop-types';
import { gql } from '@apollo/client';
import queryService from 'utils/queryService';
import entityTypes, { standardEntityTypes, standardBaseTypes } from 'constants/entityTypes';
import { standardLabels } from 'messages/standards';
import { Link, withRouter } from 'react-router-dom';
import URLService from 'utils/URLService';
import searchContext from 'Containers/searchContext';
import networkStatuses from 'constants/networkStatuses';
import COMPLIANCE_STATES from 'constants/complianceStates';

import ScanButton from 'Containers/Compliance/ScanButton';
import Query from 'Components/CacheFirstQuery';
import Widget from 'Components/Widget';
import Loader from 'Components/Loader';
import Sunburst from 'Components/visuals/Sunburst';
import TextSelect from 'Components/TextSelect';
import NoResultsMessage from 'Components/NoResultsMessage';

const passingColor = 'var(--tertiary-400)';
const failingColor = 'var(--alert-400)';
const NAColor = 'var(--base-400)';

const passingTextColor = 'var(--tertiary-500)';
const failingTextColor = 'var(--alert-500)';
const NATextColor = 'var(--base-500)';

const sunburstLegendData = [
    { title: 'Passing', color: 'var(--tertiary-400)' },
    { title: 'Failing', color: 'var(--alert-400)' },
    { title: 'N/A', color: 'var(--base-400)' },
];

const QUERY = gql`
    query complianceByControls(
        $groupBy: [ComplianceAggregation_Scope!]
        $unit: ComplianceAggregation_Scope!
        $where: String
    ) {
        aggregatedResults(groupBy: $groupBy, unit: $unit, where: $where) {
            results {
                aggregationKeys {
                    id
                    scope
                }
                numFailing
                numPassing
                numSkipped
                keys {
                    ... on ComplianceControlGroup {
                        id
                        name
                        description
                    }
                    ... on ComplianceControl {
                        id
                        name
                        description
                    }
                }
            }
        }
    }
`;

const getPercentagePassing = (numPassing, numFailing) => {
    if (numPassing === 0 && numFailing === 0) {
        return 0;
    }
    return Math.floor((numPassing / (numPassing + numFailing)) * 100);
};

const getCategoryControlMapping = (data) => {
    const categoryMapping = data.aggregatedResults.results.reduce((acc, curr) => {
        const { numPassing, numFailing } = curr;
        const [category, control] = curr.keys;
        if (acc[category.id]) {
            acc[category.id].controls = [
                ...acc[category.id].controls,
                { control, numPassing, numFailing },
            ];
        } else {
            acc[category.id] = {
                category,
                controls: [{ control, numPassing, numFailing }],
            };
        }
        return acc;
    }, {});
    return categoryMapping;
};

const getColor = (numPassing, numFailing) => {
    if (!numPassing && !numFailing) {
        return NAColor;
    }
    if (!numFailing) {
        return passingColor;
    }
    return failingColor;
};

const getTextColor = (numPassing, numFailing) => {
    if (!numPassing && !numFailing) {
        return NATextColor;
    }
    if (!numFailing) {
        return passingTextColor;
    }
    return failingTextColor;
};

const getSunburstData = (categoryMapping, urlBuilder, searchParam, standardType) => {
    const categories = Object.keys(categoryMapping);
    const data = categories.map((categoryId) => {
        const { category, controls } = categoryMapping[categoryId];
        const { totalPassing, totalFailing } = controls.reduce(
            (acc, curr) => {
                acc.totalPassing += curr.numPassing;
                acc.totalFailing += curr.numFailing;
                return acc;
            },
            { totalPassing: 0, totalFailing: 0 }
        );
        const categoryValue = getPercentagePassing(totalPassing, totalFailing);
        return {
            name: `${category.name}. ${category.description}`,
            color: getColor(totalPassing, totalFailing),
            textColor: getTextColor(totalPassing, totalFailing),
            value: categoryValue,
            children: controls.map(({ control, numPassing, numFailing }) => {
                const value = getPercentagePassing(numPassing, numFailing);
                const link = urlBuilder
                    .base(entityTypes.CONTROL)
                    .push(control.id)
                    .query({
                        [searchParam]: {
                            standard: standardLabels[standardType],
                            'Compliance State': undefined,
                        },
                    })
                    .url();
                return {
                    name: `${control.name} - ${control.description}`,
                    color: getColor(numPassing, numFailing),
                    textColor: getTextColor(numPassing, numFailing),
                    value,
                    link,
                };
            }),
        };
    });
    return data;
};

const getTotalPassingFailing = (data) => {
    const result = data.aggregatedResults.results.reduce(
        (acc, curr) => {
            const { numPassing, numFailing } = curr;
            const value = getPercentagePassing(numPassing, numFailing);
            if (value === 100) {
                acc.controlsPassing += 1;
            } else if (!numPassing && !numFailing) {
                acc.controlsNA += 1;
            } else {
                acc.controlsFailing += 1;
            }
            return acc;
        },
        { controlsPassing: 0, controlsFailing: 0, controlsNA: 0 }
    );
    return result;
};

const getSunburstRootData = (
    controlsPassing,
    controlsFailing,
    controlsNA,
    urlBuilder,
    standardType,
    searchParam
) => {
    const controlsPassingLink = urlBuilder
        .base(entityTypes.CONTROL)
        .query({
            [searchParam]: {
                standard: standardLabels[standardType],
                'Compliance State': COMPLIANCE_STATES.PASS,
            },
        })
        .url();

    const controlsFailingLink = urlBuilder
        .base(entityTypes.CONTROL)
        .query({
            [searchParam]: {
                standard: standardLabels[standardType],
                'Compliance State': COMPLIANCE_STATES.FAIL,
            },
        })
        .url();

    const controlsNALink = urlBuilder
        .base(entityTypes.CONTROL)
        .query({
            [searchParam]: {
                standard: standardLabels[standardType],
                'Compliance State': COMPLIANCE_STATES['N/A'],
            },
        })
        .url();

    const sunburstRootData = [
        {
            text: `${controlsPassing} Controls Passing`,
            link: controlsPassingLink,
            className: 'text-tertiary-700',
        },
        {
            text: `${controlsFailing} Controls Failing`,
            link: controlsFailingLink,
            className: 'text-alert-700',
        },
        {
            text: `${controlsNA} Controls N/A`,
            link: controlsNALink,
            className: 'text-base-700',
        },
    ];
    return sunburstRootData;
};

const getSunburstProps = (data, urlBuilder, standardType, searchParam) => {
    const categoryMapping = getCategoryControlMapping(data);
    const { controlsPassing, controlsFailing, controlsNA } = getTotalPassingFailing(data);
    const sunburstRootData = getSunburstRootData(
        controlsPassing,
        controlsFailing,
        controlsNA,
        urlBuilder,
        standardType,
        searchParam
    );
    const sunburstData = getSunburstData(categoryMapping, urlBuilder, searchParam, standardType);
    return {
        sunburstData,
        sunburstRootData,
        totalPassing: getPercentagePassing(controlsPassing, controlsFailing),
    };
};

const ViewStandardButton = ({ standardType, searchParam, urlBuilder }) => {
    const linkTo = urlBuilder
        .base(entityTypes.CONTROL)
        .query({
            [searchParam]: {
                standard: standardLabels[standardType],
                groupBy: entityTypes.CATEGORY,
            },
        })
        .url();

    const viewStandardLink = (
        <Link to={linkTo} className="no-underline">
            <button className="btn-sm btn-base" type="button">
                View Standard
            </button>
        </Link>
    );
    return viewStandardLink;
};

const ComplianceByControls = ({
    match,
    location,
    className,
    standardOptions,
    isConfigMangement,
}) => {
    const searchParam = useContext(searchContext);
    const options = standardOptions.map((standard) => ({
        label: standardLabels[standard],
        jsonpath: standardLabels[standard],
        value: standardLabels[standard],
        standard,
    }));
    const [selectedStandard, selectStandard] = useState(options[0]);

    function onChange(datum) {
        const standard = options.find((option) => option.value === datum);
        selectStandard(standard);
    }

    const variables = {
        groupBy: [standardEntityTypes.CATEGORY, standardEntityTypes.CONTROL],
        unit: standardEntityTypes.CONTROL,
        where: queryService.objectToWhereClause({ Standard: selectedStandard.value }),
    };

    return (
        <Query query={QUERY} variables={variables}>
            {({ data, networkStatus }) => {
                const titleComponents = (
                    <TextSelect
                        value={selectedStandard.value}
                        onChange={onChange}
                        options={options}
                    />
                );

                const headerComponents = (
                    <div className="flex">
                        {isConfigMangement && (
                            <ScanButton
                                key={selectedStandard.standard}
                                className="btn-sm btn-base mr-2"
                                text="Scan"
                                textClass="hidden lg:block"
                                textCondensed={`Scan ${
                                    standardBaseTypes[selectedStandard.standard]
                                }`}
                                clusterId="*"
                                standardId={selectedStandard.standard}
                                loaderSize={10}
                            />
                        )}
                        <ViewStandardButton
                            urlBuilder={URLService.getURL(match, location)}
                            standardType={selectedStandard.standard}
                            searchParam={searchParam}
                        />
                    </div>
                );
                let contents = <Loader />;
                if (data && networkStatus === networkStatuses.READY) {
                    if (data.aggregatedResults.results.length) {
                        const { sunburstData, sunburstRootData, totalPassing } = getSunburstProps(
                            data,
                            URLService.getURL(match, location),
                            selectedStandard.standard,
                            searchParam
                        );
                        contents = (
                            <Sunburst
                                data={sunburstData}
                                rootData={sunburstRootData}
                                legendData={sunburstLegendData}
                                totalValue={totalPassing}
                                key={selectedStandard.value}
                            />
                        );
                    } else {
                        contents = (
                            <NoResultsMessage message="No data available. Please run a scan." />
                        );
                    }
                }
                return (
                    <Widget
                        className={`s-2 ${className}`}
                        id="compliance-by-controls"
                        titleComponents={titleComponents}
                        headerComponents={headerComponents}
                    >
                        {contents}
                    </Widget>
                );
            }}
        </Query>
    );
};

ComplianceByControls.propTypes = {
    match: ReactRouterPropTypes.match.isRequired,
    location: ReactRouterPropTypes.location.isRequired,
    className: PropTypes.string,
    standardOptions: PropTypes.arrayOf(PropTypes.shape).isRequired,
    isConfigMangement: PropTypes.string,
};

ComplianceByControls.defaultProps = {
    className: '',
    isConfigMangement: 'false',
};

export default withRouter(ComplianceByControls);
