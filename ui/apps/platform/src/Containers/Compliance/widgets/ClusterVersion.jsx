import React from 'react';
import PropTypes from 'prop-types';
import { CLUSTER_VERSION_QUERY as QUERY } from 'queries/cluster';
import { clusterVersionLabels } from 'messages/common';
import { format as dateFormat } from 'date-fns';
import NoResultsMessage from 'Components/NoResultsMessage';

import Widget from 'Components/Widget';
import Query from 'Components/CacheFirstQuery';
import Loader from 'Components/Loader';

const ClusterVersion = ({ clusterId }) => {
    const variables = { id: clusterId };
    return (
        <Query query={QUERY} variables={variables}>
            {({ loading, data }) => {
                let contents = <Loader />;
                let headerText = '';
                if (!loading && data) {
                    const cluster = data.cluster || {};
                    const { type } = cluster;
                    const status = cluster.status || {};
                    const { orchestratorMetadata } = status;
                    if (!orchestratorMetadata || !type) {
                        contents = (
                            <NoResultsMessage message="An error occurred retrieving cluster version data." />
                        );
                    } else {
                        headerText = clusterVersionLabels[type];
                        let version;
                        if (type === 'KUBERNETES_CLUSTER') {
                            version = orchestratorMetadata.version;
                        } else if (orchestratorMetadata.openshiftVersion) {
                            version = orchestratorMetadata.openshiftVersion;
                        } else {
                            version = 'OpenShift version cannot be determined';
                        }
                        contents = (
                            <div className="py-8 w-full flex flex-col items-center justify-between">
                                <div className="text-4xl text-center" data-testid="cluster-version">
                                    {version}
                                </div>
                                <div>
                                    Build date:&nbsp;
                                    {dateFormat(orchestratorMetadata.buildDate, 'MMMM DD, YYYY')}
                                </div>
                            </div>
                        );
                    }
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
    clusterId: PropTypes.string,
};

ClusterVersion.defaultProps = {
    clusterId: null,
};

export default ClusterVersion;
