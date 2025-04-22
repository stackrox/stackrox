import React, { useContext } from 'react';
import { Link } from 'react-router-dom';

import CollapsibleSection from 'Components/CollapsibleSection';
import Metadata from 'Components/Metadata';
import RiskScore from 'Components/RiskScore';
import entityTypes from 'constants/entityTypes';
import CvesByCvssScore from 'Containers/VulnMgmt/widgets/CvesByCvssScore';
import RecentlyDetectedImageVulnerabilities from 'Containers/VulnMgmt/widgets/RecentlyDetectedImageVulnerabilities';
import MostCommonVulnerabiltiesInDeployment from 'Containers/VulnMgmt/widgets/MostCommonVulnerabiltiesInDeployment';
import TopRiskiestEntities from 'Containers/VulnMgmt/widgets/TopRiskiestEntities';
import workflowStateContext from 'Containers/workflowStateContext';
import { entityGridContainerClassName } from '../WorkflowEntityPage';

import RelatedEntitiesSideList from '../RelatedEntitiesSideList';
import TableWidgetFixableCves from '../TableWidgetFixableCves';

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
    priority: 0,
    vulnCount: 0,
    vulnerabilities: [],
};

const VulnMgmtDeploymentOverview = ({ data, entityContext }) => {
    const workflowState = useContext(workflowStateContext);

    // guard against incomplete GraphQL-cached data
    const safeData = { ...emptyDeployment, ...data };

    const { id, cluster, priority, namespace, namespaceId, labels, annotations } = safeData;

    const metadataKeyValuePairs = [];

    if (!entityContext[entityTypes.CLUSTER] && cluster?.name && cluster?.id) {
        const clusterLink = workflowState
            .pushRelatedEntity(entityTypes.CLUSTER, cluster.id)
            .toUrl();
        metadataKeyValuePairs.push({
            key: 'Cluster',
            value: cluster && cluster.name && <Link to={clusterLink}>{cluster.name}</Link>,
        });
    }
    if (!entityContext[entityTypes.NAMESPACE] && namespace && namespaceId) {
        const namespaceLink = workflowState
            .pushRelatedEntity(entityTypes.NAMESPACE, namespaceId)
            .toUrl();
        metadataKeyValuePairs.push({
            key: 'Namespace',
            value: <Link to={namespaceLink}>{namespace}</Link>,
        });
    }

    const deploymentStats = [<RiskScore key="risk-score" score={priority} />];
    const currentEntity = { [entityTypes.DEPLOYMENT]: id };
    const newEntityContext = { ...entityContext, ...currentEntity };

    const deploymentFindingsContent = (
        <div className="flex pdf-page pdf-stretch pdf-new relative rounded mb-4 ml-4 mr-4">
            <TableWidgetFixableCves
                workflowState={workflowState}
                entityContext={entityContext}
                entityType={entityTypes.DEPLOYMENT}
                name={safeData?.name}
                id={safeData?.id}
                vulnType={entityTypes.IMAGE_CVE}
            />
        </div>
    );

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
                                title="Details and metadata"
                                labels={labels}
                                annotations={annotations}
                            />
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
