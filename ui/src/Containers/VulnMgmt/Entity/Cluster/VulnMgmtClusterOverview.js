import React, { useContext } from 'react';

import CollapsibleSection from 'Components/CollapsibleSection';
import DateTimeField from 'Components/DateTimeField';
import Metadata from 'Components/Metadata';
import RiskScore from 'Components/RiskScore';
import StatusChip from 'Components/StatusChip';
import workflowStateContext from 'Containers/workflowStateContext';
import entityTypes from 'constants/entityTypes';

import TopRiskyEntitiesByVulnerabilities from '../../widgets/TopRiskyEntitiesByVulnerabilities';
import MostRecentVulnerabilities from '../../widgets/MostRecentVulnerabilities';
import TopRiskiestImagesAndComponents from '../../widgets/TopRiskiestImagesAndComponents';
import DeploymentsWithMostSeverePolicyViolations from '../../widgets/DeploymentsWithMostSeverePolicyViolations';
import RelatedEntitiesSideList from '../RelatedEntitiesSideList';

const VulnMgmtClusterOverview = ({ data }) => {
    const workflowState = useContext(workflowStateContext);

    const {
        priority,
        policyStatus,
        createdAt,
        status: { orchestratorMetadata = null },
        istioEnabled,
        deploymentCount,
        imageComponentCount,
        imageCount,
        namespaceCount,
        policyCount,
        vulnCount
    } = data;

    const { version = 'N/A' } = orchestratorMetadata;

    function yesNoMaybe(value) {
        if (!value && value !== false) {
            return '—';
        }
        return value ? 'Yes' : 'No';
    }

    const metadataKeyValuePairs = [
        {
            key: 'Created',
            value: <DateTimeField date={createdAt} />
        },
        {
            key: 'K8s version',
            value: version
        },
        {
            key: 'Istio Enabled',
            value: yesNoMaybe(istioEnabled)
        }
    ];

    const clusterStats = [
        <RiskScore key="risk-score" score={priority} />,
        <React.Fragment key="policy-status">
            <span className="pr-1">Policy status:</span>
            <StatusChip status={policyStatus.status} />
        </React.Fragment>
    ];

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
            case entityTypes.NAMESPACE:
                return namespaceCount;
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
                    <CollapsibleSection title="Cluster Details">
                        <div className="mx-4 grid grid-gap-6 xxxl:grid-gap-8 md:grid-columns-3 mb-4 pdf-page">
                            <div className="s-1">
                                <Metadata
                                    className="h-full mx-4 min-w-48 bg-base-100"
                                    keyValuePairs={metadataKeyValuePairs}
                                    statTiles={clusterStats}
                                    title="Details & Metadata"
                                />
                            </div>
                            <div className="sx-2 sy-1">
                                <TopRiskyEntitiesByVulnerabilities />
                            </div>
                            <div className="s-1">
                                <MostRecentVulnerabilities />
                            </div>
                            <div className="s-1">
                                <TopRiskiestImagesAndComponents />
                            </div>
                            <div className="s-1">
                                <DeploymentsWithMostSeverePolicyViolations />
                            </div>
                        </div>
                    </CollapsibleSection>
                </div>
                <RelatedEntitiesSideList
                    entityType={entityTypes.CLUSTER}
                    workflowState={workflowState}
                    getCountData={getCountData}
                />
            </div>
        </div>
    );
};

export default VulnMgmtClusterOverview;
