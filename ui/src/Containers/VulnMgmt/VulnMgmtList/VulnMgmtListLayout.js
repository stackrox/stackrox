import React from 'react';
import ReactRouterPropTypes from 'react-router-prop-types';
import pluralize from 'pluralize';

import PageHeader from 'Components/PageHeader';
import entityLabels from 'messages/entity';
import URLService from 'modules/URLService';

import VulnMgmtEntityList from './VulnMgmtEntityList';

const VulnMgmtListLayout = ({ match, location }) => {
    const params = URLService.getParams(match, location);
    const { pageEntityListType, entityId1 } = params;

    const header = pluralize(entityLabels[pageEntityListType]);
    return (
        <div className="flex flex-col relative min-h-full">
            <PageHeader header={header} subHeader="Entity List">
                <div className="flex flex-1 justify-end">
                    <div className="flex">
                        <div className="flex items-center">Tag and Export buttons go here</div>
                    </div>
                </div>
            </PageHeader>
            <div className="flex flex-1 h-full bg-base-100 relative z-0">
                <VulnMgmtEntityList
                    wrapperClass={`bg-base-100 ${entityId1 ? 'overlay' : ''}`}
                    entityListType={pageEntityListType}
                    entityId={entityId1}
                />
            </div>
        </div>
    );
};

VulnMgmtListLayout.propTypes = {
    match: ReactRouterPropTypes.match.isRequired,
    location: ReactRouterPropTypes.location.isRequired
};

export default VulnMgmtListLayout;
