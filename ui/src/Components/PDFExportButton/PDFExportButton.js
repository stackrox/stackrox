import React from 'react';
import PropTypes from 'prop-types';
import { FileText } from 'react-feather';

import Button from 'Components/Button';
import exportPDF from 'services/PDFExportService';

const PDFExportButton = ({ fileName, pdfId, startExportingPDF, finishExportingPDF }) => {
    function exportPDFFile() {
        exportPDF(fileName, pdfId, startExportingPDF, finishExportingPDF);
    }
    return (
        <Button
            className="btn btn-base"
            icon={<FileText className="h-4 w-4 mx-2" />}
            text="Export"
            onClick={exportPDFFile}
        />
    );
};

PDFExportButton.propTypes = {
    fileName: PropTypes.string.isRequired,
    pdfId: PropTypes.string.isRequired,
    startExportingPDF: PropTypes.func.isRequired,
    finishExportingPDF: PropTypes.func.isRequired,
};

export default PDFExportButton;
