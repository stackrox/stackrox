import React from 'react';
import { FileText } from 'react-feather';

import Button from 'Components/Button';
import exportPDF from 'services/PDFExportService';
import { RequestAction, SuccessAction } from 'utils/fetchingReduxRoutines';
import useBranding from 'hooks/useBranding';

interface PDFExportButtonProps {
    fileName: string;
    pdfId: string;
    startExportingPDF: RequestAction;
    finishExportingPDF: SuccessAction;
}

const PDFExportButton = ({
    fileName,
    pdfId,
    startExportingPDF,
    finishExportingPDF,
}: PDFExportButtonProps) => {
    const branding = useBranding();
    function exportPDFFile() {
        exportPDF(fileName, pdfId, branding, startExportingPDF, finishExportingPDF);
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

export default PDFExportButton;
