import React from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';

import { actions } from 'reducers/pdfDownload';
import PDFExportButton from 'Components/PDFExportButton';

const ReduxPDFExportButton = ({ fileName, pdfId, startExportingPDF, finishExportingPDF }) => {
    return (
        <PDFExportButton
            fileName={fileName}
            pdfId={pdfId}
            startExportingPDF={startExportingPDF}
            finishExportingPDF={finishExportingPDF}
        />
    );
};

ReduxPDFExportButton.propTypes = {
    fileName: PropTypes.string.isRequired,
    pdfId: PropTypes.string.isRequired,
    startExportingPDF: PropTypes.func.isRequired,
    finishExportingPDF: PropTypes.func.isRequired,
};

const mapDispatchToProps = {
    startExportingPDF: actions.fetchPdf.request,
    finishExportingPDF: actions.fetchPdf.success,
};

export default connect(null, mapDispatchToProps)(ReduxPDFExportButton);
