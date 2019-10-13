import React from 'react';
import CollapsibleSection from 'Components/CollapsibleSection';
import Metadata from 'Components/Metadata';
import entityTypes from 'constants/entityTypes';
import RelatedEntityListCount from 'Components/RelatedEntityListCount';

const VulnMgmtDeploymentOverview = ({ data }) => {
    const {
        nodeCount,
        deploymentCount,
        namespaceCount,
        subjectCount,
        serviceAccountCount,
        k8sroleCount,
        secretCount,
        imageCount,
        complianceControlCount,
        status: { orchestratorMetadata = null }
    } = data;

    const { version = 'N/A' } = orchestratorMetadata;

    const metadataKeyValuePairs = [
        {
            key: 'K8s version',
            value: version
        }
    ];

    const { passingCount, failingCount, unknownCount } = complianceControlCount;
    const totalControlCount = passingCount + failingCount + unknownCount;

    return (
        <div className="w-full" id="capture-dashboard-stretch">
            <CollapsibleSection title="Cluster Details">
                <div className="flex flex-wrap pdf-page">
                    <Metadata
                        className="mx-4 min-w-48 bg-base-100 h-48 mb-4"
                        keyValuePairs={metadataKeyValuePairs}
                    />
                    <RelatedEntityListCount
                        className="mx-4 min-w-48 h-48 mb-4"
                        name="Nodes"
                        value={nodeCount}
                        entityType={entityTypes.NODE}
                    />
                    <RelatedEntityListCount
                        className="mx-4 min-w-48 h-48 mb-4"
                        name="Namespaces"
                        value={namespaceCount}
                        entityType={entityTypes.NAMESPACE}
                    />
                    <RelatedEntityListCount
                        className="mx-4 min-w-48 h-48 mb-4"
                        name="Deployments"
                        value={deploymentCount}
                        entityType={entityTypes.DEPLOYMENT}
                    />
                    <RelatedEntityListCount
                        className="mx-4 min-w-48 h-48 mb-4"
                        name="Secrets"
                        value={secretCount}
                        entityType={entityTypes.SECRET}
                    />
                    <RelatedEntityListCount
                        className="mx-4 min-w-48 h-48 mb-4"
                        name="Images"
                        value={imageCount}
                        entityType={entityTypes.IMAGE}
                    />
                    <RelatedEntityListCount
                        className="mx-4 min-w-48 h-48 mb-4"
                        name="Users & Groups"
                        value={subjectCount}
                        entityType={entityTypes.SUBJECT}
                    />
                    <RelatedEntityListCount
                        className="mx-4 min-w-48 h-48 mb-4"
                        name="Service Accounts"
                        value={serviceAccountCount}
                        entityType={entityTypes.SERVICE_ACCOUNT}
                    />
                    <RelatedEntityListCount
                        className="mx-4 min-w-48 h-48 mb-4"
                        name="Roles"
                        value={k8sroleCount}
                        entityType={entityTypes.ROLE}
                    />
                    <RelatedEntityListCount
                        className="mx-4 min-w-48 h-48 mb-4"
                        name="CIS Controls"
                        value={totalControlCount}
                        entityType={entityTypes.CONTROL}
                    />
                </div>
            </CollapsibleSection>
        </div>
    );
};

export default VulnMgmtDeploymentOverview;
