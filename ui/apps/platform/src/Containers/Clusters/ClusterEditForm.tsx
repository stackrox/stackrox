import React, { ReactElement } from 'react';

import Loader from 'Components/Loader';
import { labelClassName } from 'constants/form.constants';
import useFeatureFlags from 'hooks/useFeatureFlags';
import { DecommissionedClusterRetentionInfo } from 'types/clusterService.proto';

import ClusterSummary from './Components/ClusterSummary';
import StaticConfigurationSection from './StaticConfigurationSection';
import DynamicConfigurationSection from './DynamicConfigurationSection';
import ClusterLabelsTable from './ClusterLabelsTable';
import { CentralEnv, Cluster, ClusterManagerType } from './clusterTypes';

type ClusterEditFormProps = {
    centralEnv: CentralEnv;
    centralVersion: string;
    clusterRetentionInfo: DecommissionedClusterRetentionInfo;
    selectedCluster: Cluster;
    managerType: ClusterManagerType;
    handleChange: (any) => void;
    handleChangeLabels: (labels) => void;
    isLoading: boolean;
};

function ClusterEditForm({
    centralEnv,
    centralVersion,
    clusterRetentionInfo,
    selectedCluster,
    managerType,
    handleChange,
    handleChangeLabels,
    isLoading,
}: ClusterEditFormProps): ReactElement {
    const { isFeatureFlagEnabled } = useFeatureFlags();
    const isDecommissionedClusterRetentionEnabled = isFeatureFlagEnabled(
        'ROX_DECOMMISSIONED_CLUSTER_RETENTION'
    );

    if (isLoading) {
        return <Loader />;
    }
    const isManagerTypeNonConfigurable =
        managerType === 'MANAGER_TYPE_KUBERNETES_OPERATOR' ||
        managerType === 'MANAGER_TYPE_HELM_CHART';
    return (
        <div className="bg-base-200 px-4 w-full">
            {/* @TODO, replace open prop with dynamic logic, based on clusterType */}
            {selectedCluster.id && (
                <ClusterSummary
                    healthStatus={selectedCluster.healthStatus}
                    status={selectedCluster.status}
                    centralVersion={centralVersion}
                    clusterId={selectedCluster.id}
                    clusterRetentionInfo={clusterRetentionInfo}
                    isDecommissionedClusterRetentionEnabled={
                        isDecommissionedClusterRetentionEnabled
                    }
                    isManagerTypeNonConfigurable={isManagerTypeNonConfigurable}
                />
            )}
            <form
                className="grid grid-columns-1 md:grid-columns-2 grid-gap-4 xl:grid-gap-6 mb-4 w-full"
                data-testid="cluster-form"
            >
                <StaticConfigurationSection
                    centralEnv={centralEnv}
                    isManagerTypeNonConfigurable={isManagerTypeNonConfigurable}
                    handleChange={handleChange}
                    selectedCluster={selectedCluster}
                />
                <div>
                    <DynamicConfigurationSection
                        dynamicConfig={selectedCluster.dynamicConfig}
                        helmConfig={selectedCluster.helmConfig}
                        handleChange={handleChange}
                        clusterType={selectedCluster.type}
                        isManagerTypeNonConfigurable={isManagerTypeNonConfigurable}
                    />
                    <div className="pt-4">
                        <label htmlFor="labels" className={labelClassName}>
                            Cluster labels
                        </label>
                        <ClusterLabelsTable
                            labels={selectedCluster?.labels ?? {}}
                            handleChangeLabels={handleChangeLabels}
                            hasAction
                        />
                    </div>
                </div>
            </form>
        </div>
    );
}

export default ClusterEditForm;
