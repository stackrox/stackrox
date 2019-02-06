import React from 'react';
import PropTypes from 'prop-types';
import componentTypes from 'constants/componentTypes';
import { resourceTypes } from 'constants/entityTypes';
import URLService from 'modules/URLService';
import pageTypes from 'constants/pageTypes';
import resourceLabels from 'messages/common';
import capitalize from 'lodash/capitalize';

import Widget from 'Components/Widget';
import Query from 'Components/AppQuery';
import Loader from 'Components/Loader';
import VerticalBarChart from 'Components/visuals/VerticalClusterBar';
import NoResultsMessage from 'Components/NoResultsMessage';

const componentTypeMapping = {
    [resourceTypes.CLUSTER]: componentTypes.STANDARDS_BY_CLUSTER
};

function processData(data, type, params) {
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
        const link = URLService.getLinkTo(params.context, pageTypes.LIST, {
            entityType: standard.id,
            query: {
                [`${capitalize(type)}`]: entity.name
            }
        });
        const dataPoint = {
            x: entity.name,
            y: percentagePassing,
            hint: {
                title: standard.id,
                body: `${numFailing} controls failing in this ${resourceLabels[type]}`
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
    return [standardsGrouping];
}

function getLabelLinks(data, type, params) {
    const { entityList } = data;
    const labelLinks = {};
    entityList.forEach(entity => {
        const link = URLService.getLinkTo(params.context, pageTypes.ENTITY, {
            entityType: type,
            entityId: entity.id
        });
        labelLinks[entity.name] = link;
    });
    return labelLinks;
}

const StandardsByEntity = ({ type, params, pollInterval }) => (
    <Query params={params} componentType={componentTypeMapping[type]} pollInterval={pollInterval}>
        {({ loading, data }) => {
            let contents = <Loader />;
            const headerText = `Standards By ${type}`;
            let pages;
            if (!loading || data.results) {
                const results = processData(data, type, params);
                const labelLinks = getLabelLinks(data, type, params);
                pages = results.length;

                if (pages) {
                    const VerticalBarChartPaged = ({ currentPage }) => (
                        <VerticalBarChart data={results[currentPage]} labelLinks={labelLinks} />
                    );
                    VerticalBarChartPaged.propTypes = { currentPage: PropTypes.number };
                    VerticalBarChartPaged.defaultProps = { currentPage: 0 };
                    contents = <VerticalBarChartPaged />;
                } else {
                    contents = <NoResultsMessage message="No Data Available" />;
                }
            }

            return (
                <Widget className="sx-2 sy-2" pages={pages} header={headerText} bodyClassName="p-4">
                    {contents}
                </Widget>
            );
        }}
    </Query>
);

StandardsByEntity.propTypes = {
    type: PropTypes.string.isRequired,
    params: PropTypes.shape({}).isRequired,
    pollInterval: PropTypes.number
};

StandardsByEntity.defaultProps = {
    pollInterval: 0
};

export default StandardsByEntity;
