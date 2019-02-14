import React from 'react';
import Widget from 'Components/Widget';
import Query from 'Components/ThrowingQuery';
import Loader from 'Components/Loader';
import PropTypes from 'prop-types';
import Gauge from 'Components/visuals/GaugeWithDetail';
import NoResultsMessage from 'Components/NoResultsMessage';
import { standardBaseTypes } from 'constants/entityTypes';
import { standardShortLabels } from 'messages/standards';
import { AGGREGATED_RESULTS } from 'queries/controls';

const isStandard = type => !!standardBaseTypes[type];

const sortByTitle = (a, b) => {
    if (a.title < b.title) return -1;
    if (a.title > b.title) return 1;
    return 0;
};

function processData(type, { results, complianceStandards }) {
    let filteredResults;
    if (standardBaseTypes[type]) {
        filteredResults = results.results.filter(result =>
            result.aggregationKeys[0].id.includes(type)
        );
    } else {
        filteredResults = results.results;
    }
    if (!filteredResults.length) return [{ title: type, passing: 0, failing: 0 }];
    const standardDataMapping = filteredResults
        .filter(datum => !(datum.passing === 0 && datum.failing === 0))
        .reduce((accMapping, currValue) => {
            const newMapping = { ...accMapping };
            const { id: standardId } = currValue.aggregationKeys[0];
            const standard = complianceStandards.find(cs => cs.id === standardId);
            let { numPassing: totalPassing, numFailing: totalFailing } = currValue;
            if (newMapping[standardId]) {
                totalPassing += newMapping[standardId].passing;
                totalFailing += newMapping[standardId].failing;
            }
            newMapping[standardId] = {
                title: standardShortLabels[standard.id],
                passing: totalPassing,
                failing: totalFailing
            };
            return newMapping;
        }, {});
    return Object.values(standardDataMapping).sort(sortByTitle);
}

const getQueryVariables = params => {
    const groupBy = ['STANDARD'];
    if (params.query && params.query.groupBy) {
        groupBy.push(params.query.groupBy);
    } else if (!isStandard(params.entityType)) {
        groupBy.push(params.entityType);
    }
    return {
        groupBy,
        unit: 'CONTROL'
    };
};

const ComplianceAcrossEntities = ({ params }) => {
    const variables = getQueryVariables(params);
    return (
        <Query query={AGGREGATED_RESULTS} variables={variables}>
            {({ loading, data }) => {
                let contents = <Loader />;
                const headerText = standardBaseTypes[params.entityType]
                    ? `Compliance Across ${standardShortLabels[params.entityType]} Controls`
                    : `Compliance Across ${params.entityType}s`;
                if (!loading && data) {
                    const results = processData(params.entityType, data);
                    if (!results.length) {
                        contents = (
                            <NoResultsMessage message="No data available. Please run a scan." />
                        );
                    } else {
                        contents = <Gauge data={results} dataProperty="passing" />;
                    }
                }
                return (
                    <Widget header={headerText} bodyClassName="p-2">
                        {contents}
                    </Widget>
                );
            }}
        </Query>
    );
};

ComplianceAcrossEntities.propTypes = {
    params: PropTypes.shape({}).isRequired
};

export default ComplianceAcrossEntities;
