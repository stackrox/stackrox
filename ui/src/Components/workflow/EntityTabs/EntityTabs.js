import React, { useContext } from 'react';
import PropTypes from 'prop-types';
import entityTypes from 'constants/entityTypes';
import entityLabels from 'messages/entity';
import pluralize from 'pluralize';

import GroupedTabs from 'Components/GroupedTabs';
import {
    getEntityTypesByRelationship,
    entityGroups,
    entityGroupMap
} from 'modules/entityRelationships';
import relationshipTypes from 'constants/relationshipTypes';
import workflowStateContext from 'Containers/workflowStateContext';

// eslint-disable-next-line
const EntityTabs = ({ entityType, activeTab }) => {
    const workflowState = useContext(workflowStateContext);
    function getTab(tabType) {
        const failingText =
            entityType === entityTypes.DEPLOYMENT && tabType === entityTypes.POLICY
                ? 'failing '
                : '';
        return {
            group: entityGroupMap[tabType],
            value: tabType,
            text: `${failingText}${pluralize(entityLabels[tabType])}`,
            to: workflowState.pushList(tabType).toUrl()
        };
    }

    const relationships = [
        ...getEntityTypesByRelationship(
            entityType,
            relationshipTypes.MATCHES,
            workflowState.useCase
        ),
        ...getEntityTypesByRelationship(
            entityType,
            relationshipTypes.CONTAINS,
            workflowState.useCase
        )
    ];

    if (!relationships) return null;
    const entityTabs = relationships.map(relationship => getTab(relationship, entityType));
    const groups = Object.values(entityGroups);

    const tabs = [
        {
            group: entityGroups.OVERVIEW,
            value: '',
            text: 'Overview',
            to: workflowState.base().toUrl()
        },
        ...entityTabs
    ];

    return <GroupedTabs groups={groups} tabs={tabs} activeTab={activeTab || ''} />;
};

EntityTabs.propTypes = {
    entityType: PropTypes.string.isRequired,
    activeTab: PropTypes.string
};

EntityTabs.defaultProps = {
    activeTab: null
};

export default EntityTabs;
