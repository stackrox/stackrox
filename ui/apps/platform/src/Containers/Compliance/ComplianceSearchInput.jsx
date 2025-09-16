import React from 'react';
import PropTypes from 'prop-types';
import { CLIENT_SIDE_SEARCH_OPTIONS as SEARCH_OPTIONS } from 'constants/searchOptions';

import Query from 'Components/CacheFirstQuery';
import URLSearchInput from 'Components/URLSearchInput';

import { SEARCH_OPTIONS_QUERY } from 'queries/search';

const addComplianceStateOption = (searchOptions) => {
    let modifiedSearchOptions = [];
    if (searchOptions) {
        modifiedSearchOptions = [...searchOptions];
        modifiedSearchOptions.push(SEARCH_OPTIONS.COMPLIANCE.STATE);
    }
    return modifiedSearchOptions;
};

const ComplianceSearchInput = ({ placeholder, categories, shouldAddComplianceState }) => (
    <Query query={SEARCH_OPTIONS_QUERY} action="list" variables={{ categories }}>
        {({ data }) => {
            if (!data) {
                return null;
            }
            let { searchOptions } = data;
            if (shouldAddComplianceState) {
                searchOptions = addComplianceStateOption(searchOptions);
            }
            return (
                <URLSearchInput
                    placeholder={placeholder}
                    className="w-full"
                    categoryOptions={searchOptions}
                    categories={categories}
                />
            );
        }}
    </Query>
);

ComplianceSearchInput.propTypes = {
    placeholder: PropTypes.string.isRequired,
    categories: PropTypes.arrayOf(PropTypes.string),
    shouldAddComplianceState: PropTypes.bool,
};

ComplianceSearchInput.defaultProps = {
    categories: [],
    shouldAddComplianceState: false,
};

export default ComplianceSearchInput;
