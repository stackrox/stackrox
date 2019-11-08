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
import { getCveTableColumns } from 'Containers/VulnMgmt/List/Cves/VulnMgmtListCves';
import { entityGridContainerClassName } from 'Containers/Workflow/WorkflowEntityPage';
// TO DO: factor this out from config management
import ViolationsAcrossThisDeployment from 'Containers/ConfigManagement/Entity/widgets/ViolationsAcrossThisDeployment';

import RelatedEntitiesSideList from '../RelatedEntitiesSideList';
import TableWidget from '../TableWidget';

const VulnMgmtDeploymentOverview = ({ data, entityContext }) => {
    const workflowState = useContext(workflowStateContext);

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
        failingPolicyCount,
        imageCount,
        componentCount,
        vulnCount,
        vulnerabilities
    } = data;

    const metadataKeyValuePairs = [];

    const fixableCves = vulnerabilities.filter(cve => cve.isFixable);

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
            <span className="pr-1">Policy status:</span>
            <StatusChip status={policyStatus} />
        </React.Fragment>
    ];

    function getCountData(entityType) {
        switch (entityType) {
            case entityTypes.COMPONENT:
                return componentCount;
            case entityTypes.CVE:
                return vulnCount;
            case entityTypes.IMAGE:
                return imageCount;
            case entityTypes.POLICY:
                return failingPolicyCount;
            default:
                return 0;
        }
    }

    const newEntityContext = { ...entityContext, [entityTypes.DEPLOYMENT]: id };

    let deploymentFindingsContent = null;
    if (entityContext[entityTypes.POLICY]) {
        deploymentFindingsContent = (
            <ViolationsAcrossThisDeployment
                deploymentID={id}
                policyID={entityContext[entityTypes.POLICY]}
                message="No policies failed across this deployment"
            />
        );
    } else {
        deploymentFindingsContent = (
            <div className="flex pdf-page pdf-stretch shadow rounded relative rounded bg-base-100 mb-4 ml-4 mr-4">
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
                            columns={getPolicyTableColumns(workflowState, false)}
                        />
                    </TabContent>
                    <TabContent>
                        <TableWidget
                            header={`${fixableCves.length} fixable ${pluralize(
                                entityTypes.CVE,
                                fixableCves.length
                            )} found across this deployment`}
                            rows={fixableCves}
                            entityType={entityTypes.CVE}
                            noDataText="No fixable CVEs available in this deployment"
                            className="bg-base-100"
                            columns={getCveTableColumns(workflowState, false)}
                            idAttribute="cve"
                        />
                    </TabContent>
                </Tabs>
            </div>
        );
    }

    return (
        <div className="flex h-full">
            <div className="flex flex-col flex-grow min-w-0">
                <CollapsibleSection title="Deployment summary">
                    <div className={entityGridContainerClassName}>
                        <div className="s-1">
                            <Metadata
                                className="h-full min-w-48 bg-base-100 bg-counts-widget"
                                keyValuePairs={metadataKeyValuePairs}
                                statTiles={deploymentStats}
                                title="Details & Metadata"
                                labels={labels}
                                annotations={annotations}
                            />
                        </div>
                        <div className="s-1">
                            <PolicyViolationsBySeverity entityContext={newEntityContext} />
                        </div>
                        <div className="s-1">
                            <CvesByCvssScore entityContext={newEntityContext} />
                        </div>
                        <div className="s-1">
                            <RecentlyDetectedVulnerabilities entityContext={newEntityContext} />
                        </div>
                        <div className="s-1">
                            <MostCommonVulnerabiltiesInDeployment deploymentId={id} />
                        </div>
                        <div className="s-1">
                            <TopRiskiestImagesAndComponents
                                limit={5}
                                entityContext={newEntityContext}
                            />
                        </div>
                    </div>
                </CollapsibleSection>

                <CollapsibleSection title="Deployment findings">
                    {deploymentFindingsContent}
                </CollapsibleSection>
            </div>
            <RelatedEntitiesSideList
                entityType={entityTypes.DEPLOYMENT}
                workflowState={workflowState}
                getCountData={getCountData}
                entityContext={newEntityContext}
            />
        </div>
    );
};

export default VulnMgmtDeploymentOverview;
