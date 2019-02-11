import React from 'react';
import componentTypes from 'constants/componentTypes';
import Widget from 'Components/Widget';
import Query from 'Components/AppQuery';
import Loader from 'Components/Loader';
import PropTypes from 'prop-types';
import Gauge from 'Components/visuals/GaugeWithDetail';
import NoResultsMessage from 'Components/NoResultsMessage';
import { standardTypes } from 'constants/entityTypes';
import { standardShortLabels } from 'messages/standards';

const isStandard = type => Object.values(standardTypes).includes(type);

const sortByTitle = (a, b) => {
    if (a.title < b.title) return -1;
    if (a.title > b.title) return 1;
    return 0;
};

function processData(type, { results, complianceStandards }) {
    let filteredResults;
    if (isStandard(type)) {
        filteredResults = results.results.filter(result =>
            result.aggregationKeys[0].id.includes(type)
        );
    } else {
        filteredResults = results.results;
    }
    if (!filteredResults.length) return [{ title: type, passing: 0, failing: 0 }];
    return filteredResults
        .map(result => {
            const { numPassing, numFailing, aggregationKeys } = result;
            const standard = complianceStandards.find(cs => cs.id === aggregationKeys[0].id);
            const dataPoint = {
                title: standardShortLabels[standard.id],
                passing: numPassing,
                failing: numFailing
            };
            return dataPoint;
        })
        .filter(datum => !(datum.passing === 0 && datum.failing === 0))
        .sort(sortByTitle);
}

const ComplianceAcrossEntities = ({ params, pollInterval }) => (
    <Query
        params={params}
        componentType={
            isStandard(params.entityType)
                ? componentTypes.COMPLIANCE_ACROSS_STANDARDS
                : componentTypes.COMPLIANCE_ACROSS_RESOURCES
        }
        pollInterval={pollInterval}
    >
        {({ loading, data }) => {
            let contents = <Loader />;
            const headerText = isStandard(params.entityType)
                ? `Compliance Across ${standardShortLabels[params.entityType]} Controls`
                : `Compliance Across ${params.entityType}s`;
            if (!loading && data) {
                const results = processData(params.entityType, data);
                if (!results.length) {
                    contents = <NoResultsMessage message="No data available. Please run a scan." />;
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

ComplianceAcrossEntities.propTypes = {
    params: PropTypes.shape({}).isRequired,
    pollInterval: PropTypes.number
};

ComplianceAcrossEntities.defaultProps = {
    pollInterval: 0
};

export default ComplianceAcrossEntities;
