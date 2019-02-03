import React from 'react';
import componentTypes from 'constants/componentTypes';
import Widget from 'Components/Widget';
import Query from 'Components/AppQuery';
import Loader from 'Components/Loader';
import PropTypes from 'prop-types';
import Gauge from 'Components/visuals/GaugeWithDetail';
import { standardTypes } from 'constants/entityTypes';
import standardLabels from 'messages/standards';

const isStandard = type => Object.keys(standardTypes).includes(type);

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
    return filteredResults.map(result => {
        const { numPassing, numFailing, aggregationKeys } = result;
        const standard = complianceStandards.find(cs => cs.id === aggregationKeys[0].id);
        const dataPoint = {
            title: standard.name,
            passing: numPassing,
            failing: numFailing
        };
        return dataPoint;
    });
}

const ComplianceAcrossEntities = ({ params }) => (
    <Query
        params={params}
        componentType={
            isStandard(params.entityType)
                ? componentTypes.COMPLIANCE_ACROSS_STANDARDS
                : componentTypes.COMPLIANCE_ACROSS_RESOURCES
        }
    >
        {({ loading, data }) => {
            let contents = <Loader />;
            const headerText = isStandard(params.entityType)
                ? `Compliance Across ${standardLabels[params.entityType]} Controls`
                : `Compliance Across ${standardLabels[params.entityType]}`;
            if (!loading && data) {
                const results = processData(params.entityType, data);

                contents = <Gauge data={results} dataProperty="passing" />;
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
    params: PropTypes.shape({}).isRequired
};

export default ComplianceAcrossEntities;
