import React from 'react';
import ReactRouterPropTypes from 'react-router-prop-types';
import PropTypes from 'prop-types';
import { withRouter } from 'react-router-dom';
import URLService from 'modules/URLService';
import { standardTypes } from 'constants/entityTypes';
import standardLabels from 'messages/standards';

import PageHeader from 'Components/PageHeader';
import ScanButton from 'Containers/Compliance2/ScanButton';
import ExportButton from 'Components/ExportButton';

const isStandard = type => Object.values(standardTypes).includes(type);

const ListHeader = ({ match, location, searchComponent }) => {
    const params = URLService.getParams(match, location);
    const { entityType } = params;
    const headerText = isStandard(entityType) ? standardLabels[entityType] : `${entityType}s`;
    const subHeaderText = isStandard(entityType) ? 'Standard' : 'Resource list';

    return (
        <PageHeader header={headerText} subHeader={subHeaderText}>
            <div className="w-full">{searchComponent}</div>
            {isStandard(entityType) && (
                <div className="flex flex-1 justify-end">
                    <div className="border-l-2 border-base-300 mx-3" />
                    <div className="flex">
                        <div className="flex items-center">
                            <div className="flex">
                                <ScanButton text="Scan" standardId={entityType} />
                                <ExportButton
                                    fileName={headerText}
                                    id={params.entityId}
                                    type="STANDARD"
                                />
                            </div>
                        </div>
                    </div>
                </div>
            )}
        </PageHeader>
    );
};
ListHeader.propTypes = {
    searchComponent: PropTypes.element,
    match: ReactRouterPropTypes.match.isRequired,
    location: ReactRouterPropTypes.location.isRequired
};

ListHeader.defaultProps = {
    searchComponent: null
};

export default withRouter(ListHeader);
