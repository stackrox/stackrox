import React from 'react';
import PropTypes from 'prop-types';

import entityTypes, { searchCategories } from 'constants/entityTypes';
import PageHeader from 'Components/PageHeader';
import URLSearchInput from 'Components/URLSearchInput';
import CreatePolicyFromSearch from './CreatePolicyFromSearch';

function RiskPageHeader({ autoFocusSearchInput, isViewFiltered, searchOptions }) {
    const subHeader = isViewFiltered ? 'Filtered view' : 'Default view';
    const autoCompleteCategories = [searchCategories[entityTypes.DEPLOYMENT]];

    return (
        <PageHeader header="Risk" subHeader={subHeader}>
            <URLSearchInput
                className="w-full"
                categoryOptions={searchOptions}
                categories={autoCompleteCategories}
                placeholder="Add one or more resource filters"
                autoFocus={autoFocusSearchInput}
            />
            <CreatePolicyFromSearch />
        </PageHeader>
    );
}

RiskPageHeader.propTypes = {
    autoFocusSearchInput: PropTypes.bool.isRequired,
    isViewFiltered: PropTypes.bool.isRequired,
    searchOptions: PropTypes.arrayOf(PropTypes.string).isRequired,
};

export default RiskPageHeader;
