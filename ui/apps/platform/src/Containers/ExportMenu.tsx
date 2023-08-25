import React from 'react';
import { FileText, List } from 'react-feather';

import exportPDF from 'services/PDFExportService';
import downloadCSV from 'services/CSVDownloadService';
import Menu, { MenuOption } from 'Components/Menu';

type ExportMenuProps = {
    fileName: string;
    pdfId?: string;
    csvEndpoint?: string;
    csvQueryString?: string;
};

const ExportMenu = ({ fileName, pdfId, csvEndpoint, csvQueryString = '' }: ExportMenuProps) => {
    /* Hide change from Redux action creator functions to useState functions.
     * Because its only use in event timeline modal does not seem to need a double backdrop,
     * omit setIsExporting arg and comment out calls.
     * Too bad, so sad: MenuOption does not support isDisabled or isLoading properties
     */
    const startExportingPDF = () => {
        // setIsExporting(true);
        return { type: '', params: undefined };
    };
    const finishExportingPDF = () => {
        // setIsExporting(false);
        return { type: '', response: undefined, params: undefined };
    };

    const options: MenuOption[] = [];
    if (pdfId) {
        options.push({
            className: '',
            icon: <FileText className="h-4 w-4 text-base-600" />,
            label: 'Download PDF',
            onClick: () => {
                exportPDF(fileName, pdfId, startExportingPDF, finishExportingPDF);
            },
        });
    }
    if (csvEndpoint) {
        options.push({
            className: '',
            icon: <List className="h-4 w-4 text-base-600" />,
            label: 'Download CSV',
            onClick: () => {
                return downloadCSV(fileName, csvEndpoint, csvQueryString);
            },
        });
    }
    return (
        <Menu
            className="h-full min-w-30"
            menuClassName="bg-base-100 min-w-28"
            buttonClass="btn btn-base"
            buttonText="Export"
            options={options}
            disabled={false}
            dataTestId="export-menu"
        />
    );
};

export default ExportMenu;
