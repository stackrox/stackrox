import React from 'react';
import PropTypes from 'prop-types';
import { withRouter } from 'react-router-dom';

import PageHeader from 'Components/PageHeader';
import ScanButton from 'Containers/Compliance/ScanButton';
import ExportButton from 'Components/ExportButton';
import entityTypes from 'constants/entityTypes';
import pluralize from 'pluralize';
import toLower from 'lodash/toLower';

const EntityHeader = ({
    entityType,
    listEntityType,
    entityName,
    entityId,
    searchComponent,
    headerText
}) => {
    const header = headerText || (entityName || 'Loading...');
    const subHeader = toLower(entityType);
    const exportFilename = listEntityType
        ? `${pluralize(listEntityType)} ACROSS ${entityType} "${entityName.toUpperCase()}"`
        : `${entityType} "${entityId}"`;

    const pdfId = listEntityType ? 'capture-list' : 'capture-dashboard';

    const scanCluster = entityType === entityTypes.CLUSTER ? entityId : null;
    const scanStandard = entityType === entityTypes.STANDARD ? entityId : null;

    return (
        <PageHeader classes="bg-base-100" header={header} subHeader={subHeader}>
            {searchComponent}
            <div className="flex flex-1 justify-end">
                <div className="flex">
                    <div className="flex items-center">
                        <>
                            {scanCluster && (
                                <ScanButton
                                    text="Scan"
                                    clusterId={scanCluster}
                                    standardId={scanStandard}
                                />
                            )}
                            <ExportButton
                                fileName={exportFilename}
                                type={entityType}
                                id={entityId}
                                pdfId={pdfId}
                            />
                        </>
                    </div>
                </div>
            </div>
        </PageHeader>
    );
};

EntityHeader.propTypes = {
    entityType: PropTypes.string,
    listEntityType: PropTypes.string,
    entityName: PropTypes.string,
    entityId: PropTypes.string,
    searchComponent: PropTypes.node,
    headerText: PropTypes.string
};

EntityHeader.defaultProps = {
    entityType: '',
    listEntityType: '',
    entityName: '',
    entityId: '',
    searchComponent: null,
    headerText: null
};

export default withRouter(EntityHeader);
