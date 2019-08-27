import React from 'react';
import entityTypes from 'constants/entityTypes';
import { NODES_WITH_CONTROL } from 'queries/controls';
import NoResultsMessage from 'Components/NoResultsMessage';

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
                if (!data) return null;
                const { entities = [] } = data;
                if (entities.length === 0)
                    return (
                        <NoResultsMessage
                            message="No nodes failing any controls"
                            className="p-6"
                            icon="info"
                        />
                    );
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
