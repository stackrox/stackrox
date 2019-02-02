import React from 'react';
import { withRouter } from 'react-router-dom';
import PropTypes from 'prop-types';
import PageHeader from 'Components/PageHeader';
import Button from 'Components/Button';
import * as Icon from 'react-feather';

import ReactRouterPropTypes from 'react-router-prop-types';
import URLService from 'modules/URLService';

const handleExport = () => {
    throw new Error('"Export" is not supported yet.');
};

const ListHeader = ({ match, location, searchComponent }) => {
    const params = URLService.getParams(match, location);
    const { entityType } = params;

    const headerText = entityType;

    return (
        <PageHeader header={headerText} subHeader="Resource List">
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
