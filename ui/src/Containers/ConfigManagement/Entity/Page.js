import React from 'react';
import ReactRouterPropTypes from 'react-router-prop-types';
import { withRouter } from 'react-router-dom';
import URLService from 'modules/URLService';

import pluralize from 'pluralize';
import ExportButton from 'Components/ExportButton';
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
    const exportFilename = `${pluralize(pageEntityType)}`;
    const { urlParams } = URLService.getURL(match, location);
    let pdfId = 'capture-dashboard-stretch';
    if (urlParams.entityListType1) {
        pdfId = 'capture-list';
    }

    return (
        <div className="h-full bg-base-200">
            <PageHeader entityType={pageEntityType} entityId={pageEntityId}>
                <div className="flex flex-1 justify-end">
                    <div className="flex">
                        <div className="flex items-center">
                            <ExportButton
                                fileName={exportFilename}
                                type={pageEntityType}
                                page="configManagement"
                                pdfId={pdfId}
                            />
                        </div>
                    </div>
                </div>
            </PageHeader>
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
