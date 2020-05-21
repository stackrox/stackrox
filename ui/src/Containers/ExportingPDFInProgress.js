import React from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { createStructuredSelector } from 'reselect';
import { selectors } from 'reducers';

import Loader from 'Components/Loader';

const ExportingPDFInProgress = ({ pdfLoadingStatus }) => {
    if (!pdfLoadingStatus) return null;
    return (
        <div className="absolute left-0 top-0 bg-base-100 z-70 mt-20 w-full h-full text-tertiary-800">
            <Loader message="Exporting..." />
        </div>
    );
};

ExportingPDFInProgress.propTypes = {
    pdfLoadingStatus: PropTypes.bool,
};

ExportingPDFInProgress.defaultProps = {
    pdfLoadingStatus: false,
};

const mapStateToProps = createStructuredSelector({
    pdfLoadingStatus: selectors.getPdfLoadingStatus,
});

export default connect(mapStateToProps, null)(ExportingPDFInProgress);
