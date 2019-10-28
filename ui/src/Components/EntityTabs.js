import React, { useContext } from 'react';
import { uniq } from 'lodash';
import PropTypes from 'prop-types';
import entityTypes from 'constants/entityTypes';
import entityLabels from 'messages/entity';
import pluralize from 'pluralize';

import GroupedTabs from 'Components/GroupedTabs';
import entityRelationships from 'modules/entityRelationships';
import { generateURLTo, generateURL } from 'modules/URLReadWrite';
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

// eslint-disable-next-line
const EntityTabs = ({ entityType, activeTab }) => {
    const workflowState = useContext(workflowStateContext);
    function getTab(tabType) {
        const failingText =
            entityType === entityTypes.DEPLOYMENT && tabType === entityTypes.POLICY
                ? 'failing '
                : '';
        return {
            group: ENTITY_TO_TAB[tabType],
            value: tabType,
            text: `${failingText}${pluralize(entityLabels[tabType])}`,
            to: generateURLTo(workflowState, tabType)
        };
    }

    const relationships = uniq([
        ...entityRelationships.getContains(entityType),
        ...entityRelationships.getMatches(entityType)
    ]);

    // TODO filter tabs by useCase

    if (!relationships) return null;
    const entityTabs = relationships.map(relationship => getTab(relationship, entityType));
    const groups = Object.values(TAB_GROUPS);

    const tabs = [
        {
            group: TAB_GROUPS.OVERVIEW,
            value: '',
            text: 'Overview',
            to: generateURL(workflowState.base())
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
