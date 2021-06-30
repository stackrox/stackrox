import React, { ReactElement } from 'react';

import Loader from 'Components/Loader';
import { labelClassName } from 'constants/form.constants';
import useFeatureFlagEnabled from 'hooks/useFeatureFlagEnabled';
import { knownBackendFlags } from 'utils/featureFlags';
import ClusterSummary from './Components/ClusterSummary';
import StaticConfigurationSection from './StaticConfigurationSection';
import DynamicConfigurationSection from './DynamicConfigurationSection';
import ClusterLabelsTable from './ClusterLabelsTable';
import { CentralEnv, Cluster } from './clusterTypes';

type ClusterEditFormProps = {
    centralEnv: CentralEnv;
    centralVersion: string;
    selectedCluster: Cluster;
    handleChange: () => void;
    handleChangeLabels: (labels) => void;
    isLoading: boolean;
};

function ClusterEditForm({
    centralEnv,
    centralVersion,
    selectedCluster,
    handleChange,
    handleChangeLabels,
    isLoading,
}: ClusterEditFormProps): ReactElement {
    const hasScopedAccessControl = useFeatureFlagEnabled(
        knownBackendFlags.ROX_SCOPED_ACCESS_CONTROL
    );
    if (isLoading) {
        return <Loader />;
    }
    return (
        <div className="bg-base-200 px-4 w-full">
            {/* @TODO, replace open prop with dynamic logic, based on clusterType */}
            {selectedCluster.id && (
                <ClusterSummary
                    healthStatus={selectedCluster.healthStatus}
                    status={selectedCluster.status}
                    centralVersion={centralVersion}
                    clusterId={selectedCluster.id}
                />
            )}
            <form
                className="grid grid-columns-1 md:grid-columns-2 grid-gap-4 xl:grid-gap-6 mb-4 w-full"
                data-testid="cluster-form"
            >
                <StaticConfigurationSection
                    centralEnv={centralEnv}
                    handleChange={handleChange}
                    selectedCluster={selectedCluster}
                />
                <div>
                    <DynamicConfigurationSection
                        dynamicConfig={selectedCluster.dynamicConfig}
                        helmConfig={selectedCluster.helmConfig}
                        handleChange={handleChange}
                        clusterType={selectedCluster.type}
                    />
                    {hasScopedAccessControl && (
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
                    )}
                </div>
            </form>
        </div>
    );
}

export default ClusterEditForm;
