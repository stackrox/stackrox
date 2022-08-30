import React, { useContext } from 'react';
import { Link } from 'react-router-dom';
import pluralize from 'pluralize';

import CollapsibleSection from 'Components/CollapsibleSection';
import Metadata from 'Components/Metadata';
import RiskScore from 'Components/RiskScore';
import StatusChip from 'Components/StatusChip';
import BinderTabs from 'Components/BinderTabs';
import Tab from 'Components/Tab';
import entityTypes from 'constants/entityTypes';
import PolicyViolationsBySeverity from 'Containers/VulnMgmt/widgets/PolicyViolationsBySeverity';
import CvesByCvssScore from 'Containers/VulnMgmt/widgets/CvesByCvssScore';
import RecentlyDetectedImageVulnerabilities from 'Containers/VulnMgmt/widgets/RecentlyDetectedImageVulnerabilities';
import MostCommonVulnerabiltiesInDeployment from 'Containers/VulnMgmt/widgets/MostCommonVulnerabiltiesInDeployment';
import TopRiskiestEntities from 'Containers/VulnMgmt/widgets/TopRiskiestEntities';
import workflowStateContext from 'Containers/workflowStateContext';
import { getPolicyTableColumns } from 'Containers/VulnMgmt/List/Policies/VulnMgmtListPolicies';
import { entityGridContainerClassName } from 'Containers/Workflow/WorkflowEntityPage';
import ViolationsAcrossThisDeployment from 'Containers/Workflow/widgets/ViolationsAcrossThisDeployment';
import useFeatureFlags from 'hooks/useFeatureFlags';

import RelatedEntitiesSideList from '../RelatedEntitiesSideList';
import TableWidgetFixableCves from '../TableWidgetFixableCves';
import TableWidget from '../TableWidget';

const emptyDeployment = {
    annotations: [],
    cluster: {},
    componentCount: 0,
    created: '',
    failingPolicies: [],
    id: '',
    imageCount: 0,
    inactive: false,
    labels: [],
    name: '',
    namespace: '',
    namespaceId: '',
    policyStatus: '',
    priority: 0,
    vulnCount: 0,
    vulnerabilities: [],
};

const VulnMgmtDeploymentOverview = ({ data, entityContext }) => {
    const { isFeatureFlagEnabled } = useFeatureFlags();
    const showVMUpdates = isFeatureFlagEnabled('ROX_FRONTEND_VM_UPDATES');

    const workflowState = useContext(workflowStateContext);

    // guard against incomplete GraphQL-cached data
    const safeData = { ...emptyDeployment, ...data };

    const {
        id,
        cluster,
        priority,
        namespace,
        namespaceId,
        policyStatus,
        failingPolicies,
        labels,
        annotations,
    } = safeData;

    const metadataKeyValuePairs = [];

    if (!entityContext[entityTypes.CLUSTER]) {
        const clusterLink = workflowState
            .pushRelatedEntity(entityTypes.CLUSTER, cluster.id)
            .toUrl();
        metadataKeyValuePairs.push({
            key: 'Cluster',
            value: cluster && cluster.name && <Link to={clusterLink}>{cluster.name}</Link>,
        });
    }
    if (!entityContext[entityTypes.NAMESPACE]) {
        const namespaceLink = workflowState
            .pushRelatedEntity(entityTypes.NAMESPACE, namespaceId)
            .toUrl();
        metadataKeyValuePairs.push({
            key: 'Namespace',
            value: <Link to={namespaceLink}>{namespace}</Link>,
        });
    }

    const deploymentStats = [
        <RiskScore key="risk-score" score={priority} />,
        <React.Fragment key="policy-status">
            <span className="pb-2">Policy status:</span>
            <StatusChip status={policyStatus} />
        </React.Fragment>,
    ];
    const currentEntity = { [entityTypes.DEPLOYMENT]: id };
    const newEntityContext = { ...entityContext, ...currentEntity };

    // need to know if this deployment is scoped to a policy higher up in the hierarchy
    const policyAncestor = workflowState.getSingleAncestorOfType(entityTypes.POLICY);

    let deploymentFindingsContent = null;
    if (policyAncestor) {
        deploymentFindingsContent = (
            <ViolationsAcrossThisDeployment
                deploymentID={id}
                policyID={entityContext[entityTypes.POLICY] || policyAncestor.entityId}
                message="This deployment has not failed on this policy"
            />
        );
    } else {
        deploymentFindingsContent = (
            <div className="flex pdf-page pdf-stretch pdf-new relative rounded mb-4 ml-4 mr-4">
                <BinderTabs>
                    <Tab title="Policies">
                        <TableWidget
                            header={`${failingPolicies.length} failing ${pluralize(
                                entityTypes.POLICY,
                                failingPolicies.length
                            )} across this deployment`}
                            rows={failingPolicies}
                            entityType={entityTypes.POLICY}
                            noDataText="No failing policies"
                            className="bg-base-100"
                            columns={getPolicyTableColumns(workflowState)}
                        />
                    </Tab>
                    <Tab title="Fixable CVEs">
                        <TableWidgetFixableCves
                            workflowState={workflowState}
                            entityContext={entityContext}
                            entityType={entityTypes.DEPLOYMENT}
                            name={safeData?.name}
                            id={safeData?.id}
                            vulnType={showVMUpdates ? entityTypes.IMAGE_CVE : entityTypes.CVE}
                        />
                    </Tab>
                </BinderTabs>
            </div>
        );
    }

    return (
        <div className="flex h-full">
            <div className="flex flex-col flex-grow min-w-0">
                <CollapsibleSection title="Deployment Summary">
                    <div className={entityGridContainerClassName}>
                        <div className="s-1">
                            <Metadata
                                className="h-full min-w-48 bg-base-100 pdf-page"
                                keyValuePairs={metadataKeyValuePairs}
                                statTiles={deploymentStats}
                                title="Details & Metadata"
                                labels={labels}
                                annotations={annotations}
                            />
                        </div>
                        <div className="s-1">
                            <PolicyViolationsBySeverity entityContext={currentEntity} />
                        </div>
                        <div className="s-1">
                            <CvesByCvssScore entityContext={currentEntity} />
                        </div>
                        <div className="s-1">
                            <RecentlyDetectedImageVulnerabilities entityContext={currentEntity} />
                        </div>
                        <div className="s-1">
                            <MostCommonVulnerabiltiesInDeployment deploymentId={id} />
                        </div>
                        <div className="s-1">
                            <TopRiskiestEntities limit={5} entityContext={currentEntity} />
                        </div>
                    </div>
                </CollapsibleSection>

                <CollapsibleSection title="Deployment Findings">
                    {deploymentFindingsContent}
                </CollapsibleSection>
            </div>
            <RelatedEntitiesSideList
                entityType={entityTypes.DEPLOYMENT}
                entityContext={newEntityContext}
                data={safeData}
            />
        </div>
    );
};

export default VulnMgmtDeploymentOverview;
