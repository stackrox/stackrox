import React from 'react';
import Widget from 'Components/Widget';
import Query from 'Components/ThrowingQuery';
import Loader from 'Components/Loader';
import PropTypes from 'prop-types';
import Gauge from 'Components/visuals/GaugeWithDetail';
import NoResultsMessage from 'Components/NoResultsMessage';
import entityTypes, { standardBaseTypes } from 'constants/entityTypes';
import { standardShortLabels } from 'messages/standards';
import { resourceLabels } from 'messages/common';
import { AGGREGATED_RESULTS } from 'queries/controls';
import URLService from 'modules/URLService';
import contextTypes from 'constants/contextTypes';
import pageTypes from 'constants/pageTypes';
import { CLIENT_SIDE_SEARCH_OPTIONS } from 'constants/searchOptions';
import queryService from 'modules/queryService';

const isStandard = type => !!standardBaseTypes[type];

const sortByTitle = (a, b) => {
    if (a.title < b.title) return -1;
    if (a.title > b.title) return 1;
    return 0;
};

function processData(entityType, query, { results, complianceStandards }) {
    let filteredResults;
    if (standardBaseTypes[entityType]) {
        filteredResults = results.results.filter(result =>
            result.aggregationKeys[0].id.includes(entityType)
        );
    } else {
        filteredResults = results.results;
    }
    if (!filteredResults.length)
        return [
            {
                id: entityType,
                title: entityType,
                passing: { value: 0, link: '' },
                failing: { value: 0, link: '' }
            }
        ];
    const standardDataMapping = filteredResults
        .filter(datum => !(datum.passing === 0 && datum.failing === 0))
        .reduce((accMapping, currValue) => {
            const newMapping = { ...accMapping };
            const { id: standardId } = currValue.aggregationKeys[0];
            const standard = complianceStandards.find(cs => cs.id === standardId);
            let { numPassing: totalPassing, numFailing: totalFailing } = currValue;
            if (newMapping[standardId]) {
                totalPassing += newMapping[standardId].passing.value;
                totalFailing += newMapping[standardId].failing.value;
            }
            const complianceStateKey = CLIENT_SIDE_SEARCH_OPTIONS.COMPLIANCE.STATE;
            const newQuery = { ...query };
            newQuery[complianceStateKey] = 'Pass';
            if (!isStandard(entityType)) newQuery.Standard = standard.name;
            const passingLink = URLService.getLinkTo(contextTypes.COMPLIANCE, pageTypes.LIST, {
                entityType,
                query: newQuery
            });
            newQuery[complianceStateKey] = 'Fail';
            if (!isStandard(entityType)) newQuery.Standard = standard.name;
            const failingLink = URLService.getLinkTo(contextTypes.COMPLIANCE, pageTypes.LIST, {
                entityType,
                query: newQuery
            });
            delete newQuery[complianceStateKey];
            delete newQuery.Standard;
            const defaultLink = URLService.getLinkTo(contextTypes.COMPLIANCE, pageTypes.LIST, {
                entityType,
                query: newQuery
            });
            newMapping[standardId] = {
                id: standard.id,
                title: standardShortLabels[standard.id],
                passing: {
                    value: totalPassing,
                    link: passingLink.url
                },
                failing: {
                    value: totalFailing,
                    link: failingLink.url
                },
                defaultLink: defaultLink.url
            };
            return newMapping;
        }, {});
    return Object.values(standardDataMapping).sort(sortByTitle);
}

const getQueryVariables = (entityType, groupBy, query) => {
    const where = queryService.objectToWhereClause(query);
    if (!isStandard(entityType)) {
        return {
            groupBy: [entityTypes.STANDARD, entityType],
            unit: entityType,
            where
        };
    }

    return {
        groupBy: [entityTypes.STANDARD, ...(groupBy ? [groupBy] : [])],
        unit: entityTypes.CONTROL,
        where
    };
};

const ComplianceAcrossEntities = ({ entityType, groupBy, query }) => {
    const variables = getQueryVariables(entityType, groupBy, query);
    return (
        <Query query={AGGREGATED_RESULTS} variables={variables}>
            {({ loading, data }) => {
                let contents = <Loader />;
                const headerText = standardBaseTypes[entityType]
                    ? `Controls in Compliance`
                    : `${resourceLabels[entityType]}s in Compliance`;
                if (!loading && data) {
                    const results = processData(entityType, query, data);
                    if (!results.length) {
                        contents = (
                            <NoResultsMessage message="No data available. Please run a scan." />
                        );
                    } else {
                        contents = <Gauge data={results} />;
                    }
                }
                return (
                    <Widget header={headerText} bodyClassName="p-2" id="compliance-across-entities">
                        {contents}
                    </Widget>
                );
            }}
        </Query>
    );
};

ComplianceAcrossEntities.propTypes = {
    entityType: PropTypes.string,
    groupBy: PropTypes.string,
    query: PropTypes.shape({})
};

ComplianceAcrossEntities.defaultProps = {
    entityType: null,
    groupBy: null,
    query: null
};

export default ComplianceAcrossEntities;
