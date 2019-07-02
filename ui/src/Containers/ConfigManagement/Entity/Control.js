import React from 'react';
import PropTypes from 'prop-types';
import { CONTROL_QUERY as QUERY } from 'queries/controls';
import entityTypes from 'constants/entityTypes';
import queryService from 'modules/queryService';
import { entityAcrossControlsColumns } from 'constants/listColumns';

import Query from 'Components/ThrowingQuery';
import Loader from 'Components/Loader';
import PageNotFound from 'Components/PageNotFound';
import CollapsibleSection from 'Components/CollapsibleSection';
import ControlDetails from 'Components/ControlDetails';
import RelatedEntityListCount from 'Containers/ConfigManagement/Entity/widgets/RelatedEntityListCount';
import Widget from 'Components/Widget';
import TableWidget from './widgets/TableWidget';

const getVariables = (id, relatedEntityType) => {
    const query = {
        'Control Id': id
    };
    const where = queryService.objectToWhereClause(query);
    return {
        id,
        where,
        groupBy: [entityTypes.CONTROL, relatedEntityType]
    };
};

const getRelatedEntities = (data, entityType) => {
    const relatedEntities = {};
    let entityKey = 0;
    data.results[0].aggregationKeys.forEach(({ scope }, idx) => {
        if (scope === entityTypes[entityType]) entityKey = idx;
    });
    data.results.forEach(({ keys, numFailing }) => {
        const { id, name, clusterName } = keys[entityKey];
        if (!relatedEntities[id]) {
            relatedEntities[id] = {
                id,
                name,
                clusterName
            };
        } else if (numFailing) relatedEntities[id].passing = false;
    });
    return Object.values(relatedEntities);
};

const Control = ({ id, onRelatedEntityListClick }) => (
    <Query query={QUERY} variables={getVariables(id, entityTypes.NODE)}>
        {({ loading, data }) => {
            if (loading) return <Loader />;
            const { results: entity, entities } = data;
            if (!entity) return <PageNotFound resourceType={entityTypes.CONTROL} />;

            const onRelatedEntityListClickHandler = entityListType => () => {
                onRelatedEntityListClick(entityListType);
            };

            const {
                standardId = '',
                name = '',
                description = '',
                interpretationText = ''
            } = entity;
            const relatedEntities = getRelatedEntities(entities, entityTypes.NODE);
            const failingRelatedEntities = relatedEntities.filter(
                relatedEntity => !relatedEntity.passing
            );
            const tableHeader = `${relatedEntities.length} nodes have failed across this control`;
            return (
                <div className="bg-primary-100 w-full" id="capture-dashboard-stretch">
                    <CollapsibleSection title="Control Details">
                        <div className="flex flex-wrap pdf-page">
                            <ControlDetails
                                standardId={standardId}
                                control={name}
                                description={description}
                                className="mx-4 min-w-48 h-48 mb-4"
                            />
                            {!!interpretationText.length && (
                                <Widget
                                    className="mx-4 min-w-48 h-48 mb-4 w-1/3 overflow-auto"
                                    header="Control guidance"
                                >
                                    <div className="p-4 leading-loose whitespace-pre-wrap overflow-auto">
                                        {interpretationText}
                                    </div>
                                </Widget>
                            )}
                            <RelatedEntityListCount
                                className="mx-4 min-w-48 h-48 mb-4"
                                name="Nodes"
                                value={relatedEntities.length}
                                onClick={onRelatedEntityListClickHandler(entityTypes.NODE)}
                            />
                        </div>
                    </CollapsibleSection>
                    <CollapsibleSection title="Control Findings">
                        <div className="flex pdf-page pdf-stretch shadow rounded relative rounded bg-base-100 mb-4 ml-4 mr-4">
                            <TableWidget
                                entityType={entityTypes.NODE}
                                header={tableHeader}
                                rows={failingRelatedEntities}
                                noDataText="No Nodes"
                                className="bg-base-100 w-full"
                                columns={entityAcrossControlsColumns[entityTypes.NODE]}
                                idAttribute="id"
                            />
                        </div>
                    </CollapsibleSection>
                </div>
            );
        }}
    </Query>
);

Control.propTypes = {
    id: PropTypes.string.isRequired,
    onRelatedEntityListClick: PropTypes.func.isRequired
};

export default Control;
