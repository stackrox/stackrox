import React, { useContext } from 'react';
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
import searchContext from 'Containers/searchContext';
import { entityComponentPropTypes, entityComponentDefaultProps } from 'constants/entityPageProps';
import EntityWithFailedControls, { getRelatedEntities } from './widgets/EntityWithFailedControls';
import EntityList from '../List/EntityList';

const Control = ({ id, entityListType, query }) => {
    const searchParam = useContext(searchContext);

    const variables = {
        id,
        where: queryService.objectToWhereClause(query[searchParam]),
        groupBy: [entityTypes.CONTROL, entityTypes.NODE]
    };

    return (
        <Query query={QUERY} variables={variables}>
            {({ loading, data }) => {
                if (loading) return <Loader />;
                const { results: entity, entities } = data;
                if (!entity) return <PageNotFound resourceType={entityTypes.CONTROL} />;

                const {
                    standardId = '',
                    name = '',
                    description = '',
                    interpretationText = ''
                } = entity;
                const relatedEntities = getRelatedEntities(entities, entityTypes.NODE);

                if (entityListType) {
                    return (
                        <EntityList
                            entityListType={entityListType}
                            data={relatedEntities}
                            query={query}
                        />
                    );
                }

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
                                    entityType={entityTypes.NODE}
                                />
                            </div>
                        </CollapsibleSection>
                        <CollapsibleSection title="Control Findings">
                            <div className="flex pdf-page pdf-stretch shadow rounded relative rounded bg-base-100 mb-4 ml-4 mr-4">
                                <EntityWithFailedControls
                                    entityType={entityTypes.NODE}
                                    entities={entities}
                                />
                            </div>
                        </CollapsibleSection>
                    </div>
                );
            }}
        </Query>
    );
};

Control.propTypes = entityComponentPropTypes;
Control.defaultProps = entityComponentDefaultProps;

export default Control;
