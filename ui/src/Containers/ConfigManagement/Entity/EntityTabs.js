import React from 'react';
import PropTypes from 'prop-types';
import entityTypes from 'constants/entityTypes';
import entityLabels from 'messages/entity';
import pluralize from 'pluralize';
import URLService from 'modules/URLService';
import ReactRouterPropTypes from 'react-router-prop-types';
import { withRouter } from 'react-router-dom';
import GroupedTabs from 'Components/GroupedTabs';
import entityTabsMap from '../entityTabRelationships';

const TAB_GROUPS = {
    OVERVIEW: 'Overview',
    POLICIES: 'Policies & CIS Controls',
    VIOLATIONS_AND_FINDINGS: 'Violations & Findings',
    APPLICATION_RESOURCES: 'Application & Infrastructure Resources',
    RBAC_CONFIG: 'RBAC Visibility & Configurations'
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
    [entityTypes.CONTROL]: TAB_GROUPS.POLICIES
};

const EntityTabs = ({ match, location, entityType, entityListType, pageEntityId }) => {
    function getTab(relationship) {
        const failingText =
            entityType === entityTypes.DEPLOYMENT && relationship === entityTypes.POLICY
                ? 'failing '
                : '';
        return {
            group: ENTITY_TO_TAB[relationship],
            value: relationship,
            text: `${failingText}${pluralize(entityLabels[relationship])}`,
            to: URLService.getURL(match, location)
                .base(entityType, pageEntityId)
                .push(relationship)
                .url()
        };
    }

    const relationships = entityTabsMap[entityType];
    if (!relationships) return null;
    const entityTabs = relationships.map(relationship => getTab(relationship, entityType));
    const groups = Object.values(TAB_GROUPS);

    const tabs = [
        { group: TAB_GROUPS.OVERVIEW, value: '', text: 'Overview', to: '.' },
        ...entityTabs
    ];
    return <GroupedTabs groups={groups} tabs={tabs} activeTab={entityListType || ''} />;
};

EntityTabs.propTypes = {
    entityType: PropTypes.string.isRequired,
    entityListType: PropTypes.string,
    pageEntityId: PropTypes.string.isRequired,
    match: ReactRouterPropTypes.match.isRequired,
    location: ReactRouterPropTypes.location.isRequired
};

EntityTabs.defaultProps = {
    entityListType: null
};

export default withRouter(EntityTabs);
