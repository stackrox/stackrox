import React from 'react';
import entityTypes from 'constants/entityTypes';
import { NODES_WITH_CONTROL } from 'queries/controls';

import Query from 'Components/ThrowingQuery';
import Loader from 'Components/Loader';
import EntityWithFailedControls from './EntityWithFailedControls';

const NodesWithFailedControls = props => {
    const variables = {
        groupBy: [entityTypes.CONTROL, entityTypes.NODE]
    };
    return (
        <Query query={NODES_WITH_CONTROL} variables={variables}>
            {({ loading, data }) => {
                if (loading) return <Loader />;
                const { entities = [] } = data;
                return (
                    <EntityWithFailedControls
                        entityType={entityTypes.NODE}
                        entities={entities}
                        {...props}
                    />
                );
            }}
        </Query>
    );
};

export default NodesWithFailedControls;
