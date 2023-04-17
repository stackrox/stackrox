import React, { useContext } from 'react';
import { Link } from 'react-router-dom';

import CollapsibleSection from 'Components/CollapsibleSection';
import PolicyStatusIconText from 'Components/PatternFly/IconText/PolicyStatusIconText';
import RiskScore from 'Components/RiskScore';
import Metadata from 'Components/Metadata';
import entityTypes from 'constants/entityTypes';
import workflowStateContext from 'Containers/workflowStateContext';
import TopRiskyEntitiesByVulnerabilities from 'Containers/VulnMgmt/widgets/TopRiskyEntitiesByVulnerabilities';
import RecentlyDetectedImageVulnerabilities from 'Containers/VulnMgmt/widgets/RecentlyDetectedImageVulnerabilities';
import TopRiskiestEntities from 'Containers/VulnMgmt/widgets/TopRiskiestEntities';
import { entityGridContainerClassName } from 'Containers/Workflow/WorkflowEntityPage';
import useFeatureFlags from 'hooks/useFeatureFlags';

import RelatedEntitiesSideList from '../RelatedEntitiesSideList';
import TableWidgetFixableCves from '../TableWidgetFixableCves';

const emptyNamespace = {
    deploymentCount: 0,
    componentCount: 0,
    metadata: {
        clusterName: '',
        clusterId: '',
        name: '',
        priority: 0,
        labels: [],
        id: '',
    },
    policyStatus: {
        status: '',
        failingPolicies: [],
    },
    vulnCount: 0,
    vulnerabilities: [],
};

const VulnMgmtNamespaceOverview = ({ data, entityContext }) => {
    const { isFeatureFlagEnabled } = useFeatureFlags();
    const showVMUpdates = isFeatureFlagEnabled('ROX_POSTGRES_DATASTORE');

    const workflowState = useContext(workflowStateContext);

    // guard against incomplete GraphQL-cached data
    const safeData = {
        ...emptyNamespace,
        ...data,
    };

    const { metadata, policyStatus } = safeData;

    if (!metadata || !policyStatus) {
        return null;
    }

    const { clusterName, clusterId, priority, labels, id } = metadata;
    const { status } = policyStatus;
    const metadataKeyValuePairs = [];

    if (!entityContext[entityTypes.CLUSTER]) {
        const clusterLink = workflowState.pushRelatedEntity(entityTypes.CLUSTER, clusterId).toUrl();
        metadataKeyValuePairs.push({
            key: 'Cluster',
            value: <Link to={clusterLink}>{clusterName}</Link>,
        });
    }

    const namespaceStats = [
        <RiskScore key="risk-score" score={priority} />,
        <React.Fragment key="policy-status">
            <span className="pb-2">Policy status:</span>
            <PolicyStatusIconText isPass={status === 'pass'} isTextOnly={false} />
        </React.Fragment>,
    ];

    const currentEntity = { [entityTypes.NAMESPACE]: id };
    const newEntityContext = { ...entityContext, ...currentEntity };

    return (
        <div className="flex h-full">
            <div className="flex flex-col flex-grow min-w-0">
                <CollapsibleSection title="Namespace Summary">
                    <div className={entityGridContainerClassName}>
                        <div className="s-1">
                            <Metadata
                                className="h-full min-w-48 bg-base-100"
                                keyValuePairs={metadataKeyValuePairs}
                                statTiles={namespaceStats}
                                labels={labels}
                                title="Details & Metadata"
                            />
                        </div>
                        <div className="sx-1 lg:sx-2 sy-1 min-h-55 h-full">
                            <TopRiskyEntitiesByVulnerabilities
                                defaultSelection={entityTypes.DEPLOYMENT}
                                riskEntityTypes={[
                                    entityTypes.DEPLOYMENT,
                                    entityTypes.IMAGE,
                                    entityTypes.NODE,
                                ]}
                                entityContext={currentEntity}
                                small
                            />
                        </div>
                        <div className="s-1">
                            <RecentlyDetectedImageVulnerabilities entityContext={currentEntity} />
                        </div>
                        <div className="s-1">
                            <TopRiskiestEntities entityContext={currentEntity} />
                        </div>
                    </div>
                </CollapsibleSection>
                <CollapsibleSection title="Namespace findings">
                    <div className="flex pdf-page pdf-stretch pdf-new relative rounded mb-4 ml-4 mr-4">
                        <TableWidgetFixableCves
                            workflowState={workflowState}
                            entityContext={entityContext}
                            entityType={entityTypes.NAMESPACE}
                            name={safeData?.metadata?.name}
                            id={safeData?.metadata?.id}
                            vulnType={showVMUpdates ? entityTypes.IMAGE_CVE : entityTypes.CVE}
                        />
                    </div>
                </CollapsibleSection>
            </div>

            <RelatedEntitiesSideList
                entityType={entityTypes.NAMESPACE}
                entityContext={newEntityContext}
                data={safeData}
            />
        </div>
    );
};

export default VulnMgmtNamespaceOverview;
