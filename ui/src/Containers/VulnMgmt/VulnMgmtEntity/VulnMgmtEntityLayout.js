import React from 'react';
import ReactRouterPropTypes from 'react-router-prop-types';
import pluralize from 'pluralize';

import PageHeader from 'Components/PageHeader';
import entityLabels from 'messages/entity';
import URLService from 'modules/URLService';

const VulnMgmtListLayout = ({ match, location }) => {
    const params = URLService.getParams(match, location);
    const { pageEntityType, pageEntityId } = params;

    const header = pluralize(entityLabels[pageEntityType]);
    return (
        <div className="flex flex-col relative min-h-full">
            <PageHeader header={header} subHeader={pageEntityId}>
                <div className="flex flex-1 justify-end">
                    <div className="flex">
                        <div className="flex items-center">Tag and Export buttons go here</div>
                        <div className="flex items-center">
                            ALL ENTITIES dropdown thingy goes here
                        </div>
                    </div>
                </div>
            </PageHeader>
            <div className="flex flex-1 h-full bg-base-100 relative z-0">
                <div>
                    <p>
                        Placeholder for Entity Type {pageEntityType} ID {pageEntityId}
                    </p>
                </div>
            </div>
        </div>
    );
};

VulnMgmtListLayout.propTypes = {
    match: ReactRouterPropTypes.match.isRequired,
    location: ReactRouterPropTypes.location.isRequired
};

export default VulnMgmtListLayout;
