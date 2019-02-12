import React from 'react';
import { NAMESPACES_QUERY } from 'queries/namespace';
import { resourceLabels } from 'messages/common';
import URLService from 'modules/URLService';
import contextTypes from 'constants/contextTypes';
import pageTypes from 'constants/pageTypes';
import entityTypes from 'constants/entityTypes';
import uniq from 'lodash/uniq';

import Query from 'Components/ThrowingQuery';
import TileLink from 'Components/TileLink';

const link = URLService.getLinkTo(contextTypes.COMPLIANCE, pageTypes.LIST, {
    entityType: entityTypes.NAMESPACE
});

const NamespacesTile = () => (
    <Query query={NAMESPACES_QUERY} action="list">
        {({ loading, data }) => {
            let value = 0;
            if (!loading) {
                value = uniq(
                    data.results.map(deployment => `${deployment.cluster}-${deployment.namespace}`)
                ).length;
            }
            return (
                <TileLink
                    value={value}
                    caption={resourceLabels.NAMESPACE}
                    to={link.url}
                    loading={loading}
                />
            );
        }}
    </Query>
);

export default NamespacesTile;
