import React from 'react';
import ReactRouterPropTypes from 'react-router-prop-types';
import { withRouter } from 'react-router-dom';
import URLService from 'modules/URLService2';

import PageHeader from './EntityPageHeader';
import Tabs from './EntityTabs';
import Overview from '../Entity';
import List from '../EntityList';

const SidePanel = () => null;

const EntityPage = ({ match, location, history }) => {
    const params = URLService.getParams(match, location);
    const { pageEntityType, pageEntityId, entityListType1 } = params;

    function onTabClick({ value }) {
        const urlBuilder = URLService.getURL(match, location).base(pageEntityType, pageEntityId);
        const url = value !== null ? urlBuilder.push(value).url() : urlBuilder.url();
        history.push(url);
    }

    function onRowClick(entityId) {
        const urlBuilder = URLService.getURL(match, location)
            .base(pageEntityType, pageEntityId)
            .push(entityListType1)
            .push(entityListType1, entityId);
        history.push(urlBuilder.url());
    }

    function onRelatedEntityClick(entityType, entityId) {
        const urlBuilder = URLService.getURL(match, location).base(entityType, entityId);
        history.push(urlBuilder.url());
    }

    function onRelatedEntityListClick(entityListType) {
        const urlBuilder = URLService.getURL(match, location).push(entityListType);
        history.push(urlBuilder.url());
    }

    const component = !entityListType1 ? (
        <Overview
            entityType={pageEntityType}
            entityId={pageEntityId}
            onRelatedEntityClick={onRelatedEntityClick}
            onRelatedEntityListClick={onRelatedEntityListClick}
        />
    ) : (
        <div className="flex h-full bg-base-200">
            <List entityListType={entityListType1} onRowClick={onRowClick} />
            <SidePanel />
        </div>
    );

    return (
        <div className="h-full bg-base-200">
            <PageHeader entityType={pageEntityType} entityId={pageEntityId} />
            <Tabs
                entityType={pageEntityType}
                entityListType={entityListType1}
                onClick={onTabClick}
            />
            {component}
        </div>
    );
};

EntityPage.propTypes = {
    match: ReactRouterPropTypes.match.isRequired,
    location: ReactRouterPropTypes.location.isRequired,
    history: ReactRouterPropTypes.history.isRequired
};

export default withRouter(EntityPage);
