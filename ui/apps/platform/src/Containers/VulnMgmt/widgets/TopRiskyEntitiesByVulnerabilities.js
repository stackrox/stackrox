import React, { useState, useContext } from 'react';
import PropTypes from 'prop-types';
import pluralize from 'pluralize';
import { gql, useQuery } from '@apollo/client';

import queryService from 'utils/queryService';
import workflowStateContext from 'Containers/workflowStateContext';
import Loader from 'Components/Loader';
import NoResultsMessage from 'Components/NoResultsMessage';
import Widget from 'Components/Widget';
import TextSelect from 'Components/TextSelect';
import entityTypes from 'constants/entityTypes';
import { policySeverityColorMap } from 'constants/visuals/colors';
import { severityLabels as policySeverityLabels } from 'messages/common';
import { policySeverities } from 'types/policy.proto';
import { checkForPermissionErrorMessage } from 'utils/permissionUtils';
import { getSeverityByCvss } from 'utils/vulnerabilityUtils';
import { entitySortFieldsMap, cveSortFields } from 'constants/sortFields';
import { WIDGET_PAGINATION_START_OFFSET } from 'constants/workflowPages.constants';
import { entityPriorityField } from '../VulnMgmt.constants';

import { entityNounOrdinaryCasePlural } from '../entitiesForVulnerabilityManagement';
import Scatterplot from './Scatterplot';

import ViewAllButton from './ViewAllButton';

// Beware, policy instead of vulnerability severities because of getSeverityByCvss function!

const legendData = policySeverities.map((severity) => ({
    title: policySeverityLabels[severity],
    color: policySeverityColorMap[severity],
}));

const ENTITY_COUNT = 25;
const VULN_COUNT = 50;

// Data Queries
const IMAGE_VULN_FRAGMENT = gql`
    fragment vulnFields on ImageVulnerability {
        id
        cve
        cvss
        severity
    }
`;
const NODE_VULN_FRAGMENT = gql`
    fragment vulnFields on NodeVulnerability {
        id
        cve
        cvss
        severity
    }
`;

const DEPLOYMENT_QUERY = gql`
    query topRiskyDeployments(
        $query: String
        $vulnQuery: String
        $entityPagination: Pagination
        $vulnPagination: Pagination
    ) {
        results: deployments(query: $query, pagination: $entityPagination) {
            id
            name
            clusterName
            namespaceName: namespace
            priority
            plottedVulns: plottedImageVulnerabilities(query: $vulnQuery) {
                basicVulnCounter: basicImageVulnerabilityCounter {
                    all {
                        total
                        fixable
                    }
                }
                vulns: imageVulnerabilities(pagination: $vulnPagination) {
                    ...vulnFields
                }
            }
        }
    }
    ${IMAGE_VULN_FRAGMENT}
`;

const NODE_QUERY = gql`
    query topRiskyNodes(
        $query: String
        $vulnQuery: String
        $entityPagination: Pagination
        $vulnPagination: Pagination
    ) {
        results: nodes(query: $query, pagination: $entityPagination) {
            id
            name
            clusterName
            priority
            plottedVulns: plottedNodeVulnerabilities(query: $vulnQuery) {
                basicVulnCounter: basicNodeVulnerabilityCounter {
                    all {
                        total
                        fixable
                    }
                }
                vulns: nodeVulnerabilities(pagination: $vulnPagination) {
                    ...vulnFields
                }
            }
        }
    }
    ${NODE_VULN_FRAGMENT}
`;

const NAMESPACE_QUERY = gql`
    query topRiskyNamespaces(
        $query: String
        $vulnQuery: String
        $entityPagination: Pagination
        $vulnPagination: Pagination
    ) {
        results: namespaces(query: $query, pagination: $entityPagination) {
            metadata {
                clusterName
                name
                id
                priority
            }
            plottedVulns: plottedImageVulnerabilities(query: $vulnQuery) {
                basicVulnCounter: basicImageVulnerabilityCounter {
                    all {
                        total
                        fixable
                    }
                }
                vulns: imageVulnerabilities(pagination: $vulnPagination) {
                    ...vulnFields
                }
            }
        }
    }
    ${IMAGE_VULN_FRAGMENT}
`;

const IMAGE_QUERY = gql`
    query topRiskyImages(
        $query: String
        $vulnQuery: String
        $entityPagination: Pagination
        $vulnPagination: Pagination
    ) {
        results: images(query: $query, pagination: $entityPagination) {
            id
            name {
                fullName
            }
            priority
            plottedVulns: plottedImageVulnerabilities(query: $vulnQuery) {
                basicVulnCounter: basicImageVulnerabilityCounter {
                    all {
                        total
                        fixable
                    }
                }
                vulns: imageVulnerabilities(pagination: $vulnPagination) {
                    ...vulnFields
                }
            }
        }
    }
    ${IMAGE_VULN_FRAGMENT}
`;

const TopRiskyEntitiesByVulnerabilities = ({
    entityContext,
    defaultSelection,
    riskEntityTypes,
    cveFilter,
    small,
}) => {
    const workflowState = useContext(workflowStateContext);

    // Entity Type selection
    const [selectedEntityType, setEntityType] = useState(defaultSelection);
    const entityOptions = riskEntityTypes.map((entityType) => ({
        label: `Top risky ${entityNounOrdinaryCasePlural[entityType]} by CVE count and CVSS score`,
        value: entityType,
    }));
    function onChange(datum) {
        setEntityType(datum);
    }

    // View all button
    const viewAllUrl = workflowState
        .pushList(selectedEntityType)
        .setSort([
            {
                id: entitySortFieldsMap[selectedEntityType].PRIORITY,
                desc: false,
            },
            {
                id: entitySortFieldsMap[selectedEntityType].NAME,
                desc: false,
            },
        ])
        .toUrl();

    const titleComponents = (
        <TextSelect value={selectedEntityType} onChange={onChange} options={entityOptions} />
    );
    const viewAll = <ViewAllButton url={viewAllUrl} />;

    const queryMap = {
        [entityTypes.DEPLOYMENT]: DEPLOYMENT_QUERY,
        [entityTypes.NAMESPACE]: NAMESPACE_QUERY,
        [entityTypes.IMAGE]: IMAGE_QUERY,
        [entityTypes.NODE]: NODE_QUERY,
    };

    const query = queryMap[selectedEntityType];

    function getAverageSeverity(vulns) {
        if (vulns.length === 0) {
            return 0;
        }

        // 1. total the CVSS scores of the top X vulns returned
        const total = vulns.reduce((acc, curr) => {
            return acc + parseFloat(curr.cvss);
        }, 0);

        // 2. Take the average of those top 5 (or total, if less than 5)
        const avgScore = total / vulns.length;

        return avgScore.toFixed(1);
    }

    function processData(data) {
        if (!data || !data.results) {
            return [];
        }
        const results = data.results
            .filter((result) => !!result?.plottedVulns?.basicVulnCounter?.all?.total) // only show entities with CVEs
            .map((result) => {
                const entityId = result.id || result.metadata.id;
                const vulnCount = result?.plottedVulns?.basicVulnCounter?.all?.total;
                const url = workflowState.pushRelatedEntity(selectedEntityType, entityId).toUrl();
                const avgSeverity = getAverageSeverity(result.plottedVulns.vulns);
                const color = policySeverityColorMap[getSeverityByCvss(avgSeverity)];

                return {
                    x: vulnCount,
                    y: +avgSeverity,
                    color,
                    url,
                };
            })
            .sort((a, b) => {
                return a.vulnCount - b.vulnCount;
            });

        return results;
    }

    const vulnQuery = cveFilter === 'Fixable' ? { Fixable: true } : '';
    const variables = {
        query: queryService.entityContextToQueryString(entityContext),
        vulnQuery: queryService.objectToWhereClause(vulnQuery),
        entityPagination: queryService.getPagination(
            {
                id: entityPriorityField[selectedEntityType],
                desc: false,
            },
            WIDGET_PAGINATION_START_OFFSET,
            ENTITY_COUNT
        ),
        vulnPagination: queryService.getPagination(
            {
                id: cveSortFields.CVSS_SCORE,
                desc: true,
            },
            WIDGET_PAGINATION_START_OFFSET,
            VULN_COUNT
        ),
        scopeQuery: queryService.entityContextToQueryString(entityContext),
    };
    const { data, loading, error } = useQuery(query, { variables });

    let content = <Loader />;

    if (!loading) {
        if (error) {
            const defaultMessage = `An error occurred in retrieving ${pluralize(
                selectedEntityType.toLowerCase()
            )}. Please refresh the page. If this problem continues, please contact support.`;

            const parsedMessage = checkForPermissionErrorMessage(error, defaultMessage);

            content = <NoResultsMessage message={parsedMessage} className="p-3" icon="warn" />;
        } else if (data) {
            const results = processData(data);
            if (!results || results.length === 0) {
                content = (
                    <NoResultsMessage
                        message={`No ${pluralize(
                            selectedEntityType.toLowerCase()
                        )} with vulnerabilities found`}
                        className="p-3"
                        icon="info"
                    />
                );
            } else {
                content = (
                    <Scatterplot
                        data={results}
                        lowerX={0}
                        xMultiple={20}
                        yMultiple={5}
                        yAxisTitle="Weighted CVSS Score"
                        xAxisTitle="Critical Vulnerabilities & Exposures"
                        legendData={!small ? legendData : []}
                    />
                );
            }
        }
    }

    return (
        <Widget
            className="h-full pdf-page pdf-stretch"
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
    cveFilter: PropTypes.string,
    small: PropTypes.bool,
};

TopRiskyEntitiesByVulnerabilities.defaultProps = {
    entityContext: {},
    riskEntityTypes: [
        entityTypes.DEPLOYMENT,
        entityTypes.NAMESPACE,
        entityTypes.IMAGE,
        entityTypes.NODE,
    ],
    cveFilter: 'All',
    small: false,
};

export default TopRiskyEntitiesByVulnerabilities;
