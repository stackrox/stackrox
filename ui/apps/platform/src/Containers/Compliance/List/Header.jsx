import React from 'react';
import PropTypes from 'prop-types';
import lowerCase from 'lodash/lowerCase';
import startCase from 'lodash/startCase';

import PageHeader from 'Components/PageHeader';
import ExportButton from 'Components/ExportButton';
import ScanButton from 'Containers/Compliance/ScanButton';
import { standardLabels } from 'messages/standards';
import useCaseTypes from 'constants/useCaseTypes';
import usePermissions from 'hooks/usePermissions';

// Disable to prevent merge conflict below if we need to cherry pick fix to previous release.
/* eslint-disable prettier/prettier */
const ListHeader = ({ entityType, searchComponent, standard, isExporting, setIsExporting }) => {
    const { hasReadWriteAccess } = usePermissions();
    const hasWriteAccessForCompliance = hasReadWriteAccess('Compliance');

    // ROX-25189: Comment out, because standardLabels prevents Download Evidence as CSV for compliance operator standards.
    // const standardId = findKey(standardLabels, (key) => key === standard);
    const standardId = standard;

    // ROX-25189: Add nullish coalescing to include compliance operator standard id:
    // in page header
    // in file name
    const headerText = standardId
        ? standardLabels[standardId] ?? standardId
        : `${startCase(lowerCase(entityType))}s`;

    // If standardId is truthy, then standard page entity is CONTROL for address controls?s[standard]=WHAT_EVER&s[groupBy]=CATEGORY
    const subHeaderText = standardId ? 'Standard' : 'Resource list';
    let tableOptions = null;
    if (standardId) {
        tableOptions = {
            columnStyles: {
                0: { columnWidth: 80 },
                1: { columnWidth: 80 },
                2: { columnWidth: 25 },
            },
        };
    }
    return (
        <PageHeader header={headerText} subHeader={subHeaderText}>
            <div className="w-full">{searchComponent}</div>
            <div className="flex flex-1 items-center justify-end pl-4">
                {hasWriteAccessForCompliance && standardId && (
                    <ScanButton text="Scan" standardId={standardId} />
                )}
                <ExportButton
                    fileName={`${headerText} Compliance Report`}
                    id={standardId || entityType}
                    type={standardId ? 'STANDARD' : ''}
                    page={useCaseTypes.COMPLIANCE}
                    pdfId="capture-list"
                    tableOptions={tableOptions}
                    isExporting={isExporting}
                    setIsExporting={setIsExporting}
                />
            </div>
        </PageHeader>
    );
};
/* eslint-enable prettier/prettier */
ListHeader.propTypes = {
    searchComponent: PropTypes.element,
    entityType: PropTypes.string.isRequired,
    standard: PropTypes.string,
    isExporting: PropTypes.bool.isRequired,
    setIsExporting: PropTypes.func.isRequired,
};

ListHeader.defaultProps = {
    searchComponent: null,
    standard: null,
};

export default ListHeader;
