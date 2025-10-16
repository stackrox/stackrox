import { SelectOption } from '@patternfly/react-core';

import type { ClusterScopeObject } from 'services/RolesService';
import SelectSingle from 'Components/SelectSingle/SelectSingle';
import { ClusterIcon } from '../common/NetworkGraphIcons';

export type ClusterSelectorProps = {
    clusters: ClusterScopeObject[];
    selectedClusterName?: string;
    searchFilter: Partial<Record<string, string | string[]>>;
    setSearchFilter: (newFilter: Partial<Record<string, string | string[]>>) => void;
};

function ClusterSelector({
    clusters = [],
    selectedClusterName = '',
    searchFilter,
    setSearchFilter,
}: ClusterSelectorProps) {
    const handleSelect = (_name: string, value: string) => {
        if (value !== selectedClusterName) {
            const modifiedSearchObject = { ...searchFilter };
            modifiedSearchObject.Cluster = value;
            delete modifiedSearchObject.Namespace;
            delete modifiedSearchObject.Deployment;
            setSearchFilter(modifiedSearchObject);
        }
    };

    const clusterSelectOptions: JSX.Element[] = clusters.map((cluster) => (
        <SelectOption key={cluster.id} value={cluster.name} icon={<ClusterIcon />}>
            {cluster.name}
        </SelectOption>
    ));

    return (
        <SelectSingle
            id="cluster-select"
            className="cluster-select"
            toggleIcon={<ClusterIcon />}
            variant="plainText"
            placeholderText="Cluster"
            toggleAriaLabel="Select a cluster"
            value={selectedClusterName}
            handleSelect={handleSelect}
            isDisabled={clusters.length === 0}
        >
            {clusterSelectOptions}
        </SelectSingle>
    );
}

export default ClusterSelector;
