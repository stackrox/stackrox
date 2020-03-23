import React, { useContext } from 'react';
import { Link } from 'react-router-dom';
import pluralize from 'pluralize';

import CollapsibleSection from 'Components/CollapsibleSection';
import Metadata from 'Components/Metadata';
import RiskScore from 'Components/RiskScore';
import StatusChip from 'Components/StatusChip';
import Tabs from 'Components/Tabs';
import TabContent from 'Components/TabContent';
import entityTypes from 'constants/entityTypes';
import PolicyViolationsBySeverity from 'Containers/VulnMgmt/widgets/PolicyViolationsBySeverity';
import CvesByCvssScore from 'Containers/VulnMgmt/widgets/CvesByCvssScore';
import RecentlyDetectedVulnerabilities from 'Containers/VulnMgmt/widgets/RecentlyDetectedVulnerabilities';
import MostCommonVulnerabiltiesInDeployment from 'Containers/VulnMgmt/widgets/MostCommonVulnerabiltiesInDeployment';
import TopRiskiestImagesAndComponents from 'Containers/VulnMgmt/widgets/TopRiskiestImagesAndComponents';
import workflowStateContext from 'Containers/workflowStateContext';
import { getPolicyTableColumns } from 'Containers/VulnMgmt/List/Policies/VulnMgmtListPolicies';
import { entityGridContainerClassName } from 'Containers/Workflow/WorkflowEntityPage';
import ViolationsAcrossThisDeployment from 'Containers/Workflow/widgets/ViolationsAcrossThisDeployment';

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
    vulnerabilities: []
};

const VulnMgmtDeploymentOverview = ({ data, entityContext }) => {
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
        annotations
    } = safeData;

    const metadataKeyValuePairs = [];

    if (!entityContext[entityTypes.CLUSTER]) {
        const clusterLink = workflowState
            .pushRelatedEntity(entityTypes.CLUSTER, cluster.id)
            .toUrl();
        metadataKeyValuePairs.push({
            key: 'Cluster',
            value: cluster && cluster.name && <Link to={clusterLink}>{cluster.name}</Link>
        });
    }
    if (!entityContext[entityTypes.NAMESPACE]) {
        const namespaceLink = workflowState
            .pushRelatedEntity(entityTypes.NAMESPACE, namespaceId)
            .toUrl();
        metadataKeyValuePairs.push({
            key: 'Namespace',
            value: <Link to={namespaceLink}>{namespace}</Link>
        });
    }

    const deploymentStats = [
        <RiskScore key="risk-score" score={priority} />,
        <React.Fragment key="policy-status">
            <span className="pb-2">Policy status:</span>
            <StatusChip status={policyStatus} />
        </React.Fragment>
    ];
    const currentEntity = { [entityTypes.DEPLOYMENT]: id };
    const newEntityContext = { ...entityContext, ...currentEntity };

    // the parent entity is needed for cases where the deployment single is a child
    //   from a click on a Failing Policies table row
    const parentEntity = workflowState.pop();

    let deploymentFindingsContent = null;
    if (
        entityContext[entityTypes.POLICY] ||
        parentEntity.getCurrentEntityType() === entityTypes.POLICY
    ) {
        deploymentFindingsContent = (
            <ViolationsAcrossThisDeployment
                deploymentID={id}
                policyID={entityContext[entityTypes.POLICY] || parentEntity.getCurrentEntityId()}
                message="This deployment has not failed on this policy"
            />
        );
    } else {
        deploymentFindingsContent = (
            <div className="flex pdf-page pdf-stretch pdf-new shadow rounded relative rounded bg-base-100 mb-4 ml-4 mr-4">
                <Tabs hasTabSpacing headers={[{ text: 'Policies' }, { text: 'Fixable CVEs' }]}>
                    <TabContent>
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
                    </TabContent>
                    <TabContent>
                        <TableWidgetFixableCves
                            workflowState={workflowState}
                            entityContext={entityContext}
                            entityType={entityTypes.DEPLOYMENT}
                            name={safeData?.name}
                            id={safeData?.id}
                        />
                    </TabContent>
                </Tabs>
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
                                bgClass
                            />
                        </div>
                        <div className="s-1">
                            <PolicyViolationsBySeverity entityContext={currentEntity} />
                        </div>
                        <div className="s-1">
                            <CvesByCvssScore entityContext={currentEntity} />
                        </div>
                        <div className="s-1">
                            <RecentlyDetectedVulnerabilities entityContext={currentEntity} />
                        </div>
                        <div className="s-1">
                            <MostCommonVulnerabiltiesInDeployment deploymentId={id} />
                        </div>
                        <div className="s-1">
                            <TopRiskiestImagesAndComponents
                                limit={5}
                                entityContext={currentEntity}
                            />
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
