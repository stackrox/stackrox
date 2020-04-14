import React from 'react';
import PropTypes from 'prop-types';

import entityTypes, { searchCategories } from 'constants/entityTypes';
import PageHeader from 'Components/PageHeader';
import URLSearchInput from 'Components/URLSearchInput';

function RiskPageHeader({ isViewFiltered, searchOptions }) {
    const subHeader = isViewFiltered ? 'Filtered view' : 'Default view';
    const autoCompleteCategories = [searchCategories[entityTypes.DEPLOYMENT]];

    return (
        <PageHeader header="Risk" subHeader={subHeader}>
            <URLSearchInput
                className="w-full"
                categoryOptions={searchOptions}
                categories={autoCompleteCategories}
                placeholder="Add one or more resource filters"
                autoFocusSearchInput
            />
        </PageHeader>
    );
}

RiskPageHeader.propTypes = {
    isViewFiltered: PropTypes.bool.isRequired,
    searchOptions: PropTypes.arrayOf(PropTypes.object).isRequired
};

export default RiskPageHeader;
