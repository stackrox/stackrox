import React, { useContext } from 'react';
import { Link } from 'react-router-dom';

import CollapsibleSection from 'Components/CollapsibleSection';
import RiskScore from 'Components/RiskScore';
import Metadata from 'Components/Metadata';
import entityTypes from 'constants/entityTypes';
import workflowStateContext from 'Containers/workflowStateContext';
import TopRiskyEntitiesByVulnerabilities from 'Containers/VulnMgmt/widgets/TopRiskyEntitiesByVulnerabilities';
import RecentlyDetectedImageVulnerabilities from 'Containers/VulnMgmt/widgets/RecentlyDetectedImageVulnerabilities';
import TopRiskiestEntities from 'Containers/VulnMgmt/widgets/TopRiskiestEntities';
import { entityGridContainerClassName } from '../WorkflowEntityPage';

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
    vulnCount: 0,
    vulnerabilities: [],
};

const VulnMgmtNamespaceOverview = ({ data, entityContext }) => {
    const workflowState = useContext(workflowStateContext);

    // guard against incomplete GraphQL-cached data
    const safeData = {
        ...emptyNamespace,
        ...data,
    };

    const { metadata } = safeData;

    if (!metadata) {
        return null;
    }

    const { clusterName, clusterId, priority, labels, id } = metadata;
    const metadataKeyValuePairs = [];

    if (!entityContext[entityTypes.CLUSTER]) {
        const clusterLink = workflowState.pushRelatedEntity(entityTypes.CLUSTER, clusterId).toUrl();
        metadataKeyValuePairs.push({
            key: 'Cluster',
            value: <Link to={clusterLink}>{clusterName}</Link>,
        });
    }

    const namespaceStats = [<RiskScore key="risk-score" score={priority} />];

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
                                title="Details and metadata"
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
                            vulnType={entityTypes.IMAGE_CVE}
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
