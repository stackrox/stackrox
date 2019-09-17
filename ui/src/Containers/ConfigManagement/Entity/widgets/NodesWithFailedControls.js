import React from 'react';
import entityTypes from 'constants/entityTypes';
import { NODES_WITH_CONTROL } from 'queries/controls';
import NoResultsMessage from 'Components/NoResultsMessage';
import { useQuery } from 'react-apollo';
import Raven from 'raven-js';

import Loader from 'Components/Loader';
import EntityWithFailedControls from './EntityWithFailedControls';

const NodesWithFailedControls = props => {
    const { loading, error, data } = useQuery(NODES_WITH_CONTROL, {
        variables: {
            groupBy: [entityTypes.CONTROL, entityTypes.NODE]
        }
    });
    if (loading) return <Loader />;
    if (error) Raven.captureException(error);
    if (!data) return null;
    const { entities = [] } = data;
    if (entities.length === 0)
        return (
            <NoResultsMessage message="No nodes failing any controls" className="p-6" icon="info" />
        );
    return (
        <EntityWithFailedControls entityType={entityTypes.NODE} entities={entities} {...props} />
    );
};

export default NodesWithFailedControls;
