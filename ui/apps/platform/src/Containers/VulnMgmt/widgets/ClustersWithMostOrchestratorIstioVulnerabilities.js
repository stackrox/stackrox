import React, { useContext } from 'react';
import PropTypes from 'prop-types';
import { gql, useQuery } from '@apollo/client';
import { HelpCircle, AlertCircle } from 'react-feather';
import sortBy from 'lodash/sortBy';

import { checkForPermissionErrorMessage } from 'utils/permissionUtils';
import queryService from 'utils/queryService';
import entityTypes from 'constants/entityTypes';
import workflowStateContext from 'Containers/workflowStateContext';
import ViewAllButton from 'Components/ViewAllButton';
import Loader from 'Components/Loader';
import Widget from 'Components/Widget';
import NoResultsMessage from 'Components/NoResultsMessage';
import NumberedGrid from 'Components/NumberedGrid';
import FixableCVECount from 'Components/FixableCVECount';
import kubeSVG from 'images/kube.svg';
import istioSVG from 'images/istio.svg';
import openShiftSVG from 'images/openShift.svg';
import { Tooltip, TooltipOverlay } from '@stackrox/ui-components';
import useFeatureFlags from 'hooks/useFeatureFlags';

// need to add query for fixable cves for dashboard once it's supported
const CLUSTER_WITH_MOST_ORCHESTRATOR_ISTIO_VULNERABILTIES = gql`
    query clustersWithMostOrchestratorIstioVulnerabilities {
        results: clusters {
            id
            name
            isGKECluster
            isOpenShiftCluster
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
            openShiftVulnCount
            openShiftVulns {
                cve
                isFixable
            }
        }
    }
`;

const getVulnDataByType = (workflowState, clusterId, vulnType, vulns, showVmUpdates) => {
    const cveCount = vulns.length;
    const fixableCount = vulns.filter((vuln) => vuln.isFixable).length;
    const targetState = workflowState
        .resetPage(entityTypes.CLUSTER, clusterId)
        .pushList(showVmUpdates ? entityTypes.CLUSTER_CVE : entityTypes.CVE)
        .setSearch({ 'CVE Type': vulnType });

    const url = targetState.toUrl();
    const fixableUrl = targetState
        .setSearch({
            Fixable: true,
            'CVE Type': vulnType,
        })
        .toUrl();

    return {
        cveCount,
        fixableCount,
        url,
        fixableUrl,
    };
};

const processData = (data, workflowState, limit, showVmUpdates) => {
    if (!data.results) {
        return [];
    }
    const results = sortBy(data.results, ['k8sVulnCount'])
        .slice(-limit)
        .map(
            ({
                id,
                name,
                isGKECluster,
                isOpenShiftCluster,
                k8sVulns,
                istioVulns,
                openShiftVulns,
            }) => {
                const {
                    cveCount: k8sCveCount,
                    fixableCount: k8sFixableCount,
                    url: k8sUrl,
                    fixableUrl: k8sFixableUrl,
                } = getVulnDataByType(workflowState, id, 'K8S_CVE', k8sVulns, showVmUpdates);
                const {
                    cveCount: istioCveCount,
                    fixableCount: istioFixableCount,
                    url: istioUrl,
                    fixableUrl: istioFixableUrl,
                } = getVulnDataByType(workflowState, id, 'ISTIO_CVE', istioVulns, showVmUpdates);
                const {
                    cveCount: openShiftCveCount,
                    fixableCount: openShiftFixableCount,
                    url: openShiftUrl,
                    fixableUrl: openShiftFixableUrl,
                } = getVulnDataByType(
                    workflowState,
                    id,
                    'OPENSHIFT_CVE',
                    openShiftVulns,
                    showVmUpdates
                );
                const clusterUrl = workflowState.resetPage(entityTypes.CLUSTER, id).toUrl();
                const indicationTooltipText = isGKECluster
                    ? 'These CVEs might have been patched by GKE. Please check the GKE release notes or security bulletin to find out more.'
                    : 'These CVEs were not patched in the current Kubernetes version of this cluster.';

                const indicatorIcon = isGKECluster ? (
                    <HelpCircle className="w-4 h-4 text-warning-700 ml-2" />
                ) : (
                    <AlertCircle className="w-4 h-4 text-alert-700 ml-2" />
                );

                const orchestratorContent = isOpenShiftCluster ? (
                    <div className="flex items-center justify-left mr-8">
                        <Tooltip
                            content={<TooltipOverlay>OpenShift Vulnerabilities</TooltipOverlay>}
                        >
                            <div className="flex">
                                <img src={openShiftSVG} alt="openshift" className="pr-2" />
                                <FixableCVECount
                                    cves={openShiftCveCount}
                                    url={openShiftUrl}
                                    fixableUrl={openShiftFixableUrl}
                                    fixable={openShiftFixableCount}
                                    orientation="vertical"
                                    showZero
                                />
                            </div>
                        </Tooltip>
                    </div>
                ) : (
                    <div className="flex flex-1 items-center justify-left mr-8">
                        <Tooltip
                            content={<TooltipOverlay>Kubernetes Vulnerabilities</TooltipOverlay>}
                        >
                            <div className="flex">
                                <img src={kubeSVG} alt="kube" className="pr-2" />
                                <FixableCVECount
                                    cves={k8sCveCount}
                                    url={k8sUrl}
                                    fixableUrl={k8sFixableUrl}
                                    fixable={k8sFixableCount}
                                    orientation="vertical"
                                    showZero
                                />
                            </div>
                        </Tooltip>
                        <Tooltip content={<TooltipOverlay>{indicationTooltipText}</TooltipOverlay>}>
                            {indicatorIcon}
                        </Tooltip>
                    </div>
                );

                const orchestratorIstioContent = (
                    <div className="flex">
                        {orchestratorContent}
                        <Tooltip content={<TooltipOverlay>Istio Vulnerabilities</TooltipOverlay>}>
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
                        </Tooltip>
                    </div>
                );

                return {
                    text: name,
                    url: clusterUrl,
                    component: orchestratorIstioContent,
                };
            }
        );
    return results.slice(0, 8); // @TODO: Remove and add pagination when available
};

const ClustersWithMostOrchestratorVulnerabilities = ({ entityContext, limit }) => {
    const {
        loading,
        data = {},
        error,
    } = useQuery(CLUSTER_WITH_MOST_ORCHESTRATOR_ISTIO_VULNERABILTIES, {
        variables: {
            query: queryService.entityContextToQueryString(entityContext),
        },
    });
    const { isFeatureFlagEnabled } = useFeatureFlags();
    const showVmUpdates = isFeatureFlagEnabled('ROX_FRONTEND_VM_UPDATES');

    let content = <Loader />;

    const workflowState = useContext(workflowStateContext);
    if (!loading) {
        if (error) {
            const defaultMessage = `An error occurred in retrieving vulnerabilities or clusters. Please refresh the page. If this problem continues, please contact support.`;

            const parsedMessage = checkForPermissionErrorMessage(error, defaultMessage);

            content = <NoResultsMessage message={parsedMessage} className="p-3" icon="warn" />;
        } else {
            const processedData = processData(data, workflowState, limit, showVmUpdates);

            if (!processedData || processedData.length === 0) {
                content = (
                    <NoResultsMessage
                        message="No vulnerabilities found"
                        className="p-3"
                        icon="info"
                    />
                );
            } else {
                content = (
                    <div className="w-full">
                        <NumberedGrid data={processedData} />
                    </div>
                );
            }
        }
    }

    const viewAllURL = workflowState
        .pushList(entityTypes.CLUSTER)
        // @TODO: re-enable sorting again, after this fields is available for sorting in back-end pagination
        // .setSort([{ id: 'vulnCounter.all.total', desc: true }])
        .toUrl();

    return (
        <Widget
            className="h-full pdf-page"
            header="Clusters With Most Orchestrator & Istio Vulnerabilities"
            headerComponents={<ViewAllButton url={viewAllURL} />}
        >
            {content}
        </Widget>
    );
};

ClustersWithMostOrchestratorVulnerabilities.propTypes = {
    entityContext: PropTypes.shape({}),
    limit: PropTypes.number,
};

ClustersWithMostOrchestratorVulnerabilities.defaultProps = {
    entityContext: {},
    limit: 8,
};

export default ClustersWithMostOrchestratorVulnerabilities;
