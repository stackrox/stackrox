import React, { useContext } from 'react';
import PropTypes from 'prop-types';
import { useQuery } from 'react-apollo';
import gql from 'graphql-tag';
import { HelpCircle, AlertCircle } from 'react-feather';
import sortBy from 'lodash/sortBy';

import queryService from 'modules/queryService';
import entityTypes from 'constants/entityTypes';
import workflowStateContext from 'Containers/workflowStateContext';
import ViewAllButton from 'Components/ViewAllButton';
import Loader from 'Components/Loader';
import Widget from 'Components/Widget';
import NumberedGrid from 'Components/NumberedGrid';
import FixableCVECount from 'Components/FixableCVECount';
import kubeSVG from 'images/kube.svg';
import istioSVG from 'images/istio.svg';
import Tooltip from 'Components/Tooltip';
import TooltipOverlay from 'Components/TooltipOverlay';

// need to add query for fixable cves for dashboard once it's supported
const CLUSTER_WITH_MOST_K8S_ISTIO_VULNERABILTIES = gql`
    query clustersWithMostK8sVulnerabilities {
        results: clusters {
            id
            name
            isGKECluster
            k8sVulnCount
            k8sVulns {
                cve
                isFixable
            }
            istioVulnCount
            istioVulns {
                cve
                isFixable
            }
        }
    }
`;

const getVulnDataByType = (workflowState, clusterId, vulnType, vulns) => {
    const cveCount = vulns.length;
    const fixableCount = vulns.filter(vuln => vuln.isFixable).length;
    const targetState = workflowState
        .resetPage(entityTypes.CLUSTER, clusterId)
        .pushList(entityTypes.CVE)
        .setSearch({ 'CVE Type': vulnType });

    const url = targetState.toUrl();
    const fixableUrl = targetState
        .setSearch({
            Fixable: true,
            'CVE Type': vulnType
        })
        .toUrl();

    return {
        cveCount,
        fixableCount,
        url,
        fixableUrl
    };
};

const processData = (data, workflowState, limit) => {
    if (!data.results) return [];
    const results = sortBy(data.results, ['k8sVulnCount'])
        .slice(-limit)
        .map(({ id, name, isGKECluster, k8sVulns, istioVulns }) => {
            const {
                cveCount: k8sCveCount,
                fixableCount: k8sFixableCount,
                url: k8sUrl,
                fixableUrl: k8sFixableUrl
            } = getVulnDataByType(workflowState, id, 'K8S_CVE', k8sVulns);
            const {
                cveCount: istioCveCount,
                fixableCount: istioFixableCount,
                url: istioUrl,
                fixableUrl: istioFixableUrl
            } = getVulnDataByType(workflowState, id, 'ISTIO_CVE', istioVulns);
            const clusterUrl = workflowState.resetPage(entityTypes.CLUSTER, id).toUrl();
            const indicationTooltipText = isGKECluster
                ? 'These CVEs might have been patched by GKE. Please check the GKE release notes or security bulletin to find out more.'
                : 'These CVEs were not patched in the current Kubernetes version of this cluster.';

            const indicatorIcon = isGKECluster ? (
                <HelpCircle className="w-4 h-4 text-warning-700" />
            ) : (
                <AlertCircle className="w-4 h-4 text-alert-700" />
            );

            const k8sIstioContent = (
                <div className="flex">
                    <div className="flex flex-1 items-center justify-left mr-8">
                        <img src={kubeSVG} alt="kube" className="pr-2" />
                        <FixableCVECount
                            cves={k8sCveCount}
                            url={k8sUrl}
                            fixableUrl={k8sFixableUrl}
                            fixable={k8sFixableCount}
                            orientation="vertical"
                            showZero
                        />
                        <Tooltip content={<TooltipOverlay>{indicationTooltipText}</TooltipOverlay>}>
                            {/* https://github.com/feathericons/react-feather/issues/56 */}
                            <div className="ml-2">{indicatorIcon}</div>
                        </Tooltip>
                    </div>
                    <div className="flex items-center justify-left pr-2">
                        <img src={istioSVG} alt="istio" className="pr-2" />
                        <FixableCVECount
                            cves={istioCveCount}
                            url={istioUrl}
                            fixableUrl={istioFixableUrl}
                            fixable={istioFixableCount}
                            orientation="vertical"
                            showZero
                        />
                    </div>
                </div>
            );

            return {
                text: name,
                url: clusterUrl,
                component: k8sIstioContent
            };
        });
    return results.slice(0, 8); // @TODO: Remove and add pagination when available
};

const ClustersWithMostK8sVulnerabilities = ({ entityContext, limit }) => {
    const { loading, data = {} } = useQuery(CLUSTER_WITH_MOST_K8S_ISTIO_VULNERABILTIES, {
        variables: {
            query: queryService.entityContextToQueryString(entityContext)
        }
    });

    let content = <Loader />;

    const workflowState = useContext(workflowStateContext);
    if (!loading) {
        const processedData = processData(data, workflowState, limit);

        content = (
            <div className="w-full">
                <NumberedGrid data={processedData} />
            </div>
        );
    }

    const viewAllURL = workflowState
        .pushList(entityTypes.CLUSTER)
        // @TODO: re-enable sorting again, after this fields is available for sorting in back-end pagination
        // .setSort([{ id: 'vulnCounter.all.total', desc: true }])
        .toUrl();

    return (
        <Widget
            className="h-full pdf-page"
            header="Clusters With Most K8s & Istio Vulnerabilities"
            headerComponents={<ViewAllButton url={viewAllURL} />}
        >
            {content}
        </Widget>
    );
};

ClustersWithMostK8sVulnerabilities.propTypes = {
    entityContext: PropTypes.shape({}),
    limit: PropTypes.number
};

ClustersWithMostK8sVulnerabilities.defaultProps = {
    entityContext: {},
    limit: 8
};

export default ClustersWithMostK8sVulnerabilities;
