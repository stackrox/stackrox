import React from 'react';
import PropTypes from 'prop-types';

import { getEntityTypesByRelationship } from 'modules/entityRelationships';
import relationshipTypes from 'constants/relationshipTypes';
import { generateURLTo } from 'modules/URLReadWrite';
import TileList from 'Components/TileList';
import pluralize from 'pluralize';

const RelatedEntitiesSideList = ({ entityType, workflowState, getCountData }) => {
    const { useCase } = workflowState;

    const matches = getEntityTypesByRelationship(entityType, relationshipTypes.MATCHES, useCase)
        .map(matchEntity => {
            const count = getCountData(matchEntity);
            return {
                count,
                label: pluralize(matchEntity, count),
                url: generateURLTo(workflowState, matchEntity)
            };
        })
        .filter(matchObj => matchObj.count);
    const contains = getEntityTypesByRelationship(entityType, relationshipTypes.CONTAINS, useCase)
        .map(containEntity => {
            const count = getCountData(containEntity);
            return {
                count,
                label: pluralize(containEntity, count),
                url: generateURLTo(workflowState, containEntity)
            };
        })
        .filter(containObj => containObj.count);

    return (
        <div className="bg-primary-300 h-full relative">
            {/* TODO: decide if this should be added as custom tailwind class, or a "component" CSS class in app.css */}
            <h2
                style={{
                    position: 'relative',
                    left: '-0.5rem',
                    width: 'calc(100% + 0.5rem)'
                }}
                className="my-4 p-2 bg-primary-700 text-base text-base-100 rounded-l"
            >
                Related entities
            </h2>
            {!!matches.length && <TileList items={matches} title="Matches" />}
            {!!contains.length && <TileList items={contains} title="Contains" />}
        </div>
    );
};

RelatedEntitiesSideList.propTypes = {
    entityType: PropTypes.string.isRequired,
    workflowState: PropTypes.shape({
        useCase: PropTypes.string.isRequired
    }).isRequired,
    getCountData: PropTypes.func.isRequired
};

export default RelatedEntitiesSideList;
