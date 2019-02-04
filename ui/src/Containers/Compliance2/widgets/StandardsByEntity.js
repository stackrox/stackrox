import React from 'react';
import componentTypes from 'constants/componentTypes';
import Widget from 'Components/Widget';
import Query from 'Components/AppQuery';
import Loader from 'Components/Loader';
import PropTypes from 'prop-types';
import VerticalBarChart from 'Components/visuals/VerticalClusterBar';
import { resourceTypes } from 'constants/entityTypes';
import pluralize from 'pluralize';
import capitalize from 'lodash/capitalize';
import URLService from 'modules/URLService';
import pageTypes from 'constants/pageTypes';

const componentTypeMapping = {
    [resourceTypes.CLUSTERS]: componentTypes.STANDARDS_BY_CLUSTER
};

function processData(data, type, params) {
    if (!data.results || !data.entityList) return [];
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
                [capitalize(pluralize.singular(type))]: entity.name
            }
        });
        const dataPoint = {
            x: entity.name,
            y: percentagePassing,
            hint: {
                title: standard.id,
                body: `${numFailing} controls failing in this ${pluralize.singular(type)}`
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

const StandardsByEntity = ({ type, params }) => (
    <Query params={params} componentType={componentTypeMapping[type]}>
        {({ loading, data }) => {
            let contents = <Loader />;
            const headerText = `Standards By ${type}`;
            let pages;
            if (!loading && data && data.results) {
                const results = processData(data, type, params);
                const labelLinks = getLabelLinks(data, type, params);
                pages = results.length;

                const VerticalBarChartPaged = ({ currentPage }) => (
                    <VerticalBarChart data={results[currentPage]} labelLinks={labelLinks} />
                );
                VerticalBarChartPaged.propTypes = { currentPage: PropTypes.number };
                VerticalBarChartPaged.defaultProps = { currentPage: 0 };
                contents = <VerticalBarChartPaged />;
            }

            return (
                <Widget pages={pages} header={headerText}>
                    {contents}
                </Widget>
            );
        }}
    </Query>
);

StandardsByEntity.propTypes = {
    type: PropTypes.string.isRequired,
    params: PropTypes.shape({}).isRequired
};

export default StandardsByEntity;
