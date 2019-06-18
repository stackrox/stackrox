import React from 'react';
import ReactRouterPropTypes from 'react-router-prop-types';
import entityLabels from 'messages/entity';
import pluralize from 'pluralize';
import URLService from 'modules/URLService';

import PageHeader from 'Components/PageHeader';
import ExportButton from 'Components/ExportButton';
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
    const exportFilename = `${pluralize(pageEntityListType)}`;

    return (
        <>
            <PageHeader header={header} subHeader="Entity List">
                <div className="flex flex-1 justify-end">
                    <div className="flex">
                        <div className="flex items-center">
                            <ExportButton
                                fileName={exportFilename}
                                type={pageEntityListType}
                                page="configManagement"
                                pdfId="capture-list"
                            />
                        </div>
                    </div>
                </div>
            </PageHeader>
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
