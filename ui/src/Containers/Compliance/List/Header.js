import React from 'react';
import ReactRouterPropTypes from 'react-router-prop-types';
import PropTypes from 'prop-types';
import { withRouter } from 'react-router-dom';
import URLService from 'modules/URLService';
import { standardLabels } from 'messages/standards';
import lowerCase from 'lodash/lowerCase';
import startCase from 'lodash/startCase';
import { standardTypes } from 'constants/entityTypes';
import PageHeader from 'Components/PageHeader';
import ScanButton from 'Containers/Compliance/ScanButton';
import ExportButton from 'Components/ExportButton';

const ListHeader = ({ match, location, searchComponent }) => {
    const params = URLService.getParams(match, location);
    const { entityType } = params;
    const standardId = standardTypes[entityType];
    const headerText = standardId
        ? standardLabels[standardId]
        : `${startCase(lowerCase(entityType))}s`;

    const subHeaderText = standardId ? 'Standard' : 'Resource list';
    let tableOptions = null;
    if (standardId) {
        tableOptions = {
            columnStyles: {
                0: { columnWidth: 80 },
                1: { columnWidth: 80 },
                2: { columnWidth: 25 }
            }
        };
    }

    return (
        <PageHeader header={headerText} subHeader={subHeaderText}>
            <div className="w-full">{searchComponent}</div>
            <div className="flex flex-1 justify-end">
                <div className="border-l-2 border-base-300 mx-3" />
                <div className="flex">
                    <div className="flex items-center">
                        <div className="flex">
                            {standardId && <ScanButton text="Scan" standardId={standardId} />}
                            <ExportButton
                                fileName={`${headerText} Compliance`}
                                id={entityType || standardId}
                                type={standardId ? 'STANDARD' : ''}
                                pdfId="capture-list"
                                tableOptions={tableOptions}
                            />
                        </div>
                    </div>
                </div>
            </div>
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
