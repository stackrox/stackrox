import React, { ReactElement, useContext } from 'react';

import GroupedTabs from 'Components/GroupedTabs';
import {
    VulnerabilityManagementEntityType,
    getVulnerabilityManagementEntityTypesByRelationship,
    entityGroups,
    entityGroupMap,
} from 'utils/entityRelationships';
import workflowStateContext from '../../workflowStateContext';
import { entityNounSentenceCasePlural } from '../entitiesForVulnerabilityManagement';

export type EntityTabsProps = {
    entityType: VulnerabilityManagementEntityType;
    activeTab?: VulnerabilityManagementEntityType;
};

function EntityTabs({ entityType, activeTab }: EntityTabsProps): ReactElement {
    const workflowState = useContext(workflowStateContext);
    function getTab(tabType) {
        return {
            group: entityGroups[entityGroupMap[tabType]],
            value: tabType,
            text: entityNounSentenceCasePlural[tabType],
            to: workflowState.pushList(tabType).setSearch('').toUrl(),
        };
    }

    const relationships = [
        ...getVulnerabilityManagementEntityTypesByRelationship(entityType, 'MATCHES'),
        ...getVulnerabilityManagementEntityTypesByRelationship(entityType, 'CONTAINS'),
    ];

    const entityTabs = relationships.map((entityTypeByRelationship) =>
        getTab(entityTypeByRelationship)
    );
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
}

export default EntityTabs;
