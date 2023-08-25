import React, { useContext } from 'react';
import PropTypes from 'prop-types';
import entityTypes from 'constants/entityTypes';
import { gql, useQuery } from '@apollo/client';
import queryService from 'utils/queryService';

import Loader from 'Components/Loader';
import Widget from 'Components/Widget';
import Sunburst from 'Components/visuals/Sunburst';
import NoComponentVulnMessage from 'Components/NoComponentVulnMessage';
import workflowStateContext from 'Containers/workflowStateContext';
import { vulnerabilitySeverityColorMap } from 'constants/severityColors';
import { vulnerabilitySeverityLabels } from 'messages/common';
import { getScopeQuery } from 'Containers/VulnMgmt/Entity/VulnMgmtPolicyQueryUtil';

import ViewAllButton from './ViewAllButton';

const IMAGE_CVES_QUERY = gql`
    query getImageCvesByCVSS($query: String, $scopeQuery: String) {
        results: imageVulnerabilities(query: $query, scopeQuery: $scopeQuery) {
            cve
            cvss
            severity
            summary
        }
    }
`;

const NODE_CVES_QUERY = gql`
    query getNodeCvesByCVSS($query: String, $scopeQuery: String) {
        results: nodeVulnerabilities(query: $query, scopeQuery: $scopeQuery) {
            cve
            cvss
            severity
            summary
        }
    }
`;

const CLUSTER_CVES_QUERY = gql`
    query getClusterCvesByCVSS($query: String, $scopeQuery: String) {
        results: clusterVulnerabilities(query: $query, scopeQuery: $scopeQuery) {
            cve
            cvss
            severity
            summary
        }
    }
`;

const vulnerabilitySeverities = [
    'LOW_VULNERABILITY_SEVERITY',
    'MODERATE_VULNERABILITY_SEVERITY',
    'IMPORTANT_VULNERABILITY_SEVERITY',
    'CRITICAL_VULNERABILITY_SEVERITY',
];

const CvesByCvssScore = ({ entityContext, parentContext }) => {
    let queryToUse = IMAGE_CVES_QUERY;
    let linkTypeToUse = entityTypes.IMAGE_CVE;

    if (entityContext[entityTypes.CLUSTER]) {
        queryToUse = CLUSTER_CVES_QUERY;
        linkTypeToUse = entityTypes.CLUSTER_CVE;
    } else if (
        entityContext[entityTypes.NODE] ||
        entityContext[entityTypes.NODE_COMPONENT] ||
        parentContext[entityTypes.NODE] ||
        parentContext[entityTypes.NODE_COMPONENT]
    ) {
        queryToUse = NODE_CVES_QUERY;
        linkTypeToUse = entityTypes.NODE_CVE;
    }

    const { loading, data = {} } = useQuery(queryToUse, {
        variables: {
            query: queryService.entityContextToQueryString(entityContext),
            scopeQuery: getScopeQuery(parentContext),
        },
    });

    let content = <Loader />;
    let header;

    const workflowState = useContext(workflowStateContext);
    const viewAllURL = workflowState
        .pushList(linkTypeToUse)
        .setSort([{ id: 'cvss', desc: true }])
        .toUrl();

    function getChildren(vulns, severity) {
        return vulns
            .filter((vuln) => vuln.severity === severity)
            .map(({ cve, cvss, summary }) => {
                return {
                    // severity, // generic Sunburst does not expect this data-specific property
                    name: `${cve} -- ${summary}`,
                    color: vulnerabilitySeverityColorMap[severity],
                    labelColor: 'var(--base-600)',
                    textColor: 'var(--base-600)',
                    value: cvss,
                    link: workflowState.pushRelatedEntity(linkTypeToUse, cve).toUrl(),
                };
            });
    }

    function getSunburstData(vulns) {
        return vulnerabilitySeverities.map((severity) => {
            return {
                name: vulnerabilitySeverityLabels[severity],
                color: vulnerabilitySeverityColorMap[severity],
                children: getChildren(vulns, severity),
                textColor: 'var(--base-600)',
                value: 0,
            };
        });
    }

    function getSidePanelData(vulns) {
        return vulnerabilitySeverities.map((severity) => {
            const category = vulns.filter((vuln) => vuln.severity === severity);
            const text = `${category.length} rated as ${vulnerabilitySeverityLabels[severity]}`;
            return {
                text,
                textColor: 'var(--base-600)',
            };
        });
    }
    if (!loading) {
        if (!data || !data.results) {
            content = (
                <div className="flex mx-auto items-center">No scanner setup for this registry.</div>
            );
        } else if (!data.results.length) {
            content = <NoComponentVulnMessage />;
        } else {
            const sunburstData = getSunburstData(data.results);
            const sidePanelData = getSidePanelData(data.results).reverse();
            header = <ViewAllButton url={viewAllURL} />;
            content = (
                <Sunburst
                    data={sunburstData}
                    rootData={sidePanelData}
                    totalValue={data.results.length}
                    units="value"
                    small
                />
            );
        }
    }

    return (
        <Widget className="h-full pdf-page" header="CVEs by CVSS score" headerComponents={header}>
            {content}
        </Widget>
    );
};

CvesByCvssScore.propTypes = {
    entityContext: PropTypes.shape({}),
    parentContext: PropTypes.shape({}),
};

CvesByCvssScore.defaultProps = {
    entityContext: {},
    parentContext: {},
};

export default CvesByCvssScore;
