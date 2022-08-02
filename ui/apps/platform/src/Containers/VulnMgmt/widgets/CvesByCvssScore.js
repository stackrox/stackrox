import React, { useContext } from 'react';
import PropTypes from 'prop-types';
import entityTypes from 'constants/entityTypes';
import { gql, useQuery } from '@apollo/client';
import queryService from 'utils/queryService';

import Loader from 'Components/Loader';
import Widget from 'Components/Widget';
import ViewAllButton from 'Components/ViewAllButton';
import Sunburst from 'Components/visuals/Sunburst';
import NoComponentVulnMessage from 'Components/NoComponentVulnMessage';
import workflowStateContext from 'Containers/workflowStateContext';
import {
    cvssSeverityColorMap,
    cvssSeverityTextColorMap,
    cvssSeverityColorLegend,
} from 'constants/severityColors';
import { getScopeQuery } from 'Containers/VulnMgmt/Entity/VulnMgmtPolicyQueryUtil';
import useFeatureFlags from 'hooks/useFeatureFlags';

const CVES_QUERY = gql`
    query getCvesByCVSS($query: String, $scopeQuery: String) {
        results: vulnerabilities(query: $query, scopeQuery: $scopeQuery) {
            cve
            cvss
            severity
            summary
        }
    }
`;

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

const vulnerabilitySeveritySuffix = '_VULNERABILITY_SEVERITY';

const CvesByCvssScore = ({ entityContext, parentContext }) => {
    const { isFeatureFlagEnabled } = useFeatureFlags();
    const showVMUpdates = isFeatureFlagEnabled('ROX_FRONTEND_VM_UPDATES');

    let queryToUse = CVES_QUERY;

    if (showVMUpdates) {
        if (entityContext[entityTypes.CLUSTER]) {
            queryToUse = CLUSTER_CVES_QUERY;
        } else if (entityContext[entityTypes.NODE] || entityContext[entityTypes.NODE_COMPONENT]) {
            queryToUse = NODE_CVES_QUERY;
        } else {
            queryToUse = IMAGE_CVES_QUERY;
        }
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
        .pushList(entityTypes.CVE)
        .setSort([{ id: 'cvss', desc: true }])
        .toUrl();

    function getChildren(vulns, severity) {
        return vulns
            .filter(
                (vuln) =>
                    vuln.severity === `${severity.toUpperCase()}${vulnerabilitySeveritySuffix}`
            )
            .map(({ cve, cvss, summary }) => {
                const severityString = `${severity.toUpperCase()}${vulnerabilitySeveritySuffix}`;
                return {
                    severity,
                    name: `${cve} -- ${summary}`,
                    color: cvssSeverityColorMap[severityString],
                    labelColor: cvssSeverityTextColorMap[severityString],
                    textColor: cvssSeverityTextColorMap[severityString],
                    value: cvss,
                    link: workflowState.pushRelatedEntity(entityTypes.CVE, cve).toUrl(),
                };
            });
    }

    function getSunburstData(vulns) {
        return cvssSeverityColorLegend.map(({ title, color, textColor }) => {
            const severity = title.toUpperCase();
            return {
                name: title,
                color,
                children: getChildren(vulns, severity),
                textColor,
                value: 0,
            };
        });
    }

    function getSidePanelData(vulns) {
        return cvssSeverityColorLegend.map(({ title, textColor }) => {
            const severity = `${title.toUpperCase()}${vulnerabilitySeveritySuffix}`;
            const category = vulns.filter((vuln) => vuln.severity === severity);
            const text = `${category.length} rated as ${title}`;
            return {
                text,
                textColor,
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
        <Widget className="h-full pdf-page" header="CVEs by CVSS Score" headerComponents={header}>
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
