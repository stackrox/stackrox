import React from 'react';
import PropTypes from 'prop-types';
import pluralize from 'pluralize';

import PageHeader from 'Components/PageHeader';
import ScanButton from 'Containers/Compliance/ScanButton';
import ExportButton from 'Components/ExportButton';
import useCaseTypes from 'constants/useCaseTypes';
import entityTypes from 'constants/entityTypes';

import { entityNounSentenceCaseSingular } from '../entitiesForCompliance';

const EntityHeader = ({
    entityType,
    listEntityType,
    entityName,
    entityId,
    searchComponent,
    isExporting,
    setIsExporting,
}) => {
    const header = entityName || 'Loading...';
    const subHeader = entityNounSentenceCaseSingular[entityType];
    // Leave raw entity types in case customer depends on this convention for export file name.
    let exportFilename = listEntityType
        ? `${pluralize(listEntityType)} ACROSS ${entityType} "${entityName.toUpperCase()}"`
        : `${entityType} "${entityId}"`;
    exportFilename = `${exportFilename} Report`;
    const pdfId = listEntityType ? 'capture-list' : 'capture-dashboard';

    const scanCluster = entityType === entityTypes.CLUSTER ? entityId : '*';
    const scanStandard = entityType === entityTypes.STANDARD ? entityId : '*';

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
                                page={useCaseTypes.COMPLIANCE}
                                id={entityId}
                                pdfId={pdfId}
                                isExporting={isExporting}
                                setIsExporting={setIsExporting}
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
    isExporting: PropTypes.bool.isRequired,
    setIsExporting: PropTypes.func.isRequired,
};

EntityHeader.defaultProps = {
    entityType: '',
    listEntityType: '',
    entityName: '',
    entityId: '',
    searchComponent: null,
};

export default EntityHeader;
