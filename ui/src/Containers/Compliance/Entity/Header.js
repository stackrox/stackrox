import React from 'react';
import PropTypes from 'prop-types';
import { withRouter } from 'react-router-dom';

import PageHeader from 'Components/PageHeader';
import ScanButton from 'Containers/Compliance/ScanButton';
import ExportButton from 'Components/ExportButton';
import entityTypes from 'constants/entityTypes';
import pluralize from 'pluralize';

const EntityHeader = ({ entityType, listEntityType, entity, searchComponent, headerText }) => {
    const header = headerText || (entity.name ? entity.name : 'Loading...');
    const subHeader = entityType;
    const exportFilename = listEntityType
        ? `${pluralize(listEntityType)} ACROSS ${entityType} "${entity.name.toUpperCase()}"`
        : `${entityType} "${entity.id}"`;
    const pdfId = listEntityType ? 'capture-list' : 'capture-dashboard';

    const scanCluster = entityType === entityTypes.CLUSTER ? entity.id : null;
    const scanStandard = entityType === entityTypes.STANDARD ? entity.id : null;

    return (
        <PageHeader header={header} subHeader={subHeader}>
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
                                id={entity.id}
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
    entity: PropTypes.shape({
        name: PropTypes.string,
        id: PropTypes.string
    }),
    searchComponent: PropTypes.node,
    headerText: PropTypes.string
};

EntityHeader.defaultProps = {
    entityType: '',
    listEntityType: '',
    entity: {
        name: '',
        id: ''
    },
    searchComponent: null,
    headerText: null
};

export default withRouter(EntityHeader);
