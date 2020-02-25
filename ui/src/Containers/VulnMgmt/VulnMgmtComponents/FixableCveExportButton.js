import React from 'react';
import { FileText } from 'react-feather';

import Button from 'Components/Button';

function FixableCveExportButton({ clickHandler, disabled }) {
    return (
        <Button
            className="inline-flex px-1 rounded-sm font-600 uppercase text-center items-center min-w-24 justify-center border-2 !important;
    line-height text-base-600 border-base-300 bg-base-100 text-base-600 text-sm my-2 py-2 pr-3"
            disabled={disabled}
            text="Export as CSV"
            textCondensed="Export as CSV"
            icon={<FileText size="14" className="mx-1 lg:ml-1 lg:mr-2" />}
            onClick={clickHandler}
        />
    );
}

export default FixableCveExportButton;
