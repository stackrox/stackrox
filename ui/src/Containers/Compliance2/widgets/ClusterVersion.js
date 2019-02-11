import React from 'react';
import PropTypes from 'prop-types';
import { resourceTypes } from 'constants/entityTypes';
import { CLUSTER_VERSION_QUERY } from 'queries/cluster';
import { clusterVersionLabels } from 'messages/common';
import { format as dateFormat } from 'date-fns';

import Widget from 'Components/Widget';
import Query from 'Components/ThrowingQuery';
import Loader from 'Components/Loader';

const queryMap = {
    [resourceTypes.CLUSTER]: CLUSTER_VERSION_QUERY
};

const ClusterVersion = ({ entityType, params }) => {
    const query = queryMap[entityType];
    const variables = { id: params.entityId };

    return (
        <Query query={query} variables={variables} pollInterval={5000}>
            {({ loading, data }) => {
                let contents = <Loader />;
                let headerText = '';
                if (!loading && data && data.cluster) {
                    const { orchestratorMetadata, type } = data.cluster;
                    headerText = clusterVersionLabels[type];
                    contents = (
                        <div className="px-2 py-8 w-full flex flex-col items-center justify-between">
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
    entityType: PropTypes.string.isRequired,
    params: PropTypes.shape({}).isRequired
};

export default ClusterVersion;
