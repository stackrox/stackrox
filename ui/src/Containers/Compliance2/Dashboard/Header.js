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
            <div className="ml-3 border-l border-base-100 mr-3 opacity-50" />
            <div className="flex">
                <div className="flex items-center">
                    <ScanButton
                        className="flex items-center justify-center border-2 border-primary-400 text-base-100 rounded p-2 uppercase hover:bg-primary-800 lg:min-w-32 xl:min-w-43 h-10"
                        text="Scan environment"
                        textClass="hidden lg:block"
                        textCondensed="Scan all"
                        clusterId="*"
                        standardId="*"
                    />
                </div>
                <div className="flex items-center">
                    <ExportButton
                        className="flex items-center border-2 border-primary-400 text-base-100 rounded p-2 uppercase hover:bg-primary-800 h-10"
                        fileName="compliance-dashboard"
                        textClass="hidden lg:block"
                    />
                </div>
            </div>
        </div>
    </PageHeader>
);

ComplianceDashboardHeader.propTypes = {
    classes: PropTypes.string,
    bgStyle: PropTypes.shape({})
};

ComplianceDashboardHeader.defaultProps = {
    classes: null,
    bgStyle: null
};

export default ComplianceDashboardHeader;
