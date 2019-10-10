import React, { useContext } from 'react';
import PropTypes from 'prop-types';
import entityTypes from 'constants/entityTypes';
import entityLabels from 'messages/entity';
import pluralize from 'pluralize';
import GroupedTabs from 'Components/GroupedTabs';
import WorkflowStateManager from 'modules/WorkflowStateManager';
import entityRelationships from 'modules/entityRelationships';
import { generateURL } from 'modules/URLReadWrite';
import workflowStateContext from 'Containers/workflowStateContext';

const TAB_GROUPS = {
    OVERVIEW: 'Overview',
    POLICIES: 'Policies & CIS Controls',
    VIOLATIONS_AND_FINDINGS: 'Violations & Findings',
    APPLICATION_RESOURCES: 'Application & Infrastructure Resources',
    RBAC_CONFIG: 'RBAC Visibility & Configurations',
    SECURITY: 'Security Findings'
};

const ENTITY_TO_TAB = {
    [entityTypes.ROLE]: TAB_GROUPS.RBAC_CONFIG,
    [entityTypes.SUBJECT]: TAB_GROUPS.RBAC_CONFIG,
    [entityTypes.SERVICE_ACCOUNT]: TAB_GROUPS.RBAC_CONFIG,

    [entityTypes.DEPLOYMENT]: TAB_GROUPS.APPLICATION_RESOURCES,
    [entityTypes.SECRET]: TAB_GROUPS.APPLICATION_RESOURCES,
    [entityTypes.NODE]: TAB_GROUPS.APPLICATION_RESOURCES,
    [entityTypes.CLUSTER]: TAB_GROUPS.APPLICATION_RESOURCES,
    [entityTypes.NAMESPACE]: TAB_GROUPS.APPLICATION_RESOURCES,
    [entityTypes.IMAGE]: TAB_GROUPS.APPLICATION_RESOURCES,

    [entityTypes.POLICY]: TAB_GROUPS.POLICIES,
    [entityTypes.CONTROL]: TAB_GROUPS.POLICIES,

    [entityTypes.COMPONENT]: TAB_GROUPS.SECURITY,
    [entityTypes.CVE]: TAB_GROUPS.SECURITY
};

const EntityTabs = ({ entityType, listType }) => {
    const workflowState = useContext(workflowStateContext);

    function getTab(tabType) {
        const failingText =
            entityType === entityTypes.DEPLOYMENT && tabType === entityTypes.POLICY
                ? 'failing '
                : '';
        const newState = new WorkflowStateManager(workflowState).pushList(tabType).workflowState;
        return {
            group: ENTITY_TO_TAB[tabType],
            value: tabType,
            text: `${failingText}${pluralize(entityLabels[tabType])}`,
            to: generateURL(newState)
        };
    }

    const relationships = [
        ...entityRelationships.getContains(entityType),
        ...entityRelationships.getMatches(entityType)
    ];

    if (!relationships) return null;
    const entityTabs = relationships.map(relationship => getTab(relationship, entityType));
    const groups = Object.values(TAB_GROUPS);

    const tabs = [
        { group: TAB_GROUPS.OVERVIEW, value: '', text: 'Overview', to: '.' },
        ...entityTabs
    ];
    return <GroupedTabs groups={groups} tabs={tabs} activeTab={listType || ''} />;
};

EntityTabs.propTypes = {
    entityType: PropTypes.string.isRequired,
    listType: PropTypes.string
};

EntityTabs.defaultProps = {
    listType: null
};

export default EntityTabs;
