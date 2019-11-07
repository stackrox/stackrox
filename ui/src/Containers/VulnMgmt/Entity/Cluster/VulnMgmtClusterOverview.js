import React, { useContext } from 'react';
import pluralize from 'pluralize';

import CollapsibleSection from 'Components/CollapsibleSection';
import DateTimeField from 'Components/DateTimeField';
import Metadata from 'Components/Metadata';
import RiskScore from 'Components/RiskScore';
import StatusChip from 'Components/StatusChip';
import Tabs from 'Components/Tabs';
import TabContent from 'Components/TabContent';
import workflowStateContext from 'Containers/workflowStateContext';
import { getPolicyTableColumns } from 'Containers/VulnMgmt/List/Policies/VulnMgmtListPolicies';
import { getCveTableColumns } from 'Containers/VulnMgmt/List/Cves/VulnMgmtListCves';
import entityTypes from 'constants/entityTypes';
import { overviewLimit } from 'constants/workflowPages.constants';
import { entityGridContainerClassName } from 'Containers/Workflow/WorkflowEntityPage';

import TopRiskyEntitiesByVulnerabilities from '../../widgets/TopRiskyEntitiesByVulnerabilities';
import RecentlyDetectedVulnerabilities from '../../widgets/RecentlyDetectedVulnerabilities';
import TopRiskiestImagesAndComponents from '../../widgets/TopRiskiestImagesAndComponents';
import DeploymentsWithMostSeverePolicyViolations from '../../widgets/DeploymentsWithMostSeverePolicyViolations';
import RelatedEntitiesSideList from '../RelatedEntitiesSideList';
import TableWidget from '../TableWidget';

const VulnMgmtClusterOverview = ({ data, entityContext }) => {
    const workflowState = useContext(workflowStateContext);

    const {
        priority,
        policyStatus,
        createdAt,
        status: { orchestratorMetadata = null },
        istioEnabled,
        deploymentCount,
        componentCount,
        imageCount,
        namespaceCount,
        policyCount,
        vulnCount,
        vulnerabilities,
        id
    } = data;

    const { version = 'N/A' } = orchestratorMetadata;
    const { failingPolicies } = policyStatus;
    const fixableCves = vulnerabilities.filter(cve => cve.isFixable);

    function yesNoMaybe(value) {
        if (!value && value !== false) {
            return '—';
        }
        return value ? 'Yes' : 'No';
    }

    const metadataKeyValuePairs = [
        {
            key: 'Created',
            value: <DateTimeField date={createdAt} />
        },
        {
            key: 'K8s version',
            value: version
        },
        {
            key: 'Istio Enabled',
            value: yesNoMaybe(istioEnabled)
        }
    ];

    const clusterStats = [
        <RiskScore key="risk-score" score={priority} />,
        <React.Fragment key="policy-status">
            <span className="pb-2">Policy status:</span>
            <StatusChip status={policyStatus.status} />
        </React.Fragment>
    ];

    function getCountData(entityType) {
        switch (entityType) {
            case entityTypes.DEPLOYMENT:
                return deploymentCount;
            case entityTypes.COMPONENT:
                return componentCount;
            case entityTypes.CVE:
                return vulnCount;
            case entityTypes.IMAGE:
                return imageCount;
            case entityTypes.NAMESPACE:
                return namespaceCount;
            case entityTypes.POLICY:
                return policyCount;
            default:
                return 0;
        }
    }

    const newEntityContext = { ...entityContext, [entityTypes.CLUSTER]: id };

    return (
        <div className="flex h-full">
            <div className="flex flex-col flex-grow min-w-0">
                <CollapsibleSection title="Cluster Details">
                    <div className={entityGridContainerClassName}>
                        <div className="s-1">
                            <Metadata
                                className="h-full min-w-48 bg-base-100"
                                keyValuePairs={metadataKeyValuePairs}
                                statTiles={clusterStats}
                                title="Details & Metadata"
                            />
                        </div>
                        <div className="sx-1 lg:sx-2 sy-1 h-55">
                            <TopRiskyEntitiesByVulnerabilities
                                defaultSelection={entityTypes.NAMESPACE}
                                limit={overviewLimit}
                                riskEntityTypes={[
                                    entityTypes.NAMESPACE,
                                    entityTypes.DEPLOYMENT,
                                    entityTypes.IMAGE
                                ]}
                                entityContext={newEntityContext}
                                small
                            />
                        </div>
                        <div className="s-1">
                            <RecentlyDetectedVulnerabilities
                                limit={overviewLimit}
                                entityContext={newEntityContext}
                            />
                        </div>
                        <div className="s-1">
                            <TopRiskiestImagesAndComponents
                                limit={overviewLimit}
                                entityContext={newEntityContext}
                            />
                        </div>
                        <div className="s-1">
                            <DeploymentsWithMostSeverePolicyViolations
                                limit={overviewLimit}
                                entityContext={newEntityContext}
                            />
                        </div>
                    </div>
                </CollapsibleSection>

                <CollapsibleSection title="Cluster findings">
                    <div className="flex pdf-page pdf-stretch shadow rounded relative rounded bg-base-100 mb-4 ml-4 mr-4">
                        <Tabs
                            hasTabSpacing
                            headers={[{ text: 'Policies' }, { text: 'Fixable CVEs' }]}
                        >
                            <TabContent>
                                <TableWidget
                                    header={`${failingPolicies.length} failing ${pluralize(
                                        entityTypes.POLICY,
                                        failingPolicies.length
                                    )} across this cluster`}
                                    rows={failingPolicies}
                                    noDataText="No failing policies"
                                    className="bg-base-100"
                                    columns={getPolicyTableColumns(workflowState, false)}
                                    idAttribute="id"
                                />
                            </TabContent>
                            <TabContent>
                                <TableWidget
                                    header={`${fixableCves.length} fixable ${pluralize(
                                        entityTypes.CVE,
                                        fixableCves.length
                                    )} found across this cluster`}
                                    rows={fixableCves}
                                    entityType={entityTypes.CVE}
                                    noDataText="No fixable CVEs available in this cluster"
                                    className="bg-base-100"
                                    columns={getCveTableColumns(workflowState, false)}
                                    idAttribute="cve"
                                />
                            </TabContent>
                        </Tabs>
                    </div>
                </CollapsibleSection>
            </div>
            <RelatedEntitiesSideList
                entityType={entityTypes.CLUSTER}
                workflowState={workflowState}
                getCountData={getCountData}
                entityContext={newEntityContext}
            />
        </div>
    );
};

export default VulnMgmtClusterOverview;
