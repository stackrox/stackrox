import React from 'react';
import PropTypes from 'prop-types';
import entityTypes from 'constants/entityTypes';
import { NODES_WITH_CONTROL } from 'queries/controls';
import NoResultsMessage from 'Components/NoResultsMessage';
import { useQuery } from 'react-apollo';
import Raven from 'raven-js';
import queryService from 'modules/queryService';
import { entityAcrossControlsColumns } from 'constants/listColumns';

import Loader from 'Components/Loader';
import TableWidget from './TableWidget';

const filterByEntityContext = entityContext => {
    const result = Object.keys(entityContext).reduce((acc, entityType) => {
        const entityId = entityContext[entityType];
        acc[`${entityType} Id`] = entityId;
        return acc;
    }, {});
    return queryService.objectToWhereClause(result);
};

export const getRelatedEntities = (data, entityType) => {
    const { results } = data;
    if (!results.length) return [];
    const relatedEntities = {};
    let entityKey = 0;
    results[0].aggregationKeys.forEach(({ scope }, idx) => {
        if (scope === entityTypes[entityType]) entityKey = idx;
    });
    results.forEach(({ keys, numPassing, numFailing }) => {
        const { id } = keys[entityKey];
        if (!relatedEntities[id]) {
            relatedEntities[id] = {
                ...keys[entityKey],
                passing: numFailing === 0 && numPassing !== 0
            };
        } else if (numFailing) relatedEntities[id].passing = false;
    });

    return Object.values(relatedEntities);
};

const NodesWithFailedControls = props => {
    const { entityType, entityContext } = props;
    const { loading, error, data } = useQuery(NODES_WITH_CONTROL, {
        variables: {
            groupBy: [entityTypes.CONTROL, entityTypes.NODE],
            where: filterByEntityContext(entityContext)
        }
    });
    if (loading)
        return (
            <div className="flex flex-1 items-center justify-center p-6">
                <Loader />
            </div>
        );
    if (error) Raven.captureException(error);
    if (!data) return null;
    const { entities = [] } = data;
    if (entities.length === 0)
        return (
            <NoResultsMessage
                message={`No nodes failing ${
                    entityType === entityTypes.CONTROL ? 'this control' : 'any controls'
                }`}
                className="p-6"
                icon="info"
            />
        );

    const localRelatedEntities = getRelatedEntities(entities, entityTypes.NODE);
    const failingRelatedEntities = localRelatedEntities.filter(
        relatedEntity => !relatedEntity.passing
    );
    const count = failingRelatedEntities.length;
    if (count === 0)
        return (
            <NoResultsMessage
                message={`No nodes failing ${
                    entityType === entityTypes.CONTROL ? 'this control' : 'any controls'
                }`}
                className="p-6 shadow"
                icon="info"
            />
        );
    const tableHeader = `${count} ${count === 1 ? 'node is' : 'nodes are'} ${
        entityType === entityTypes.CONTROL ? 'failing this control' : 'failing controls'
    }`;
    return (
        <TableWidget
            entityType={entityTypes.NODE}
            header={tableHeader}
            rows={failingRelatedEntities}
            noDataText="No Nodes"
            className="bg-base-100 w-full"
            columns={entityAcrossControlsColumns[entityTypes.NODE]}
            idAttribute="id"
            defaultSorted={[
                {
                    id: 'name',
                    desc: false
                }
            ]}
        />
    );
};

NodesWithFailedControls.propTypes = {
    entityType: PropTypes.string.isRequired,
    entityContext: PropTypes.shape({}).isRequired
};

export default NodesWithFailedControls;
