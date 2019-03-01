import React from 'react';
import PropTypes from 'prop-types';
import { CLUSTER_VERSION_QUERY as QUERY } from 'queries/cluster';
import { clusterVersionLabels } from 'messages/common';
import { format as dateFormat } from 'date-fns';

import Widget from 'Components/Widget';
import Query from 'Components/ThrowingQuery';
import Loader from 'Components/Loader';

const ClusterVersion = ({ clusterId }) => {
    const variables = { id: clusterId };
    return (
        <Query query={QUERY} variables={variables}>
            {({ loading, data }) => {
                let contents = <Loader />;
                let headerText = '';
                if (!loading && data && data.cluster) {
                    const { orchestratorMetadata, type } = data.cluster;
                    headerText = clusterVersionLabels[type];
                    contents = (
                        <div className="py-8 w-full flex flex-col items-center justify-between">
                            <div className="text-4xl text-primary-700 font-500">
                                {orchestratorMetadata.version}
                            </div>
                            <div className="text-base-500">
                                Build Date:&nbsp;
                                {dateFormat(orchestratorMetadata.buildDate, 'MMMM DD, YYYY')}
                            </div>
                        </div>
                    );
                }
                return (
                    <Widget header={headerText} bodyClassName="p-2">
                        {contents}
                    </Widget>
                );
            }}
        </Query>
    );
};

ClusterVersion.propTypes = {
    clusterId: PropTypes.string
};

ClusterVersion.defaultProps = {
    clusterId: null
};

export default ClusterVersion;
