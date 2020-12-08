import React, { ReactElement } from 'react';

import CategoryOverview from './CategoryOverview';

import {
    Cluster,
    clusterStatusLabelMap,
    clusterStatusStyleMap,
    getClusterStatusCountMap,
} from '../utils/clusters';

type Props = {
    clusters: Cluster[];
};

const ClusterOverview = ({ clusters }: Props): ReactElement => {
    const clusterStatusCountMap = getClusterStatusCountMap(clusters);

    return (
        <ul className="p-1 w-full">
            {Object.keys(clusterStatusCountMap).map((key) => (
                <li className="p-1" key={key} data-testid={key}>
                    <CategoryOverview
                        count={clusterStatusCountMap[key]}
                        label={clusterStatusLabelMap[key]}
                        style={clusterStatusStyleMap[key]}
                    />
                </li>
            ))}
        </ul>
    );
};

export default ClusterOverview;
