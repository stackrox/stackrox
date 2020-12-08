import React, { useContext } from 'react';
import PropTypes from 'prop-types';
import pluralize from 'pluralize';

import entityLabels from 'messages/entity';
import GroupedTabs from 'Components/GroupedTabs';
import useFeatureFlagEnabled from 'hooks/useFeatureFlagEnabled';
import { knownBackendFlags } from 'utils/featureFlags';
import {
    getEntityTypesByRelationship,
    entityGroups,
    entityGroupMap,
} from 'utils/entityRelationships';
import relationshipTypes from 'constants/relationshipTypes';
import workflowStateContext from 'Containers/workflowStateContext';

// eslint-disable-next-line
const EntityTabs = ({ entityType, activeTab }) => {
    const workflowState = useContext(workflowStateContext);
    const hostScanningEnabled = useFeatureFlagEnabled(knownBackendFlags.ROX_HOST_SCANNING);
    const featureFlags = {
        [knownBackendFlags.ROX_HOST_SCANNING]: hostScanningEnabled,
    };
    function getTab(tabType) {
        return {
            group: entityGroupMap[tabType],
            value: tabType,
            text: pluralize(entityLabels[tabType]),
            to: workflowState.pushList(tabType).setSearch('').toUrl(),
        };
    }

    const relationships = [
        ...getEntityTypesByRelationship(
            entityType,
            relationshipTypes.MATCHES,
            workflowState.useCase,
            featureFlags
        ),
        ...getEntityTypesByRelationship(
            entityType,
            relationshipTypes.CONTAINS,
            workflowState.useCase,
            featureFlags
        ),
    ];

    if (!relationships) {
        return null;
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
