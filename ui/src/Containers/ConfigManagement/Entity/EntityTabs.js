import React, { useState, useEffect } from 'react';
import PropTypes from 'prop-types';
import entityTypes from 'constants/entityTypes';
import entityLabels from 'messages/entity';
import pluralize from 'pluralize';

import GroupedTabs from 'Components/GroupedTabs';

const TAB_GROUPS = {
    OVERVIEW: 'Overview',
    VIOLATIONS_AND_FINDINGS: 'Violations & Findings',
    APPLICATION_RESOURCES: 'Application & Infrastructure Resources',
    RBAC_CONFIG: 'RBAC Visibility & Configurations'
};

const entityTabsMap = {
    [entityTypes.SERVICE_ACCOUNT]: [
        {
            group: TAB_GROUPS.APPLICATION_RESOURCES,
            value: entityTypes.DEPLOYMENT,
            text: pluralize(entityLabels[entityTypes.DEPLOYMENT])
        },
        {
            group: TAB_GROUPS.APPLICATION_RESOURCES,
            value: entityTypes.SECRET,
            text: pluralize(entityLabels[entityTypes.SECRET])
        },
        {
            group: TAB_GROUPS.RBAC_CONFIG,
            value: entityTypes.ROLE,
            text: pluralize(entityLabels[entityTypes.ROLE])
        }
    ],
    [entityTypes.SECRET]: [
        {
            group: TAB_GROUPS.APPLICATION_RESOURCES,
            value: entityTypes.DEPLOYMENT,
            text: pluralize(entityLabels[entityTypes.DEPLOYMENT])
        }
    ],
    [entityTypes.CLUSTER]: [
        {
            group: TAB_GROUPS.APPLICATION_RESOURCES,
            value: entityTypes.NODE,
            text: pluralize(entityLabels[entityTypes.NODE])
        },
        {
            group: TAB_GROUPS.APPLICATION_RESOURCES,
            value: entityTypes.NAMESPACE,
            text: pluralize(entityLabels[entityTypes.NAMESPACE])
        },
        {
            group: TAB_GROUPS.APPLICATION_RESOURCES,
            value: entityTypes.DEPLOYMENT,
            text: pluralize(entityLabels[entityTypes.DEPLOYMENT])
        },
        {
            group: TAB_GROUPS.RBAC_CONFIG,
            value: entityTypes.SUBJECT,
            text: pluralize(entityLabels[entityTypes.SUBJECT])
        },
        {
            group: TAB_GROUPS.RBAC_CONFIG,
            value: entityTypes.SERVICE_ACCOUNT,
            text: pluralize(entityLabels[entityTypes.SERVICE_ACCOUNT])
        },
        {
            group: TAB_GROUPS.RBAC_CONFIG,
            value: entityTypes.ROLE,
            text: pluralize(entityLabels[entityTypes.ROLE])
        }
    ],
    [entityTypes.NAMESPACE]: [
        {
            group: TAB_GROUPS.APPLICATION_RESOURCES,
            value: entityTypes.DEPLOYMENT,
            text: pluralize(entityLabels[entityTypes.DEPLOYMENT])
        },
        {
            group: TAB_GROUPS.APPLICATION_RESOURCES,
            value: entityTypes.SECRET,
            text: pluralize(entityLabels[entityTypes.SECRET])
        }
    ]
};

const EntityTabs = ({ entityType, entityListType, onClick }) => {
    const [activeTab, setActiveTab] = useState(entityListType);
    useEffect(
        () => {
            setActiveTab(entityListType);
        },
        [entityListType]
    );

    const entityTabs = entityTabsMap[entityType];
    if (!entityTabs) return null;

    const groups = Object.values(TAB_GROUPS);

    const tabs = [{ group: TAB_GROUPS.OVERVIEW, value: null, text: 'Overview' }, ...entityTabs];

    return <GroupedTabs groups={groups} tabs={tabs} activeTab={activeTab} onClick={onClick} />;
};

EntityTabs.propTypes = {
    entityType: PropTypes.string.isRequired,
    entityListType: PropTypes.string,
    onClick: PropTypes.func.isRequired
};

EntityTabs.defaultProps = {
    entityListType: null
};

export default EntityTabs;
