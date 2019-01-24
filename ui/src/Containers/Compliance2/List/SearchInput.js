import React from 'react';
import PropTypes from 'prop-types';

import Query from 'Components/ThrowingQuery';
import URLSearchInput from 'Components/URLSearchInput';

import SEARCH_OPTIONS_QUERY from 'queries/search';

const ComplianceListSearchInput = ({ categories }) => (
    <Query query={SEARCH_OPTIONS_QUERY} action="list" variables={{ categories }}>
        {({ data }) => {
            const { searchOptions } = data;
            return <URLSearchInput className="w-full" categoryOptions={searchOptions} />;
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
