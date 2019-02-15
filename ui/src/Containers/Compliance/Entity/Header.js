import React from 'react';
import PropTypes from 'prop-types';
import { withRouter } from 'react-router-dom';

import PageHeader from 'Components/PageHeader';
import ScanButton from 'Containers/Compliance/ScanButton';
import ExportButton from 'Components/ExportButton';
import entityTypes from 'constants/entityTypes';

const EntityHeader = ({
    header,
    subHeader,
    searchComponent,
    scanCluster,
    scanStandard,
    params
}) => (
    <PageHeader header={header} subHeader={subHeader}>
        {searchComponent}
        <div className="flex flex-1 justify-end">
            <div className="flex">
                <div className="flex items-center">
                    <>
                        {params.entityType === entityTypes.CLUSTER && (
                            <ScanButton
                                text="Scan"
                                clusterId={scanCluster}
                                standardId={scanStandard}
                            />
                        )}
                        <ExportButton
                            fileName={header}
                            type={params.entityType === entityTypes.CLUSTER ? 'CLUSTER' : ''}
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
    scanStandard: PropTypes.string,
    params: PropTypes.shape({})
};

EntityHeader.defaultProps = {
    header: '',
    subHeader: '',
    scanCluster: '*',
    scanStandard: '*',
    searchComponent: null,
    params: null
};

export default withRouter(EntityHeader);
