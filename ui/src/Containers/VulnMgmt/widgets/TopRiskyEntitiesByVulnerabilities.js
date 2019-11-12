import React, { useState, useContext } from 'react';
import PropTypes from 'prop-types';
import gql from 'graphql-tag';
import pluralize from 'pluralize';
import { useQuery } from 'react-apollo';
import sortBy from 'lodash/sortBy';

import queryService from 'modules/queryService';
import workflowStateContext from 'Containers/workflowStateContext';
import Loader from 'Components/Loader';
import NoResultsMessage from 'Components/NoResultsMessage';
import ViewAllButton from 'Components/ViewAllButton';
import Widget from 'Components/Widget';
import Scatterplot from 'Components/visuals/Scatterplot';
import TextSelect from 'Components/TextSelect';
import entityTypes from 'constants/entityTypes';
import entityLabels from 'messages/entity';
import isGQLLoading from 'utils/gqlLoading';
import { severityColorMap, severityColorLegend } from 'constants/severityColors';
import {
    getSeverityByCvss,
    getPriorityFieldByEntity,
    getNameFieldByEntity
} from 'utils/vulnerabilityUtils';

const TopRiskyEntitiesByVulnerabilities = ({
    entityContext,
    defaultSelection,
    riskEntityTypes,
    small
}) => {
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
        .setSort([
            {
                id: getPriorityFieldByEntity(selectedEntityType),
                desc: false
            },
            {
                id: getNameFieldByEntity(selectedEntityType),
                desc: false
            }
        ])
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
                    id
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
            results: components(query: $query) {
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

        // 1. sort the vulns in reverse CVSS order
        const sortedVulns = sortBy(vulns, vuln => {
            return vuln.cvss;
        }).reverse();
        const topVulns = sortedVulns.slice(0, 100);

        // 2. grab up to the first 5 vulns (the ones with the highest CVSS)
        const total = topVulns.reduce((acc, curr) => {
            return acc + parseFloat(curr.cvss);
        }, 0);

        // 3. Take the average of those top 5 (or total, if less than 5)
        const avgScore = total / topVulns.length;

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
                    <div>{`Weighted CVSS: ${datum.avgSeverity}`}</div>
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
                const entityId = result.id || result.metadata.id;
                const url = workflowState.pushRelatedEntity(selectedEntityType, entityId).toUrl();
                const avgSeverity = getAverageSeverity(result.vulns);
                return {
                    x: result.vulnCount,
                    y: +avgSeverity,
                    color: severityColorMap[getSeverityByCvss(avgSeverity)],
                    hint: getHint({ ...result, avgSeverity }),
                    url
                };
            })
            .sort((a, b) => {
                return a.vulnCount - b.vulnCount;
            });

        return results;
    }
    let results = [];
    const variables = {
        query: queryService.entityContextToQueryString(entityContext)
    };
    const { data, loading } = useQuery(query, { variables });

    let content = <Loader />;

    if (!isGQLLoading(loading, data)) {
        results = processData(data);
        if (!results || results.length === 0) {
            content = (
                <NoResultsMessage
                    message={`No ${pluralize(
                        selectedEntityType.toLowerCase()
                    )} with vulnerabilities found`}
                    className="p-6"
                    icon="info"
                />
            );
        } else {
            content = (
                <Scatterplot
                    data={results}
                    xMultiple={10}
                    yMultiple={10}
                    yAxisTitle="Average CVSS Score"
                    xAxisTitle="Critical Vulnerabilities & Exposures"
                    legendData={!small && severityColorLegend}
                />
            );
        }
    }

    return (
        <Widget
            className="h-full pdf-page"
            titleComponents={titleComponents}
            headerComponents={viewAll}
            bodyClassName="pr-2"
        >
            {content}
        </Widget>
    );
};

TopRiskyEntitiesByVulnerabilities.propTypes = {
    entityContext: PropTypes.shape({}),
    defaultSelection: PropTypes.string.isRequired,
    riskEntityTypes: PropTypes.arrayOf(PropTypes.string),
    small: PropTypes.bool
};

TopRiskyEntitiesByVulnerabilities.defaultProps = {
    entityContext: {},
    riskEntityTypes: [
        entityTypes.DEPLOYMENT,
        entityTypes.NAMESPACE,
        entityTypes.IMAGE,
        entityTypes.CLUSTER
    ],
    small: false
};

export default TopRiskyEntitiesByVulnerabilities;
