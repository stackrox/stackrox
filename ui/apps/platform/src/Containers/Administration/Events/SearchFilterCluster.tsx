import { useEffect, useState } from 'react';
import { Divider, SelectOption } from '@patternfly/react-core';
import SelectSingle from 'Components/SelectSingle/SelectSingle';
import { fetchClusters } from 'services/ClustersService';

const optionAll = 'All clusters';

type SearchFilterClusterProps = {
    isDisabled: boolean;
    cluster: string | undefined;
    setCluster: (cluster: string | undefined) => void;
};

function SearchFilterCluster({ isDisabled, cluster, setCluster }: SearchFilterClusterProps) {
    const [clusterNames, setClusterNames] = useState<string[]>([]);

    useEffect(() => {
        fetchClusters()
            .then((clusters) => setClusterNames(clusters.map((c) => c.name)))
            .catch(() => {});
    }, []);

    function onSelect(_id: string, selection: string) {
        setCluster(selection === optionAll ? undefined : selection);
    }

    const options = [
        <SelectOption key="All" value={optionAll}>
            {optionAll}
        </SelectOption>,
        <Divider key="Divider" />,
        ...clusterNames.map((name) => (
            <SelectOption key={name} value={name}>
                {name}
            </SelectOption>
        )),
    ];

    return (
        <SelectSingle
            id="cluster-filter"
            value={cluster ?? optionAll}
            handleSelect={onSelect}
            isDisabled={isDisabled}
            placeholderText="Select cluster"
            toggleAriaLabel="Cluster filter menu toggle"
        >
            {options}
        </SelectSingle>
    );
}

export default SearchFilterCluster;
