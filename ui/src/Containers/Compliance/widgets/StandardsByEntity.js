import React from 'react';
import PropTypes from 'prop-types';
import entityTypes from 'constants/entityTypes';
import URLService from 'modules/URLService';
import pageTypes from 'constants/pageTypes';
import labels from 'messages/common';
import capitalize from 'lodash/capitalize';
import sortBy from 'lodash/sortBy';
import chunk from 'lodash/chunk';
import Widget from 'Components/Widget';
import Query from 'Components/ThrowingQuery';
import Loader from 'Components/Loader';
import VerticalBarChart from 'Components/visuals/VerticalClusterBar';
import NoResultsMessage from 'Components/NoResultsMessage';
import { standardLabels } from 'messages/standards';
import { AGGREGATED_RESULTS as QUERY } from 'queries/controls';
import contextTypes from 'constants/contextTypes';

function processData(data, entityType) {
    if (!data.results.results.length || !data.entityList) return [];
    const standardsGrouping = {};
    const { results, entityList, complianceStandards } = data;
    results.results.forEach(result => {
        const entity = entityList.find(
            entityObject => entityObject.id === result.aggregationKeys[1].id
        );
        const standard = complianceStandards.find(c => c.id === result.aggregationKeys[0].id);
        const { numPassing, numFailing } = result;
        const percentagePassing = Math.round((numPassing / (numPassing + numFailing)) * 100);
        const link = URLService.getLinkTo(contextTypes.COMPLIANCE, pageTypes.LIST, {
            entityType: standard.id,
            query: {
                [`${capitalize(entityType)}`]: entity.name
            }
        });
        const dataPoint = {
            x: entity.name,
            y: percentagePassing,
            hint: {
                title: standardLabels[standard.id],
                body: `${numFailing} controls failing in this ${labels.resourceLabels[entityType]}`
            },
            link
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

    Object.keys(standardsGrouping).forEach(standard => {
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

function getLabelLinks(data, entityType) {
    const { entityList } = data;
    const labelLinks = {};
    entityList.forEach(entity => {
        const link = URLService.getLinkTo(contextTypes.COMPLIANCE, pageTypes.ENTITY, {
            entityType,
            entityId: entity.id
        });
        labelLinks[entity.name] = link;
    });
    return labelLinks;
}

const StandardsByEntity = ({ entityType, bodyClassName, className }) => {
    const variables = {
        groupBy: [entityTypes.STANDARD, entityType],
        unit: entityTypes.CONTROL
    };
    return (
        <Query query={QUERY} variables={variables}>
            {({ loading, data }) => {
                let contents = <Loader />;
                const headerText = `Passing standards by ${entityType}`;
                let pages;
                if (!loading || data.results) {
                    const formattedData = {
                        results: data.results,
                        complianceStandards: data.complianceStandards,
                        entityList: data.clusters
                    };
                    const results = processData(formattedData, entityType);
                    const labelLinks = getLabelLinks(formattedData, entityType);
                    pages = results.length;

                    if (pages) {
                        const VerticalBarChartPaged = ({ currentPage }) => (
                            <VerticalBarChart data={results[currentPage]} labelLinks={labelLinks} />
                        );
                        VerticalBarChartPaged.propTypes = { currentPage: PropTypes.number };
                        VerticalBarChartPaged.defaultProps = { currentPage: 0 };
                        contents = <VerticalBarChartPaged />;
                    } else {
                        contents = (
                            <NoResultsMessage message="No data available. Please run a scan." />
                        );
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
            }}
        </Query>
    );
};
StandardsByEntity.propTypes = {
    entityType: PropTypes.string.isRequired,
    bodyClassName: PropTypes.string,
    className: PropTypes.string
};

StandardsByEntity.defaultProps = {
    bodyClassName: 'p-4',
    className: ''
};

export default StandardsByEntity;
