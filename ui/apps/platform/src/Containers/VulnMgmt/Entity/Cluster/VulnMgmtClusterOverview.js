import React, { useContext } from 'react';
import pluralize from 'pluralize';

import CollapsibleSection from 'Components/CollapsibleSection';
import DateTimeField from 'Components/DateTimeField';
import Metadata from 'Components/Metadata';
import RiskScore from 'Components/RiskScore';
import StatusChip from 'Components/StatusChip';
import BinderTabs from 'Components/BinderTabs';
import Tab from 'Components/Tab';
import workflowStateContext from 'Containers/workflowStateContext';
import { getPolicyTableColumns } from 'Containers/VulnMgmt/List/Policies/VulnMgmtListPolicies';
import entityTypes from 'constants/entityTypes';
import { OVERVIEW_LIMIT } from 'constants/workflowPages.constants';
import { entityGridContainerClassName } from 'Containers/Workflow/WorkflowEntityPage';

import TopRiskyEntitiesByVulnerabilities from '../../widgets/TopRiskyEntitiesByVulnerabilities';
import RecentlyDetectedImageVulnerabilities from '../../widgets/RecentlyDetectedImageVulnerabilities';
import TopRiskiestEntities from '../../widgets/TopRiskiestEntities';
import DeploymentsWithMostSeverePolicyViolations from '../../widgets/DeploymentsWithMostSeverePolicyViolations';
import RelatedEntitiesSideList from '../RelatedEntitiesSideList';
import TableWidgetFixableCves from '../TableWidgetFixableCves';
import TableWidget from '../TableWidget';

const emptyCluster = {
    componentCount: 0,
    deploymentCount: 0,
    id: '',
    imageCount: 0,
    name: '',
    policyCount: 0,
    policyStatus: {
        status: '',
        failingPolicies: [],
    },
    priority: 0,
    status: {
        orchestratorMetadata: {
            buildDate: '',
            version: 'N/A',
        },
    },
    vulnCount: 0,
};

const VulnMgmtClusterOverview = ({ data, entityContext }) => {
    const workflowState = useContext(workflowStateContext);

    // guard against incomplete GraphQL-cached data
    const safeData = { ...emptyCluster, ...data };

    const { priority, policyStatus, status, istioEnabled, id } = safeData;

    if (!status || !policyStatus) {
        return null;
    }

    const { orchestratorMetadata = null } = status;
    const { buildDate = '', version = 'N/A' } = orchestratorMetadata;
    const { failingPolicies } = policyStatus;

    function yesNoMaybe(value) {
        if (!value && value !== false) {
            return '—';
        }
        return value ? 'Yes' : 'No';
    }

    const metadataKeyValuePairs = [
        {
            key: 'Build date',
            value: <DateTimeField date={buildDate} asString />,
        },
        {
            key: 'K8s version',
            value: version,
        },
        {
            key: 'Istio enabled',
            value: yesNoMaybe(istioEnabled),
        },
    ];

    const clusterStats = [
        <RiskScore key="risk-score" score={priority} />,
        <React.Fragment key="policy-status">
            <span className="pb-2">Policy status:</span>
            <StatusChip status={policyStatus.status} />
        </React.Fragment>,
    ];

    const currentEntity = { [entityTypes.CLUSTER]: id };
    const newEntityContext = { ...entityContext, ...currentEntity };

    return (
        <div className="flex h-full">
            <div className="flex flex-col flex-grow min-w-0">
                <CollapsibleSection title="Cluster Summary">
                    <div className={entityGridContainerClassName}>
                        <div className="s-1">
                            <Metadata
                                className="h-full min-w-48 pdf-page"
                                keyValuePairs={metadataKeyValuePairs}
                                statTiles={clusterStats}
                                title="Details & Metadata"
                            />
                        </div>
                        <div className="sx-1 md:sx-2 sy-1 min-h-55 h-full">
                            <TopRiskyEntitiesByVulnerabilities
                                defaultSelection={entityTypes.NAMESPACE}
                                limit={OVERVIEW_LIMIT}
                                riskEntityTypes={[
                                    entityTypes.NAMESPACE,
                                    entityTypes.DEPLOYMENT,
                                    entityTypes.IMAGE,
                                ]}
                                entityContext={currentEntity}
                                small
                            />
                        </div>
                        <div className="s-1">
                            <RecentlyDetectedImageVulnerabilities
                                limit={OVERVIEW_LIMIT}
                                entityContext={currentEntity}
                            />
                        </div>
                        <div className="s-1">
                            <TopRiskiestEntities
                                limit={OVERVIEW_LIMIT}
                                entityContext={currentEntity}
                            />
                        </div>
                        <div className="s-1">
                            <DeploymentsWithMostSeverePolicyViolations
                                limit={OVERVIEW_LIMIT}
                                entityContext={currentEntity}
                            />
                        </div>
                    </div>
                </CollapsibleSection>

                <CollapsibleSection title="Cluster Findings">
                    <div className="pdf-page pdf-stretch pdf-new flex relative rounded mb-4 ml-4 mr-4">
                        <BinderTabs>
                            <Tab title="Policies">
                                <TableWidget
                                    header={`${failingPolicies.length} failing ${pluralize(
                                        entityTypes.POLICY,
                                        failingPolicies.length
                                    )} across this cluster`}
                                    rows={failingPolicies}
                                    entityType={entityTypes.POLICY}
                                    noDataText="No failing policies"
                                    className="bg-base-100"
                                    columns={getPolicyTableColumns(workflowState)}
                                    idAttribute="id"
                                />
                            </Tab>
                            <Tab title="Fixable CVEs">
                                <TableWidgetFixableCves
                                    workflowState={workflowState}
                                    entityContext={entityContext}
                                    entityType={entityTypes.CLUSTER}
                                    name={safeData?.name}
                                    id={safeData?.id}
                                />
                            </Tab>
                        </BinderTabs>
                    </div>
                </CollapsibleSection>
            </div>
            <RelatedEntitiesSideList
                entityType={entityTypes.CLUSTER}
                entityContext={newEntityContext}
                data={safeData}
            />
        </div>
    );
};

export default VulnMgmtClusterOverview;
