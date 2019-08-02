import React, { useState, useContext } from 'react';
import PropTypes from 'prop-types';
import ReactRouterPropTypes from 'react-router-prop-types';
import gql from 'graphql-tag';
import queryService from 'modules/queryService';
import entityTypes, { standardEntityTypes, standardBaseTypes } from 'constants/entityTypes';
import {
    standardLabels,
    standardShortLabels,
    getStandardAcrossEntityLabel
} from 'messages/standards';
import { Link, withRouter } from 'react-router-dom';
import URLService from 'modules/URLService';
import searchContext from 'Containers/searchContext';
import networkStatuses from 'constants/networkStatuses';

import ScanButton from 'Containers/Compliance/ScanButton';
import Query from 'Components/ThrowingQuery';
import Widget from 'Components/Widget';
import Loader from 'Components/Loader';
import Sunburst from 'Components/visuals/Sunburst';
import Select from 'Components/Select';
import NoResultsMessage from 'Components/NoResultsMessage';

const passingColor = 'var(--tertiary-400)';
const failingColor = 'var(--alert-400)';

const sunburstLegendData = [
    { title: 'Passing Controls', color: 'var(--tertiary-400)' },
    { title: 'Failing Controls', color: 'var(--alert-400)' }
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
            }
        }
    }
`;

const getPercentagePassing = (numPassing, numFailing) => {
    if (numPassing === 0 && numFailing === 0) return 0;
    return Math.floor((numPassing / (numPassing + numFailing)) * 100);
};

const getCategoryControlMapping = data => {
    const categoryMapping = data.aggregatedResults.results.reduce((acc, curr) => {
        const categoryID = curr.aggregationKeys[0].id;
        const controlID = curr.aggregationKeys[1].id;
        const { numPassing, numFailing } = curr;
        acc[categoryID] = [...(acc[categoryID] || []), { controlID, numPassing, numFailing }];
        return acc;
    }, {});
    return categoryMapping;
};

const getSunburstData = (categoryMapping, urlBuilder) => {
    const categories = Object.keys(categoryMapping);
    const data = categories.map(category => {
        const controls = categoryMapping[category];
        const totalValues = controls.reduce(
            (acc, curr) => {
                acc.totalPassing += curr.numPassing;
                acc.totalFailing += curr.numFailing;
                return acc;
            },
            { totalPassing: 0, totalFailing: 0 }
        );
        const categoryValue = getPercentagePassing(
            totalValues.totalPassing,
            totalValues.totalFailing
        );
        return {
            name: category,
            color: categoryValue === 100 ? passingColor : failingColor,
            value: categoryValue,
            children: controls.map(control => {
                const value = getPercentagePassing(control.numPassing, control.numFailing);
                const link = urlBuilder.base(entityTypes.CONTROL, control.controlID).url();
                return {
                    name: control.controlID,
                    color: value === 100 ? passingColor : failingColor,
                    value,
                    link
                };
            })
        };
    });
    return data;
};

const getTotalPassingFailing = data => {
    const result = data.aggregatedResults.results.reduce(
        (acc, curr) => {
            const { numPassing, numFailing } = curr;
            const value = getPercentagePassing(numPassing, numFailing);
            if (value === 100) acc.controlsPassing += 1;
            else acc.controlsFailing += 1;
            return acc;
        },
        { controlsPassing: 0, controlsFailing: 0 }
    );
    return result;
};

const getSunburstRootData = (
    controlsPassing,
    controlsFailing,
    urlBuilder,
    selectedStandard,
    searchParam
) => {
    const controlsPassingLink = urlBuilder
        .base(entityTypes.CONTROL)
        .query({
            [searchParam]: {
                standard: standardShortLabels[selectedStandard],
                'Compliance State': 'Pass'
            }
        })
        .url();

    const controlsFailingLink = urlBuilder
        .base(entityTypes.CONTROL)
        .query({
            [searchParam]: {
                standard: standardShortLabels[selectedStandard],
                'Compliance State': 'Fail'
            }
        })
        .url();

    const sunburstRootData = [
        {
            text: `${controlsPassing} Controls Passing`,
            link: controlsPassingLink,
            className: 'text-tertiary-700'
        },
        {
            text: `${controlsFailing} Controls Failing`,
            link: controlsFailingLink,
            className: 'text-alert-700'
        }
    ];
    return sunburstRootData;
};

const getSunburstProps = (data, urlBuilder, selectedStandard, searchParam) => {
    const categoryMapping = getCategoryControlMapping(data);
    const { controlsPassing, controlsFailing } = getTotalPassingFailing(data);
    const sunburstRootData = getSunburstRootData(
        controlsPassing,
        controlsFailing,
        urlBuilder,
        selectedStandard,
        searchParam
    );
    const sunburstData = getSunburstData(categoryMapping, urlBuilder);
    return {
        sunburstData,
        sunburstRootData,
        totalPassing: getPercentagePassing(controlsPassing, controlsFailing)
    };
};

const ViewStandardButton = ({ standardType, searchParam, urlBuilder }) => {
    const linkTo = urlBuilder
        .base(entityTypes.CONTROL)
        .query({
            [searchParam]: {
                standard: standardShortLabels[standardType],
                groupBy: entityTypes.CATEGORY
            }
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
    isConfigMangement
}) => {
    const searchParam = useContext(searchContext);
    const options = standardOptions.map(standard => ({
        label: `${getStandardAcrossEntityLabel(standard, entityTypes.CLUSTER, 'plural')}`,
        jsonpath: standardLabels[standard],
        value: standardLabels[standard],
        standard
    }));
    const [selectedStandard, selectStandard] = useState(options[0]);

    function onChange(datum) {
        selectStandard(datum);
    }

    const variables = {
        groupBy: [standardEntityTypes.CATEGORY, standardEntityTypes.CONTROL],
        unit: standardEntityTypes.CONTROL,
        where: queryService.objectToWhereClause({ Standard: selectedStandard.value })
    };

    return (
        <Query query={QUERY} variables={variables}>
            {({ data, networkStatus }) => {
                const urlBuilder = URLService.getURL(match, location);
                const titleComponents = (
                    <Select
                        className="bg-base-100 w-full focus:outline-none"
                        value={selectedStandard.value}
                        onChange={onChange}
                        options={options}
                    />
                );

                const headerComponents = (
                    <div className="flex">
                        {isConfigMangement && (
                            <ScanButton
                                className="btn-sm btn-base mr-2"
                                text={`Scan ${standardBaseTypes[selectedStandard.standard]}`}
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
                            urlBuilder={urlBuilder}
                            standardType={selectedStandard}
                            searchParam={searchParam}
                        />
                    </div>
                );
                let contents = <Loader />;
                if (data && networkStatus === networkStatuses.READY) {
                    if (data.aggregatedResults.results.length) {
                        const { sunburstData, sunburstRootData, totalPassing } = getSunburstProps(
                            data,
                            urlBuilder,
                            selectedStandard,
                            searchParam
                        );
                        contents = (
                            <Sunburst
                                data={sunburstData}
                                rootData={sunburstRootData}
                                legendData={sunburstLegendData}
                                totalValue={totalPassing}
                                key={selectedStandard.label}
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
    isConfigMangement: PropTypes.bool
};

ComplianceByControls.defaultProps = {
    className: '',
    isConfigMangement: false
};

export default withRouter(ComplianceByControls);
