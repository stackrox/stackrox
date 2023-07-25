import React, { useContext } from 'react';
import { Link } from 'react-router-dom';
import { format } from 'date-fns';

import CollapsibleSection from 'Components/CollapsibleSection';
import Metadata from 'Components/Metadata';
import RiskScore from 'Components/RiskScore';
import entityTypes from 'constants/entityTypes';
import TopCvssLabel from 'Components/TopCvssLabel';
import ScanDataMessage from 'Containers/VulnMgmt/Components/ScanDataMessage';
import getNodeScanMessage from 'Containers/VulnMgmt/VulnMgmt.utils/getNodeScanMessage';
import CvesByCvssScore from 'Containers/VulnMgmt/widgets/CvesByCvssScore';
import workflowStateContext from 'Containers/workflowStateContext';
import dateTimeFormat from 'constants/dateTimeFormat';
import { entityGridContainerClassName } from '../WorkflowEntityPage';
import RelatedEntitiesSideList from '../RelatedEntitiesSideList';
import TableWidgetFixableCves from '../TableWidgetFixableCves';

const emptyNode = {
    annotations: [],
    clusterId: '',
    clusterName: '',
    created: '',
    id: '',
    labels: [],
    name: '',
    nodeStatus: '',
    priority: 0,
    topVuln: {
        cvss: 0,
        scoreVersion: '',
    },
    kubeletVersion: '',
    kernelVersion: '',
    osImage: '',
    containerRuntimeVersion: '',
    joinedAt: '',
    vulnCount: 0,
};

const VulnMgmtNodeOverview = ({ data, entityContext }) => {
    const workflowState = useContext(workflowStateContext);

    // guard against incomplete GraphQL-cached data
    const safeData = { ...emptyNode, ...data };

    const {
        id,
        clusterId,
        clusterName,
        priority,
        topVuln,
        labels,
        annotations,
        kubeletVersion,
        kernelVersion,
        osImage,
        containerRuntimeVersion,
        joinedAt,
        // eslint-disable-next-line no-unused-vars
        vulnCount,
        scan,
        notes,
    } = safeData;
    safeData.componentCount = scan?.components?.length || 0;

    safeData.nodeComponentCount = scan?.components?.length || 0;

    const metadataKeyValuePairs = [
        {
            key: 'Kubelet Version',
            value: kubeletVersion,
        },
        {
            key: 'Kernel Version',
            value: kernelVersion,
        },
        {
            key: 'Operating System',
            value: osImage,
        },
        {
            key: 'Container Runtime',
            value: containerRuntimeVersion,
        },
        {
            key: 'Join Time',
            value: joinedAt ? format(joinedAt, dateTimeFormat) : 'N/A',
        },
    ];

    if (!entityContext[entityTypes.CLUSTER]) {
        const clusterLink = workflowState.pushRelatedEntity(entityTypes.CLUSTER, clusterId).toUrl();
        metadataKeyValuePairs.unshift({
            key: 'Cluster',
            value: (
                <Link to={clusterLink} className="underline">
                    {clusterName}
                </Link>
            ),
        });
    }

    const nodeStats = [<RiskScore key="risk-score" score={priority} />];
    if (topVuln) {
        const { cvss, scoreVersion } = topVuln;
        nodeStats.push(<TopCvssLabel key="top-cvss" cvss={cvss} version={scoreVersion} expanded />);
    }

    const currentEntity = { [entityTypes.NODE]: id };
    const newEntityContext = { ...entityContext, ...currentEntity };

    const scanMessage = getNodeScanMessage(notes || [], scan?.notes || []);

    return (
        <div className="flex h-full">
            <div className="flex flex-col flex-grow min-w-0">
                <ScanDataMessage header={scanMessage.header} body={scanMessage.body} />
                <CollapsibleSection title="Node Summary">
                    <div className={entityGridContainerClassName}>
                        <div className="sx-2">
                            <Metadata
                                className="h-full min-w-48 bg-base-100 pdf-page"
                                keyValuePairs={metadataKeyValuePairs}
                                statTiles={nodeStats}
                                title="Details and metadata"
                                labels={labels}
                                annotations={annotations}
                            />
                        </div>
                        <div className="s-1">
                            <CvesByCvssScore entityContext={currentEntity} />
                        </div>
                    </div>
                </CollapsibleSection>
                <CollapsibleSection title="Node Findings">
                    <div className="flex pdf-page pdf-stretch pdf-new shadow relative rounded bg-base-100 mb-4 ml-4 mr-4">
                        <TableWidgetFixableCves
                            workflowState={workflowState}
                            entityContext={entityContext}
                            entityType={entityTypes.NODE}
                            name={safeData?.name}
                            id={safeData?.id}
                            vulnType={entityTypes.NODE_CVE}
                        />
                    </div>
                </CollapsibleSection>
            </div>
            <RelatedEntitiesSideList
                entityType={entityTypes.NODE}
                entityContext={newEntityContext}
                data={safeData}
            />
        </div>
    );
};

export default VulnMgmtNodeOverview;
