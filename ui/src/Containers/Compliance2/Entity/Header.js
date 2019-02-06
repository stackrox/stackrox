import React from 'react';
import PropTypes from 'prop-types';
import { withRouter } from 'react-router-dom';

import PageHeader from 'Components/PageHeader';
import Button from 'Components/Button';
import ScanButton from 'Containers/Compliance2/ScanButton';
import * as Icon from 'react-feather';

const handleExport = () => {
    throw new Error('"Export" is not supported yet.');
};

const EntityHeader = ({ header, subHeader, searchComponent, scanCluster, scanStandard }) => (
    <PageHeader header={header} subHeader={subHeader}>
        {searchComponent}
        <div className="flex flex-1 justify-end">
            <div className="ml-3 border-l border-base-300 mr-3" />
            <div className="flex">
                <div className="flex items-center mr-3">
                    <ScanButton text="Scan" clusterId={scanCluster} standardId={scanStandard} />
                </div>
                <div className="flex items-center">
                    <Button
                        className="btn btn-base"
                        text="Export"
                        icon={<Icon.FileText className="h-4 w-4 mr-3" />}
                        onClick={handleExport}
                    />
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
