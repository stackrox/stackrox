import React from 'react';
import uniq from 'lodash/uniq';

import Query from 'Components/ThrowingQuery';
import TileLink from 'Components/TileLink';

import NAMESPACES_QUERY from 'queries/namespace';
import { resourceLabels } from 'messages/common';

const NamespacesTile = () => (
    <Query query={NAMESPACES_QUERY} action="list">
        {({ loading, data }) => {
            let value = 0;
            if (!loading) {
                value = uniq(
                    data.deployments.map(
                        deployment => `${deployment.cluster}-${deployment.namespace}`
                    )
                ).length;
            }
            return (
                <TileLink
                    value={value}
                    caption={resourceLabels.NAMESPACE}
                    to="/main/compliance2/namespaces"
                    loading={loading}
                />
            );
        }}
    </Query>
);

export default NamespacesTile;
