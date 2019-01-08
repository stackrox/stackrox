import React from 'react';

import PageHeader from 'Components/PageHeader';
import Button from 'Components/Button';
import * as Icon from 'react-feather';

import ClustersTile from './tiles/ClustersTile';
import NamespacesTile from './tiles/NamespacesTile';
import NodesTile from './tiles/NodesTile';

const handleScanAll = () => () => {
    throw new Error('"Scan All" is not supported yet.');
};

const handleExport = () => () => {
    throw new Error('"Export" is not supported yet.');
};

const ComplianceDashboardHeader = () => (
    <PageHeader header="Compliance" subHeader="Dashboard">
        <div className="flex flex-1 justify-end">
            <div className="">
                <ClustersTile />
            </div>
            <div className="ml-3">
                <NamespacesTile />
            </div>
            <div className="ml-3">
                <NodesTile />
            </div>
            <div className="ml-3 border-l border-base-300 mr-3" />
            <div className="flex">
                <div className="flex items-center mr-3">
                    <Button
                        className="btn btn-base"
                        text="Scan All"
                        icon={<Icon.RefreshCcw className="h-4 w-4 mr-3" />}
                        onClick={handleScanAll()}
                    />
                </div>
                <div className="flex items-center">
                    <Button
                        className="btn btn-base"
                        text="Export"
                        icon={<Icon.FileText className="h-4 w-4 mr-3" />}
                        onClick={handleExport()}
                    />
                </div>
            </div>
        </div>
    </PageHeader>
);

export default ComplianceDashboardHeader;
