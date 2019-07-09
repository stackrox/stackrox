import React from 'react';
import PropTypes from 'prop-types';
import { CONTROL_QUERY as QUERY } from 'queries/controls';
import entityTypes from 'constants/entityTypes';
import queryService from 'modules/queryService';

import Query from 'Components/ThrowingQuery';
import Loader from 'Components/Loader';
import PageNotFound from 'Components/PageNotFound';
import CollapsibleSection from 'Components/CollapsibleSection';
import ControlDetails from 'Components/ControlDetails';
import RelatedEntityListCount from 'Containers/ConfigManagement/Entity/widgets/RelatedEntityListCount';
import Widget from 'Components/Widget';
import EntityWithFailedControls, { getRelatedEntities } from './widgets/EntityWithFailedControls';

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

const Control = ({ id, onRelatedEntityListClick, onRelatedEntityClick }) => (
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
                            <EntityWithFailedControls
                                entityType={entityTypes.NODE}
                                entities={entities}
                                onRelatedEntityClick={onRelatedEntityClick}
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
    onRelatedEntityListClick: PropTypes.func.isRequired,
    onRelatedEntityClick: PropTypes.func.isRequired
};

export default Control;
