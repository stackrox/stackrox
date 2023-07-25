import React from 'react';
import PropTypes from 'prop-types';
import { withRouter } from 'react-router-dom';
import lowerCase from 'lodash/lowerCase';
import startCase from 'lodash/startCase';
import findKey from 'lodash/findKey';

import PageHeader from 'Components/PageHeader';
import ExportButton from 'Components/ExportButton';
import ScanButton from 'Containers/Compliance/ScanButton';
import { standardLabels } from 'messages/standards';
import useCaseTypes from 'constants/useCaseTypes';

const ListHeader = ({ entityType, searchComponent, standard, isExporting, setIsExporting }) => {
    const standardId = findKey(standardLabels, (key) => key === standard);

    const headerText = standardId
        ? standardLabels[standardId]
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
            <div className="flex flex-1 justify-end">
                <div className="border-l-2 border-base-300 mx-3" />
                <div className="flex">
                    <div className="flex items-center">
                        <div className="flex">
                            {standardId && <ScanButton text="Scan" standardId={standardId} />}
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
                    </div>
                </div>
            </div>
        </PageHeader>
    );
};
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

export default withRouter(ListHeader);
