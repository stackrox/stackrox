import React, { useContext } from 'react';
import Widget from 'Components/Widget';
import Query from 'Components/CacheFirstQuery';
import Loader from 'Components/Loader';
import PropTypes from 'prop-types';
import Gauge from 'Components/visuals/GaugeWithDetail';
import NoResultsMessage from 'Components/NoResultsMessage';
import entityTypes, { standardBaseTypes } from 'constants/entityTypes';
import { standardShortLabels, standardLabels } from 'messages/standards';
import { resourceLabels } from 'messages/common';
import { AGGREGATED_RESULTS } from 'queries/controls';
import URLService from 'modules/URLService';
import { CLIENT_SIDE_SEARCH_OPTIONS } from 'constants/searchOptions';
import queryService from 'modules/queryService';
import { withRouter } from 'react-router-dom';
import ReactRouterPropTypes from 'react-router-prop-types';
import searchContext from 'Containers/searchContext';

const sortByTitle = (a, b) => {
    if (a.title < b.title) return -1;
    if (a.title > b.title) return 1;
    return 0;
};

function processData(
    match,
    location,
    entityType,
    { results, controls, complianceStandards },
    searchParam
) {
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
    const standardDataMapping = filteredResults.reduce((accMapping, result) => {
        const newMapping = { ...accMapping };
        const { id: standardId } = result.aggregationKeys[0];
        const standard = complianceStandards.find(cs => cs.id === standardId);
        let { numPassing: totalPassing, numFailing: totalFailing } = result;
        let totalSkipped = !(totalPassing + totalFailing > 0) ? 1 : 0;
        if (newMapping[standardId]) {
            totalPassing += newMapping[standardId].passing.value;
            totalFailing += newMapping[standardId].failing.value;
            totalSkipped += newMapping[standardId].skipped;
        }
        const complianceStateKey = CLIENT_SIDE_SEARCH_OPTIONS.COMPLIANCE.STATE;
        const currentUrl = URLService.getURL(match, location).url();

        const defaultLink = URLService.getURL(match, location)
            .push(entityType)
            .query({ [searchParam]: { [complianceStateKey]: undefined, standard: undefined } })
            .url();

        const passingLink = URLService.getURL(match, location)
            .push(entityType)
            .query({
                [searchParam]: {
                    [complianceStateKey]: 'Pass',
                    standard: standardLabels[standard.id]
                }
            })
            .url();

        const failingLink = URLService.getURL(match, location)
            .push(entityType)
            .query({
                [searchParam]: {
                    [complianceStateKey]: 'Fail',
                    standard: standardLabels[standard.id]
                }
            })
            .url();

        newMapping[standardId] = {
            id: standard.id,
            title: standardShortLabels[standard.id],
            passing: {
                value: totalPassing,
                controls: 0,
                link: passingLink,
                selected: currentUrl === passingLink
            },
            failing: {
                value: totalFailing,
                controls: 0,
                link: failingLink,
                selected: currentUrl === failingLink
            },
            skipped: totalSkipped,
            defaultLink
        };
        return newMapping;
    }, {});
    controls.results.forEach(({ numPassing, numFailing, aggregationKeys }) => {
        const { id: standardId } = aggregationKeys[0];
        standardDataMapping[standardId].passing.controls += numPassing;
        standardDataMapping[standardId].failing.controls += numFailing;
    });

    return Object.values(standardDataMapping).sort(sortByTitle);
}

const getQueryVariables = (entityType, groupBy, query) => {
    if (entityType !== entityTypes.CONTROL) {
        const queryWithoutStandard = { ...query };
        delete queryWithoutStandard.standard;
        delete queryWithoutStandard.Standard;
        return {
            groupBy: [entityTypes.STANDARD, entityType],
            unit: entityType === entityTypes.CONTROL ? entityTypes.CHECK : entityType,
            where: queryService.objectToWhereClause(queryWithoutStandard)
        };
    }

    return {
        groupBy: [entityTypes.STANDARD, ...(groupBy ? [groupBy] : [])],
        unit: entityTypes.CHECK,
        where: queryService.objectToWhereClause(query)
    };
};

const ComplianceAcrossEntities = ({ match, location, entityType, groupBy, query }) => {
    const variables = getQueryVariables(entityType, groupBy, query);
    const searchParam = useContext(searchContext);
    return (
        <Query query={AGGREGATED_RESULTS} variables={variables}>
            {({ loading, data }) => {
                let contents = <Loader />;
                const headerText = `${
                    standardBaseTypes[entityType] ? 'Control' : resourceLabels[entityType]
                }s in Compliance`;
                if (!loading && data) {
                    const results = processData(match, location, entityType, data, searchParam);
                    if (!results.length) {
                        contents = (
                            <NoResultsMessage message="No data available. Please run a scan." />
                        );
                    } else {
                        contents = <Gauge query={query} data={results} />;
                    }
                }
                return (
                    <Widget
                        header={headerText}
                        bodyClassName="flex-1 p-2"
                        id="compliance-across-entities"
                    >
                        {contents}
                    </Widget>
                );
            }}
        </Query>
    );
};

ComplianceAcrossEntities.propTypes = {
    match: ReactRouterPropTypes.match.isRequired,
    location: ReactRouterPropTypes.location.isRequired,
    entityType: PropTypes.string,
    groupBy: PropTypes.string,
    query: PropTypes.shape({})
};

ComplianceAcrossEntities.defaultProps = {
    entityType: null,
    groupBy: null,
    query: null
};

export default withRouter(ComplianceAcrossEntities);
