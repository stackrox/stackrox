import React from 'react';
import PropTypes from 'prop-types';
import PageHeader from 'Components/PageHeader';
import Button from 'Components/Button';
import * as Icon from 'react-feather';

import ScanButton from 'Containers/Compliance2/ScanButton';
import ClustersTile from './tiles/ClustersTile';
import NamespacesTile from './tiles/NamespacesTile';
import NodesTile from './tiles/NodesTile';

const handleExport = () => () => {
    throw new Error('"Export" is not supported yet.');
};

const ComplianceDashboardHeader = props => (
    <PageHeader
        classes={props.classes}
        bgStyle={props.bgStyle}
        header="Compliance"
        subHeader="Dashboard"
    >
        <div className="flex flex-1 justify-end">
            <div>
                <ClustersTile />
            </div>
            <div className="ml-3">
                <NamespacesTile />
            </div>
            <div className="ml-3">
                <NodesTile />
            </div>
            <div className="ml-3 border-l border-base-100 mr-3" />
            <div className="flex">
                <div className="flex items-center mr-3">
                    <ScanButton text="Scan All" clusterId="*" standardId="*" />
                </div>
                <div className="flex items-center">
                    <Button
                        className="flex items-center border-2 border-primary-400 text-base-100 rounded p-2 uppercase hover:bg-primary-800"
                        text="Export"
                        icon={<Icon.FileText size="14" className="mr-3" />}
                        onClick={handleExport()}
                    />
                </div>
            </div>
        </div>
    </PageHeader>
);

ComplianceDashboardHeader.propTypes = {
    classes: PropTypes.string,
    bgStyle: PropTypes.string
};

ComplianceDashboardHeader.defaultProps = {
    classes: null,
    bgStyle: null
};

export default ComplianceDashboardHeader;
