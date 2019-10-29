import React, { useContext } from 'react';

import CollapsibleSection from 'Components/CollapsibleSection';
import Metadata from 'Components/Metadata';
import RiskScore from 'Components/RiskScore';
import StatusChip from 'Components/StatusChip';
import entityTypes from 'constants/entityTypes';
import MostRecentVulnerabilities from 'Containers/VulnMgmt/widgets/MostRecentVulnerabilities';
import MostCommonVulnerabiltiesInDeployment from 'Containers/VulnMgmt/widgets/MostCommonVulnerabiltiesInDeployment';
import TopRiskiestImagesAndComponents from 'Containers/VulnMgmt/widgets/TopRiskiestImagesAndComponents';
import workflowStateContext from 'Containers/workflowStateContext';

// @TODO: update to usable for Vuln Mgmt
import PolicyViolationsBySeverity from 'Containers/ConfigManagement/Dashboard/widgets/PolicyViolationsBySeverity';

import RelatedEntitiesSideList from '../RelatedEntitiesSideList';

const VulnMgmtDeploymentOverview = ({ data }) => {
    const workflowState = useContext(workflowStateContext);

    const {
        id,
        cluster,
        priority,
        namespace,
        policyStatus,
        labels,
        annotations,
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
                            <div className="s-1">
                                <Metadata
                                    className="h-full min-w-48 bg-base-100"
                                    keyValuePairs={metadataKeyValuePairs}
                                    statTiles={deploymentStats}
                                    title="Details & Metadata"
                                    labels={labels}
                                    annotations={annotations}
                                />
                            </div>
                            <div className="s-1">
                                <PolicyViolationsBySeverity />
                            </div>
                            <div className="s-1">
                                <div className="h-full">CvesByCvssScore goes here</div>
                            </div>
                            <div className="s-1">
                                <MostRecentVulnerabilities />
                            </div>
                            <div className="s-1">
                                <MostCommonVulnerabiltiesInDeployment deploymentId={id} />
                            </div>
                            <div className="s-1">
                                <TopRiskiestImagesAndComponents limit={5} />
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
