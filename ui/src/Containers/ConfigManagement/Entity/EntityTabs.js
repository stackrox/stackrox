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
    VIOLATIONS_AND_FINDINGS: 'Violations & Findings',
    APPLICATION_RESOURCES: 'Application & Infrastructure Resources',
    RBAC_CONFIG: 'RBAC Visibility & Configurations',
    POLICIES: 'Policies & CIS Controls'
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

function getTab(relationship, entityType) {
    if (entityType === entityTypes.DEPLOYMENT && relationship === entityTypes.POLICY) {
        return {
            group: ENTITY_TO_TAB[relationship],
            value: relationship,
            text: `Failing ${pluralize(entityLabels[entityTypes.POLICY])}`
        };
    }

    return {
        group: ENTITY_TO_TAB[relationship],
        value: relationship,
        text: pluralize(entityLabels[relationship])
    };
}

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

    const relationships = entityTabsMap[entityType];
    if (!relationships) return null;
    const entityTabs = relationships.map(relationship => getTab(relationship, entityType));
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
