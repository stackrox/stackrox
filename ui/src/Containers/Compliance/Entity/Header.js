import React from 'react';
import PropTypes from 'prop-types';
import { withRouter } from 'react-router-dom';

import PageHeader from 'Components/PageHeader';
import ScanButton from 'Containers/Compliance/ScanButton';
import ExportButton from 'Components/ExportButton';

const EntityHeader = ({ header, subHeader, searchComponent, scanCluster, scanStandard }) => (
    <PageHeader header={header} subHeader={subHeader}>
        {searchComponent}
        <div className="flex flex-1 justify-end">
            <div className="flex">
                <div className="flex items-center">
                    <>
                        {scanCluster && (
                            <ScanButton
                                text="Scan"
                                clusterId={scanCluster}
                                standardId={scanStandard}
                            />
                        )}
                        <ExportButton
                            fileName={`${subHeader} "${header}" Compliance`}
                            type={scanCluster ? 'CLUSTER' : ''}
                            id={scanCluster}
                            pdfId="capture-dashboard"
                        />
                    </>
                </div>
            </div>
        </div>
    </PageHeader>
);

EntityHeader.propTypes = {
    header: PropTypes.string,
    subHeader: PropTypes.string,
    searchComponent: PropTypes.node,
    scanCluster: PropTypes.string,
    scanStandard: PropTypes.string
};

EntityHeader.defaultProps = {
    header: '',
    subHeader: '',
    scanCluster: '*',
    scanStandard: '*',
    searchComponent: null
};

export default withRouter(EntityHeader);
