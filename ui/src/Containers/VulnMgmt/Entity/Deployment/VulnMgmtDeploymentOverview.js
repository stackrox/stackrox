import React, { useContext } from 'react';
import pluralize from 'pluralize';

import CollapsibleSection from 'Components/CollapsibleSection';
import StatusChip from 'Components/StatusChip';
import TileList from 'Components/TileList';
import Widget from 'Components/Widget';
import entityTypes from 'constants/entityTypes';
import MostRecentVulnerabilities from 'Containers/VulnMgmt/widgets/MostRecentVulnerabilities';
import MostCommonVulnerabiltiesInDeployment from 'Containers/VulnMgmt/widgets/MostCommonVulnerabiltiesInDeployment';
import workflowStateContext from 'Containers/workflowStateContext';

import WorkflowStateMgr from 'modules/WorkflowStateManager';
import { generateURL } from 'modules/URLReadWrite';

function getPushEntityType(workflowState, entityType) {
    const workflowStateMgr = new WorkflowStateMgr(workflowState);
    workflowStateMgr.pushList(entityType);
    const url = generateURL(workflowStateMgr.workflowState);

    return url;
}

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

    const matchesTiles = failingPolicyCount
        ? [
              {
                  count: failingPolicyCount,
                  label: pluralize('Policy', failingPolicyCount),
                  url: getPushEntityType(workflowState, entityTypes.POLICY)
              }
          ]
        : [];

    const containsTiles =
        vulnCount || imageCount || imageComponentCount
            ? [
                  {
                      count: vulnCount,
                      label: pluralize('CVE', vulnCount),
                      url: getPushEntityType(workflowState, entityTypes.CVE)
                  },
                  {
                      count: imageCount,
                      label: pluralize('Image', imageCount),
                      url: getPushEntityType(workflowState, entityTypes.IMAGE)
                  },
                  {
                      count: imageComponentCount,
                      label: pluralize('Component', imageComponentCount),
                      url: getPushEntityType(workflowState, entityTypes.COMPONENT)
                  }
              ]
            : [];

    return (
        <div className="w-full h-full" id="capture-dashboard-stretch">
            <div className="flex h-full">
                <div className="flex flex-col flex-grow">
                    <CollapsibleSection title="CVE summary">
                        <div className="mx-4 grid grid-gap-6 xxxl:grid-gap-8 md:grid-columns-3 mb-4 pdf-page">
                            <div className="">
                                {/* TODO: abstract this into a new, more powerful Metadata component */}
                                <Widget
                                    header="Details & Metadata"
                                    className="bg-base-100 h-48 mb-4 flex-grow max-w-6xl h-full"
                                >
                                    <div className="flex flex-col w-full">
                                        <div className="border-b border-base-300 text-base-500 flex justify-between items-center">
                                            <div className="flex flex-grow p-4 justify-center items-center border-r-2 border-base-300 border-dotted">
                                                <span className="pr-1">Risk score:</span>
                                                <span className="pl-1 text-3xl">{priority}</span>
                                            </div>
                                            <div className="flex flex-col p-4 flex-grow justify-center text-center">
                                                <span>Policy status:</span>
                                                <StatusChip status={policyStatus} />
                                            </div>
                                        </div>
                                        <div>
                                            Cluster: {cluster.name} / Namespace: {namespace}
                                        </div>
                                    </div>
                                </Widget>
                            </div>
                            <div>
                                <MostRecentVulnerabilities />
                            </div>
                            <div>
                                <MostCommonVulnerabiltiesInDeployment deploymentId={id} />
                            </div>
                        </div>
                    </CollapsibleSection>
                </div>

                <div className="bg-primary-300 h-full relative">
                    {/* TODO: decide if this should be added as custom tailwind class, or a "component" CSS class in app.css */}
                    <h2
                        style={{
                            position: 'relative',
                            left: '-0.5rem',
                            width: 'calc(100% + 0.5rem)'
                        }}
                        className="my-4 p-2 bg-primary-700 text-base text-base-100 rounded-l"
                    >
                        Related entities
                    </h2>
                    {matchesTiles.length > 0 && <TileList items={matchesTiles} title="Matches" />}
                    {containsTiles.length > 0 && (
                        <TileList items={containsTiles} title="Contains" />
                    )}
                </div>
            </div>
        </div>
    );
};

export default VulnMgmtDeploymentOverview;
