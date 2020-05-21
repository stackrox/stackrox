import React from 'react';
import PropTypes from 'prop-types';
import { FileText, List } from 'react-feather';
import { connect } from 'react-redux';

import { actions } from 'reducers/pdfDownload';
import exportPDF from 'services/PDFExportService';
import downloadCSV from 'services/CSVDownloadService';
import Menu from 'Components/Menu';

const ExportMenu = ({
    fileName,
    pdfId,
    csvEndpoint,
    csvEndpointParams,
    startExportingPDF,
    finishExportingPDF,
}) => {
    const options = [];
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
                downloadCSV(fileName, csvEndpoint, csvEndpointParams);
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

ExportMenu.propTypes = {
    fileName: PropTypes.string.isRequired,
    pdfId: PropTypes.string,
    csvEndpoint: PropTypes.string,
    csvEndpointParams: PropTypes.shape({}),
    startExportingPDF: PropTypes.func.isRequired,
    finishExportingPDF: PropTypes.func.isRequired,
};

ExportMenu.defaultProps = {
    pdfId: null,
    csvEndpoint: null,
    csvEndpointParams: {},
};

const mapDispatchToProps = {
    startExportingPDF: actions.fetchPdf.request,
    finishExportingPDF: actions.fetchPdf.success,
};

export default connect(null, mapDispatchToProps)(ExportMenu);
