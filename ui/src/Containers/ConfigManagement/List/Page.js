import React from 'react';
import ReactRouterPropTypes from 'react-router-prop-types';
import entityLabels from 'messages/entity';
import pluralize from 'pluralize';
import URLService from 'modules/URLService2';

import PageHeader from 'Components/PageHeader';
import List from '../EntityList';

const ListPage = ({ match, location, history }) => {
    const params = URLService.getParams(match, location);
    const { pageEntityListType } = params;

    function onRowClick(entityId) {
        const urlBuilder = URLService.getURL(match, location)
            .base(pageEntityListType)
            .push(pageEntityListType, entityId);
        history.push(urlBuilder.url());
    }

    const header = pluralize(entityLabels[pageEntityListType]);

    return (
        <>
            <PageHeader header={header} subHeader="Entity List" />
            <div className="h-full">
                <List entityListType={pageEntityListType} onRowClick={onRowClick} />
            </div>
        </>
    );
};

ListPage.propTypes = {
    match: ReactRouterPropTypes.match.isRequired,
    location: ReactRouterPropTypes.location.isRequired,
    history: ReactRouterPropTypes.history.isRequired
};

export default ListPage;
