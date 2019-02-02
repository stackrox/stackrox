import React from 'react';
import { withRouter } from 'react-router-dom';
import PropTypes from 'prop-types';
import PageHeader from 'Components/PageHeader';
import Button from 'Components/Button';
import * as Icon from 'react-feather';

import ReactRouterPropTypes from 'react-router-prop-types';
import { resourceTypes, standardTypes } from 'constants/entityTypes';
import URLService from 'modules/URLService';
import labels from 'messages/common';

const handleExport = () => {
    throw new Error('"Export" is not supported yet.');
};

const ListHeader = ({ match, location, searchComponent }) => {
    const headerTexts = {
        [resourceTypes.NODES]: `${labels.resourceLabels.NODE}S`,
        [resourceTypes.NAMESPACES]: `${labels.resourceLabels.NAMESPACE}S`,
        [resourceTypes.CLUSTERS]: `${labels.resourceLabels.CLUSTER}S`,
        [standardTypes.PCI_DSS_3_2]: `${labels.standardLabels.PCI} Standard`,
        [standardTypes.NIST_800_190]: `${labels.standardLabels.NIST} Standard`,
        [standardTypes.HIPAA_164]: `${labels.standardLabels.HIPAA} Standard`,
        [standardTypes.CIS_DOCKER_V1_1_0]: `${labels.standardLabels.CIS_DOCKER} Standard`,
        [standardTypes.CIS_KUBERENETES_V1_2_0]: `${labels.standardLabels.CIS_KUBERNETES} Standard`
    };
    const params = URLService.getParams(match, location);
    const { entityType } = params;

    return (
        <PageHeader header={headerTexts[entityType]} subHeader="Resource List">
            <div className="w-full">{searchComponent}</div>
            <div className="flex flex-1 justify-end">
                <div className="ml-3 border-l border-base-300 mr-3" />
                <div className="flex">
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
};
ListHeader.propTypes = {
    searchComponent: PropTypes.element,
    match: ReactRouterPropTypes.match.isRequired,
    location: ReactRouterPropTypes.location.isRequired
};

ListHeader.defaultProps = {
    searchComponent: null
};

export default withRouter(ListHeader);
