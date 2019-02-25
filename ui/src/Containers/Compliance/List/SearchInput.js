import React from 'react';
import PropTypes from 'prop-types';
import { CLIENT_SIDE_SEARCH_OPTIONS as SEARCH_OPTIONS } from 'constants/searchOptions';

import Query from 'Components/ThrowingQuery';
import URLSearchInput from 'Components/URLSearchInput';

import SEARCH_OPTIONS_QUERY from 'queries/search';

const addComplianceStateOption = searchOptions => {
    let modifiedSearchOptions = [];
    if (searchOptions) {
        modifiedSearchOptions = [...searchOptions];
        modifiedSearchOptions.push(SEARCH_OPTIONS.COMPLIANCE.STATE);
    }
    return modifiedSearchOptions;
};

const ComplianceListSearchInput = ({ categories }) => (
    <Query query={SEARCH_OPTIONS_QUERY} action="list" variables={{ categories }}>
        {({ data }) => {
            let { searchOptions } = data;
            searchOptions = addComplianceStateOption(searchOptions);
            return (
                <URLSearchInput
                    className="w-full"
                    categoryOptions={searchOptions}
                    categories={categories}
                />
            );
        }}
    </Query>
);

ComplianceListSearchInput.propTypes = {
    categories: PropTypes.arrayOf(PropTypes.string)
};

ComplianceListSearchInput.defaultProps = {
    categories: []
};

export default ComplianceListSearchInput;
