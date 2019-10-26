import React, { useContext } from 'react';

import CollapsibleSection from 'Components/CollapsibleSection';
import Metadata from 'Components/Metadata';
import RiskScore from 'Components/RiskScore';
import StatusChip from 'Components/StatusChip';
import entityTypes from 'constants/entityTypes';
import MostRecentVulnerabilities from 'Containers/VulnMgmt/widgets/MostRecentVulnerabilities';
import MostCommonVulnerabiltiesInDeployment from 'Containers/VulnMgmt/widgets/MostCommonVulnerabiltiesInDeployment';
import workflowStateContext from 'Containers/workflowStateContext';

import RelatedEntitiesSideList from '../RelatedEntitiesSideList';

const VulnMgmtDeploymentOverview = ({ data }) => {
    const workflowState = useContext(workflowStateContext);

    const {
        id,
        cluster,
        priority,
        namespace,
        policyStatus,
        failingPolicyCount,
        imageCount,
        imageComponentCount,
        vulnCount
    } = data;

    const metadataKeyValuePairs = [
        {
            key: 'Cluster:',
            value: cluster && cluster.name
        },
        {
            key: 'Namespace:',
            value: namespace
        }
    ];

    const deploymentStats = [
        <RiskScore score={priority} />,
        <>
            <span className="pr-1">Policy status:</span>
            <StatusChip status={policyStatus} />
        </>
    ];

    function getCountData(entityType) {
        switch (entityType) {
            case entityTypes.COMPONENT:
                return imageComponentCount;
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

    return (
        <div className="w-full h-full" id="capture-dashboard-stretch">
            <div className="flex h-full">
                <div className="flex flex-col flex-grow">
                    <CollapsibleSection title="Deployment summary">
                        <div className="mx-4 grid grid-gap-6 xxxl:grid-gap-8 md:grid-columns-3 mb-4 pdf-page">
                            <Metadata
                                className="mx-4 min-w-48 bg-base-100 h-48 mb-4"
                                keyValuePairs={metadataKeyValuePairs}
                                statTiles={deploymentStats}
                                title="Details & Metadata"
                            />
                            <div>
                                <MostRecentVulnerabilities />
                            </div>
                            <div>
                                <MostCommonVulnerabiltiesInDeployment deploymentId={id} />
                            </div>
                        </div>
                    </CollapsibleSection>
                </div>
                <RelatedEntitiesSideList
                    entityType={entityTypes.DEPLOYMENT}
                    workflowState={workflowState}
                    getCountData={getCountData}
                />
            </div>
        </div>
    );
};

export default VulnMgmtDeploymentOverview;
