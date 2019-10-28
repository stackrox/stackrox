import React, { useContext } from 'react';
import workflowStateContext from 'Containers/workflowStateContext';

import CollapsibleSection from 'Components/CollapsibleSection';
import StatusChip from 'Components/StatusChip';
import Widget from 'Components/Widget';
import Tabs from 'Components/Tabs';
import TabContent from 'Components/TabContent';
import ResourceCountPopper from 'Components/ResourceCountPopper';
import entityTypes from 'constants/entityTypes';
import TopRiskyEntitiesByVulnerabilities from 'Containers/VulnMgmt/widgets/TopRiskyEntitiesByVulnerabilities';
import MostRecentVulnerabilities from 'Containers/VulnMgmt/widgets/MostRecentVulnerabilities';
import TopRiskiestImagesAndComponents from 'Containers/VulnMgmt/widgets/TopRiskiestImagesAndComponents';
import DeploymentsWithMostSeverePolicyViolations from 'Containers/VulnMgmt/widgets/DeploymentsWithMostSeverePolicyViolations';

import RelatedEntitiesSideList from '../RelatedEntitiesSideList';

const VulnMgmtNamespaceOverview = ({ data }) => {
    const workflowState = useContext(workflowStateContext);

    const {
        metadata,
        policyStatus,
        policyCount,
        deploymentCount,
        imageCount,
        imageComponentCount,
        vulnCount
    } = data;

    const { clusterName, priority, labels } = metadata;

    function getCountData(entityType) {
        switch (entityType) {
            case entityTypes.DEPLOYMENT:
                return deploymentCount;
            case entityTypes.COMPONENT:
                return imageComponentCount;
            case entityTypes.CVE:
                return vulnCount;
            case entityTypes.IMAGE:
                return imageCount;
            case entityTypes.POLICY:
                return policyCount;
            default:
                return 0;
        }
    }

    return (
        <div className="w-full h-full" id="capture-dashboard-stretch">
            <div className="flex h-full">
                <div className="flex flex-col flex-grow">
                    <CollapsibleSection title="Namespace summary">
                        <div className="mx-4 grid grid-gap-6 xxxl:grid-gap-8 md:grid-columns-3 mb-4 pdf-page">
                            <div className="">
                                {/* TODO: abstract this into a new, more powerful Metadata component */}
                                <Widget
                                    header="Details & Metadata"
                                    className="bg-base-100 h-48 mb-4 flex-grow max-w-6xl h-full"
                                >
                                    <div className="flex flex-col w-full bg-counts-widget">
                                        <div className="border-b border-base-300 text-base-500 flex justify-between items-center">
                                            <div className="flex flex-grow p-4 justify-center items-center border-r-2 border-base-300 border-dotted">
                                                <span className="pr-1">Risk score:</span>
                                                <span className="pl-1 text-3xl">{priority}</span>
                                            </div>
                                            <div className="flex flex-col p-4 flex-grow justify-center text-center">
                                                <span>Policy status:</span>
                                                <StatusChip status={policyStatus.status} />
                                            </div>
                                        </div>
                                        <div>Cluster: {clusterName}</div>
                                        <div>
                                            <ResourceCountPopper data={labels} label="Labels" />
                                        </div>
                                    </div>
                                </Widget>
                            </div>
                            <div>
                                <TopRiskyEntitiesByVulnerabilities />
                            </div>
                            <div>
                                <MostRecentVulnerabilities />
                            </div>
                            <div>
                                <TopRiskiestImagesAndComponents />
                            </div>
                            <div>
                                <DeploymentsWithMostSeverePolicyViolations />
                            </div>
                        </div>
                    </CollapsibleSection>
                    <CollapsibleSection title="Namespace findings">
                        <div className="flex pdf-page pdf-stretch shadow rounded relative rounded bg-base-100 mb-4 ml-4 mr-4">
                            <Tabs
                                hasTabSpacing
                                headers={[{ text: 'Policies' }, { text: 'Fixable CVEs' }]}
                            >
                                <TabContent>
                                    <div>policies</div>
                                </TabContent>
                                <TabContent>
                                    <div>fixable CVEs</div>
                                </TabContent>
                            </Tabs>
                        </div>
                    </CollapsibleSection>
                </div>

                <RelatedEntitiesSideList
                    entityType={entityTypes.NAMESPACE}
                    workflowState={workflowState}
                    getCountData={getCountData}
                />
            </div>
        </div>
    );
};

export default VulnMgmtNamespaceOverview;
