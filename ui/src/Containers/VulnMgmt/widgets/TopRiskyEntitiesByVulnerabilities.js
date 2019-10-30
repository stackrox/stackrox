import React, { useState, useContext } from 'react';
import gql from 'graphql-tag';
import pluralize from 'pluralize';
import { useQuery } from 'react-apollo';

import workflowStateContext from 'Containers/workflowStateContext';
import ViewAllButton from 'Components/ViewAllButton';
import Widget from 'Components/Widget';
import Scatterplot from 'Components/visuals/Scatterplot';
import TextSelect from 'Components/TextSelect';
import entityTypes from 'constants/entityTypes';
import entityLabels from 'messages/entity';
import isGQLLoading from 'utils/gqlLoading';
import { severityColorMap } from 'constants/severityColors';
import { getSeverityByCvss } from 'utils/vulnerabilityUtils';
import { severities } from 'constants/severities';
import PropTypes from 'prop-types';

const TopRiskyEntitiesByVulnerabilities = ({ defaultSelection, riskEntityTypes }) => {
    const workflowState = useContext(workflowStateContext);
    // Entity Type selection
    const [selectedEntityType, setEntityType] = useState(defaultSelection);
    const entityOptions = riskEntityTypes.map(entityType => ({
        label: `top risky ${pluralize(entityLabels[entityType])} by CVE count & CVSS score`,
        value: entityType
    }));
    function onChange(datum) {
        setEntityType(datum);
    }

    // View all button
    const viewAllUrl = workflowState
        .pushList(selectedEntityType)
        .setSort('risk')
        .toUrl();

    const titleComponents = (
        <TextSelect value={selectedEntityType} onChange={onChange} options={entityOptions} />
    );
    const viewAll = <ViewAllButton url={viewAllUrl} />;

    // Data Queries
    const VULN_FRAGMENT = gql`
        fragment vulnFields on EmbeddedVulnerability {
            cve
            cvss
            isFixable
            severity
        }
    `;
    const DEPLOYMENT_QUERY = gql`
        query topRiskyDeployments($query: String) {
            results: deployments(query: $query) {
                id
                name
                clusterName
                namespaceName: namespace
                vulnCount
                vulns {
                    ...vulnFields
                }
            }
        }
        ${VULN_FRAGMENT}
    `;

    const CLUSTER_QUERY = gql`
        query topRiskyClusters($query: String) {
            results: clusters(query: $query) {
                id
                name
                vulnCount
                vulns {
                    ...vulnFields
                }
            }
        }
        ${VULN_FRAGMENT}
    `;

    const NAMESPACE_QUERY = gql`
        query topRiskyNamespaces($query: String) {
            results: namespaces(query: $query) {
                metadata {
                    clusterName
                    name
                }
                vulnCount
                vulns {
                    ...vulnFields
                }
            }
        }
        ${VULN_FRAGMENT}
    `;

    const IMAGE_QUERY = gql`
        query topRiskyImages($query: String) {
            results: images(query: $query) {
                id
                name {
                    fullName
                }
                vulnCount
                vulns {
                    ...vulnFields
                }
            }
        }
        ${VULN_FRAGMENT}
    `;

    const COMPONENT_QUERY = gql`
        query topRiskyComponents($query: String) {
            results: imageComponents(query: $query) {
                id
                name
                vulnCount
                vulns {
                    ...vulnFields
                }
            }
        }
        ${VULN_FRAGMENT}
    `;

    const queryMap = {
        [entityTypes.DEPLOYMENT]: DEPLOYMENT_QUERY,
        [entityTypes.NAMESPACE]: NAMESPACE_QUERY,
        [entityTypes.CLUSTER]: CLUSTER_QUERY,
        [entityTypes.COMPONENT]: COMPONENT_QUERY,
        [entityTypes.IMAGE]: IMAGE_QUERY
    };
    const query = queryMap[selectedEntityType];

    function getAverageSeverity(vulns) {
        if (vulns.length === 0) return 0;
        const total = vulns.reduce((acc, curr) => {
            return acc + parseFloat(curr.cvss);
        }, 0);
        const avgScore = total / vulns.length;
        return avgScore.toFixed(1);
    }

    function getHint(datum) {
        return {
            title:
                (datum.name && datum.name.fullName) ||
                datum.name ||
                (datum.metadata && datum.metadata.name),
            body: (
                <div>
                    <div>{`AVG CVSS score: ${datum.avgSeverity}`}</div>
                    <div>{`CVEs: ${datum.vulnCount}`}</div>
                </div>
            ),
            clusterName: datum.clusterName || (datum.metadata && datum.metadata.clusterName),
            namespaceName: datum.namespaceName
        };
    }
    function processData(data) {
        if (!data || !data.results) return [];

        const results = data.results
            .filter(datum => datum.vulnCount > 0)
            .map(result => {
                const avgSeverity = getAverageSeverity(result.vulns);
                return {
                    x: result.vulnCount,
                    y: +avgSeverity,
                    color: severityColorMap[getSeverityByCvss(avgSeverity)],
                    hint: getHint({ ...result, avgSeverity })
                };
            })
            .sort((a, b) => {
                return a.vulnCount - b.vulnCount;
            });
        return results;
    }
    let results = [];
    const { data, loading } = useQuery(query);
    if (!isGQLLoading(loading, data)) {
        results = processData(data);
    }

    return (
        <Widget
            className="h-full pdf-page"
            titleComponents={titleComponents}
            headerComponents={viewAll}
        >
            <Scatterplot
                data={results}
                xMultiple={10}
                yMultiple={10}
                yAxisTitle="Average CVSS Score"
                xAxisTitle="Critical Vulnerabilities & Exposures"
                legendData={[
                    { title: 'Low', color: severityColorMap[severities.LOW_SEVERITY] },
                    { title: 'Medium', color: severityColorMap[severities.MEDIUM_SEVERITY] },
                    { title: 'High', color: severityColorMap[severities.HIGH_SEVERITY] },
                    { title: 'Critical', color: severityColorMap[severities.CRITICAL_SEVERITY] }
                ]}
            />
        </Widget>
    );
};

TopRiskyEntitiesByVulnerabilities.propTypes = {
    defaultSelection: PropTypes.string.isRequired,
    riskEntityTypes: PropTypes.arrayOf(PropTypes.string)
};

TopRiskyEntitiesByVulnerabilities.defaultProps = {
    riskEntityTypes: [
        entityTypes.DEPLOYMENT,
        entityTypes.NAMESPACE,
        entityTypes.IMAGE,
        entityTypes.CLUSTER
    ]
};

export default TopRiskyEntitiesByVulnerabilities;
