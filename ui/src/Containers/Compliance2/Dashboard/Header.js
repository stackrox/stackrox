import React from 'react';
import PropTypes from 'prop-types';
import PageHeader from 'Components/PageHeader';
import ScanButton from 'Containers/Compliance2/ScanButton';
import ExportButton from 'Components/ExportButton';
import ClustersTile from './tiles/ClustersTile';
import NamespacesTile from './tiles/NamespacesTile';
import NodesTile from './tiles/NodesTile';

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
                    <ExportButton fileName="compliance-dashboard" />
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
