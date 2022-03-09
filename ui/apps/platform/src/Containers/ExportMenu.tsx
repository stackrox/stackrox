import React from 'react';
import { FileText, List } from 'react-feather';
import { connect } from 'react-redux';

import { actions } from 'reducers/pdfDownload';
import exportPDF from 'services/PDFExportService';
import downloadCSV from 'services/CSVDownloadService';
import Menu, { MenuOption } from 'Components/Menu';
import { RequestAction, SuccessAction } from 'utils/fetchingReduxRoutines';

type ExportMenuProps = {
    fileName: string;
    pdfId?: string;
    csvEndpoint?: string;
    csvQueryString?: string;
    startExportingPDF: RequestAction;
    finishExportingPDF: SuccessAction;
};

const ExportMenu = ({
    fileName,
    pdfId,
    csvEndpoint,
    csvQueryString = '',
    startExportingPDF,
    finishExportingPDF,
}: ExportMenuProps) => {
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

const mapDispatchToProps = {
    startExportingPDF: actions.fetchPdf.request,
    finishExportingPDF: actions.fetchPdf.success,
};

export default connect(null, mapDispatchToProps)(ExportMenu);
