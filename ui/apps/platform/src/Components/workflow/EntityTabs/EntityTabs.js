import React, { useContext } from 'react';
import PropTypes from 'prop-types';
import pluralize from 'pluralize';

import entityLabels from 'messages/entity';
import GroupedTabs from 'Components/GroupedTabs';
import {
    getEntityTypesByRelationship,
    entityGroups,
    entityGroupMap,
} from 'utils/entityRelationships';
import relationshipTypes from 'constants/relationshipTypes';
import workflowStateContext from 'Containers/workflowStateContext';
import useFeatureFlags from 'hooks/useFeatureFlags';
import filterEntityRelationship from 'Containers/VulnMgmt/VulnMgmt.utils/filterEntityRelationship';

const EntityTabs = ({ entityType, activeTab }) => {
    const { isFeatureFlagEnabled } = useFeatureFlags();
    const showVMUpdates = isFeatureFlagEnabled('ROX_FRONTEND_VM_UPDATES');

    const workflowState = useContext(workflowStateContext);
    function getTab(tabType) {
        return {
            group: entityGroupMap[tabType],
            value: tabType,
            text: pluralize(entityLabels[tabType]),
            to: workflowState.pushList(tabType).setSearch('').toUrl(),
        };
    }

    let relationships = [
        ...getEntityTypesByRelationship(
            entityType,
            relationshipTypes.MATCHES,
            workflowState.useCase
        ),
        ...getEntityTypesByRelationship(
            entityType,
            relationshipTypes.CONTAINS,
            workflowState.useCase
        ),
    ]
        // @TODO: Remove the following filter step once ROX_FRONTEND_VM_UPDATES is ON
        .filter((match) => {
            return filterEntityRelationship(showVMUpdates, match);
        });

    if (!relationships) {
        return null;
    }

    if (showVMUpdates && entityType === 'NODE') {
        relationships = relationships.map((relatedEntityType) => {
            if (relatedEntityType === 'COMPONENT') {
                return 'NODE_COMPONENT';
            }
            if (relatedEntityType === 'CVE') {
                return 'NODE_CVE';
            }
            return relatedEntityType;
        });
    }
    if (showVMUpdates && entityType === 'NODE_COMPONENT') {
        relationships = relationships.map((relatedEntityType) => {
            if (relatedEntityType === 'CVE') {
                return 'NODE_CVE';
            }
            return relatedEntityType;
        });
    }
    if (showVMUpdates && entityType === 'IMAGE_COMPONENT') {
        relationships = relationships.map((relatedEntityType) => {
            if (relatedEntityType === 'CVE') {
                return 'IMAGE_CVE';
            }
            return relatedEntityType;
        });
    }
    if (showVMUpdates && entityType === 'IMAGE') {
        relationships = relationships.map((relatedEntityType) => {
            if (relatedEntityType === 'COMPONENT') {
                return 'IMAGE_COMPONENT';
            }
            if (relatedEntityType === 'CVE') {
                return 'IMAGE_CVE';
            }
            return relatedEntityType;
        });
    }

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
