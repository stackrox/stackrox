import React from 'react';
import gql from 'graphql-tag';
import { useQuery } from 'react-apollo';
import Raven from 'raven-js';

import SummaryTileCount from 'Components/SummaryTileCount';

const SUMMARY_COUNTS = gql`
    query summary_counts {
        clusterCount
        nodeCount
        violationCount
        deploymentCount
        imageCount
        secretCount
    }
`;

const SummaryCounts = () => {
    const { loading, error, data } = useQuery(SUMMARY_COUNTS, { pollInterval: 30000 });
    if (error) Raven.captureException(error);
    const { clusterCount, nodeCount, violationCount, deploymentCount, imageCount, secretCount } =
        data || {};
    return (
        <ul className="flex uppercase text-sm p-0 w-full">
            <SummaryTileCount label="Cluster" value={clusterCount} loading={loading} />
            <SummaryTileCount label="Node" value={nodeCount} loading={loading} />
            <SummaryTileCount label="Violation" value={violationCount} loading={loading} />
            <SummaryTileCount label="Deployment" value={deploymentCount} loading={loading} />
            <SummaryTileCount label="Image" value={imageCount} loading={loading} />
            <SummaryTileCount label="Secret" value={secretCount} loading={loading} />
        </ul>
    );
};

export default SummaryCounts;
