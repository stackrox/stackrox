import React from 'react';
import ReactRouterPropTypes from 'react-router-prop-types';
import { withRouter } from 'react-router-dom';

import { SEARCH_CATEGORIES } from 'constants/searchOptions';

import PageHeader from 'Components/PageHeader';
import Button from 'Components/Button';
import * as Icon from 'react-feather';
import SearchInput from './SearchInput';

const handleExport = () => {
    throw new Error('"Export" is not supported yet.');
};

const ComplianceListHeader = ({ match }) => {
    const type = match.params.entityType;
    return (
        <PageHeader header={type} subHeader="Resource list">
            <div className="flex flex-1 justify-end">
                <SearchInput categories={[SEARCH_CATEGORIES.DEPLOYMENTS]} />
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

ComplianceListHeader.propTypes = {
    match: ReactRouterPropTypes.match.isRequired
};

export default withRouter(ComplianceListHeader);
