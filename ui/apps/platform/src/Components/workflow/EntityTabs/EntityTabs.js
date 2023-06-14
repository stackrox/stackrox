import React, { useContext } from 'react';
import PropTypes from 'prop-types';
import pluralize from 'pluralize';

import entityLabels from 'messages/entity';
import GroupedTabs from 'Components/GroupedTabs';
import {
    getVulnerabilityManagementEntityTypesByRelationship,
    entityGroups,
    entityGroupMap,
} from 'utils/entityRelationships';
import workflowStateContext from 'Containers/workflowStateContext';

const EntityTabs = ({ entityType, activeTab }) => {
    const workflowState = useContext(workflowStateContext);
    function getTab(tabType) {
        return {
            group: entityGroupMap[tabType],
            value: tabType,
            text: pluralize(entityLabels[tabType]),
            to: workflowState.pushList(tabType).setSearch('').toUrl(),
        };
    }

    const relationships = [
        ...getVulnerabilityManagementEntityTypesByRelationship(entityType, 'MATCHES'),
        ...getVulnerabilityManagementEntityTypesByRelationship(entityType, 'CONTAINS'),
    ];

    const entityTabs = relationships.map((relationship) => getTab(relationship, entityType));
    const groups = Object.values(entityGroups);

    const tabs = [
        {
            group: entityGroups.OVERVIEW,
            value: '',
            text: 'Overview',
            to: workflowState.base().setSearch('').toUrl(),
        },
        ...entityTabs,
    ];

    return <GroupedTabs groups={groups} tabs={tabs} activeTab={activeTab || ''} />;
};

EntityTabs.propTypes = {
    entityType: PropTypes.string.isRequired,
    activeTab: PropTypes.string,
};

EntityTabs.defaultProps = {
    activeTab: null,
};

export default EntityTabs;
