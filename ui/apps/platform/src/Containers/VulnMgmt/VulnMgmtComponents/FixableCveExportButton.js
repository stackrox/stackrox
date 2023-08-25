import React, { useState } from 'react';
import { FileText } from 'react-feather';

import Button from 'Components/Button';
import { exportCvesAsCsv } from 'services/VulnerabilitiesService';
import { getCveExportName } from 'utils/vulnerabilityUtils';

const FixableCveExportButton = ({ workflowState, entityName, disabled, exportType }) => {
    const [isLoading, setIsLoading] = useState(false);

    function clickHandler() {
        const { useCase } = workflowState;
        const pageEntityType = workflowState.getCurrentEntityType();
        const csvName = getCveExportName(useCase, pageEntityType, entityName);

        const stateWithFixable = workflowState.setSearch({ Fixable: 'true' });
        setIsLoading(true);

        exportCvesAsCsv(csvName, stateWithFixable, exportType).finally(() => {
            setIsLoading(false);
        });
    }

    return (
        <Button
            className="inline-flex px-1 rounded-sm text-center items-center min-w-24 justify-center border-2 !important;
    line-height text-base-600 border-base-300 bg-base-100 text-sm my-2 py-2 pr-3 hover:bg-base-200"
            disabled={disabled}
            text="Export as CSV"
            textCondensed="Export as CSV"
            icon={<FileText size="14" className="mx-1 lg:ml-1 lg:mr-2" />}
            onClick={clickHandler}
            isLoading={isLoading}
        />
    );
};

export default FixableCveExportButton;
