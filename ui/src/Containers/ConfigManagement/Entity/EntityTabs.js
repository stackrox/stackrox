import React from 'react';
import PropTypes from 'prop-types';
import entityTypes from 'constants/entityTypes';
import entityLabels from 'messages/entity';
import pluralize from 'pluralize';
import URLService from 'modules/URLService';
import ReactRouterPropTypes from 'react-router-prop-types';
import { withRouter } from 'react-router-dom';
import GroupedTabs from 'Components/GroupedTabs';

const TAB_GROUPS = {
    OVERVIEW: 'Overview',
    VIOLATIONS_AND_FINDINGS: 'Violations & Findings',
    APPLICATION_RESOURCES: 'Application & Infrastructure Resources',
    RBAC_CONFIG: 'RBAC Visibility & Configurations',
    POLICIES: 'Policies & CIS Controls'
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
            group: TAB_GROUPS.APPLICATION_RESOURCES,
            value: entityTypes.SECRET,
            text: pluralize(entityLabels[entityTypes.SECRET])
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
        },
        {
            group: TAB_GROUPS.POLICIES,
            value: entityTypes.POLICY,
            text: pluralize(entityLabels[entityTypes.POLICY])
        }
    ],
    [entityTypes.DEPLOYMENT]: [
        {
            group: TAB_GROUPS.APPLICATION_RESOURCES,
            value: entityTypes.IMAGE,
            text: pluralize(entityLabels[entityTypes.IMAGE])
        },
        {
            group: TAB_GROUPS.POLICIES,
            value: entityTypes.POLICY,
            text: `Failing ${pluralize(entityLabels[entityTypes.POLICY])}`
        },
        {
            group: TAB_GROUPS.APPLICATION_RESOURCES,
            value: entityTypes.SECRET,
            text: pluralize(entityLabels[entityTypes.SECRET])
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
        },
        {
            group: TAB_GROUPS.APPLICATION_RESOURCES,
            value: entityTypes.IMAGE,
            text: pluralize(entityLabels[entityTypes.IMAGE])
        },
        {
            group: TAB_GROUPS.POLICIES,
            value: entityTypes.POLICY,
            text: pluralize(entityLabels[entityTypes.POLICY])
        }
    ],
    [entityTypes.NODE]: [
        {
            group: TAB_GROUPS.POLICY,
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
    [entityTypes.POLICY]: [
        {
            group: TAB_GROUPS.APPLICATION_RESOURCES,
            value: entityTypes.DEPLOYMENT,
            text: pluralize(entityLabels[entityTypes.DEPLOYMENT])
        }
    ],
    [entityTypes.SUBJECT]: [
        {
            group: TAB_GROUPS.RBAC_CONFIG,
            value: entityTypes.ROLE,
            text: pluralize(entityLabels[entityTypes.ROLE])
        }
    ]
};

const EntityTabs = ({
    match,
    location,
    entityType,
    entityListType,
    pageEntityId,
    history,
    disabled
}) => {
    function onClick({ value }) {
        if (disabled) return;

        const builder = URLService.getURL(match, location);
        if (value) builder.push(value);
        else builder.base(entityType, pageEntityId);
        history.push(builder.url());
    }

    // this is because each standard relates to different resources, so we need to show different tabs
    const getStandardId = controlId => controlId.split(':')[0];
    const key = entityType === entityTypes.CONTROL ? getStandardId(pageEntityId) : entityType;
    const entityTabs = entityTabsMap[key];
    if (!entityTabs) return null;

    const groups = Object.values(TAB_GROUPS);

    const tabs = [{ group: TAB_GROUPS.OVERVIEW, value: '', text: 'Overview' }, ...entityTabs];
    // TODO: disabled style for tabs
    return (
        <GroupedTabs
            groups={groups}
            tabs={tabs}
            activeTab={entityListType || ''}
            onClick={onClick}
        />
    );
};

EntityTabs.propTypes = {
    entityType: PropTypes.string.isRequired,
    entityListType: PropTypes.string,
    pageEntityId: PropTypes.string.isRequired,
    match: ReactRouterPropTypes.match.isRequired,
    location: ReactRouterPropTypes.location.isRequired,
    history: ReactRouterPropTypes.history.isRequired,
    disabled: PropTypes.bool
};

EntityTabs.defaultProps = {
    entityListType: null,
    disabled: false
};

export default withRouter(EntityTabs);
