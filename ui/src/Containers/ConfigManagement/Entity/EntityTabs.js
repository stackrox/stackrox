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
    [entityTypes.ROLE]: [
        {
            group: TAB_GROUPS.RBAC_CONFIG,
            value: entityTypes.SUBJECT,
            text: pluralize(entityLabels[entityTypes.SUBJECT])
        },
        {
            group: TAB_GROUPS.RBAC_CONFIG,
            value: entityTypes.SERVICE_ACCOUNT,
            text: pluralize(entityLabels[entityTypes.SERVICE_ACCOUNT])
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
    ],
    [entityTypes.NODE]: [
        {
            group: TAB_GROUPS.RBAC_CONFIG,
            value: entityTypes.CONTROL,
            text: pluralize(entityLabels[entityTypes.CONTROL])
        }
    ],
    [entityTypes.IMAGE]: [
        {
            group: TAB_GROUPS.APPLICATION_RESOURCES,
            value: entityTypes.DEPLOYMENT,
            text: pluralize(entityLabels[entityTypes.DEPLOYMENT])
        }
    ],
    [entityTypes.CIS_Docker_v1_1_0]: [
        {
            group: TAB_GROUPS.APPLICATION_RESOURCES,
            value: entityTypes.NODE,
            text: pluralize(entityLabels[entityTypes.NODE])
        }
    ],
    [entityTypes.CIS_Kubernetes_v1_2_0]: [
        {
            group: TAB_GROUPS.APPLICATION_RESOURCES,
            value: entityTypes.NODE,
            text: pluralize(entityLabels[entityTypes.NODE])
        }
    ],
    [entityTypes.POLICY]: [],
    [entityTypes.SUBJECT]: [
        {
            group: TAB_GROUPS.RBAC_CONFIG,
            value: entityTypes.ROLE,
            text: pluralize(entityLabels[entityTypes.ROLE])
        }
    ]
};

const EntityTabs = ({ entityType, entityListType, pageEntityId, onClick }) => {
    const [activeTab, setActiveTab] = useState(entityListType);
    useEffect(
        () => {
            setActiveTab(entityListType);
        },
        [entityListType]
    );
    // this is because each standard relates to different resources, so we need to show different tabs
    const getStandardId = controlId => controlId.split(':')[0];
    const key = entityType === entityTypes.CONTROL ? getStandardId(pageEntityId) : entityType;
    const entityTabs = entityTabsMap[key];
    if (!entityTabs) return null;

    const groups = Object.values(TAB_GROUPS);

    const tabs = [{ group: TAB_GROUPS.OVERVIEW, value: null, text: 'Overview' }, ...entityTabs];

    return <GroupedTabs groups={groups} tabs={tabs} activeTab={activeTab} onClick={onClick} />;
};

EntityTabs.propTypes = {
    entityType: PropTypes.string.isRequired,
    entityListType: PropTypes.string,
    pageEntityId: PropTypes.string.isRequired,
    onClick: PropTypes.func.isRequired
};

EntityTabs.defaultProps = {
    entityListType: null
};

export default EntityTabs;
