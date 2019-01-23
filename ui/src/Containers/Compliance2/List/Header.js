import React from 'react';
import { withRouter } from 'react-router-dom';
import PropTypes from 'prop-types';
import PageHeader from 'Components/PageHeader';
import Button from 'Components/Button';
import * as Icon from 'react-feather';

import ReactRouterPropTypes from 'react-router-prop-types';
import { resourceTypes } from 'constants/entityTypes';
import URLService from 'modules/URLService';
import labels from 'messages/common';

const handleExport = () => {
    throw new Error('"Export" is not supported yet.');
};

const ListHeader = ({ match, location, searchComponent }) => {
    const headerTexts = {
        [resourceTypes.NODES]: `${labels.resourceLabels.NODE}S`,
        [resourceTypes.NAMESPACES]: `${labels.resourceLabels.NAMESPACE}S`,
        [resourceTypes.CLUSTERS]: `${labels.resourceLabels.CLUSTER}S`
    };
    const params = new URLService(match, location).getParams();
    const { entityType } = params;

    return (
        <PageHeader header={headerTexts[entityType]} subHeader="Resource List">
            {searchComponent}
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
