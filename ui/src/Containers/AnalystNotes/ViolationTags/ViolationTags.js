import React from 'react';
import PropTypes from 'prop-types';

import ANALYST_NOTES_TYPES from 'constants/analystnotes';
import AnalystTags from 'Containers/AnalystNotes/AnalystTags';
import ViolationTagsSearchAutoComplete from './ViolationTagsSearchAutoComplete';

const ViolationTags = ({ resourceId, isCollapsible }) => {
    const variables = { resourceId };

    return (
        <ViolationTagsSearchAutoComplete>
            {({ isLoading, options, autoCompleteVariables, onInputChange }) => (
                <AnalystTags
                    type={ANALYST_NOTES_TYPES.VIOLATION}
                    variables={variables}
                    isCollapsible={isCollapsible}
                    autoComplete={options}
                    autoCompleteVariables={autoCompleteVariables}
                    isLoadingAutoComplete={isLoading}
                    onInputChange={onInputChange}
                />
            )}
        </ViolationTagsSearchAutoComplete>
    );
};

ViolationTags.propTypes = {
    resourceId: PropTypes.string.isRequired,
    isCollapsible: PropTypes.bool
};

ViolationTags.defaultProps = {
    isCollapsible: true
};

export default ViolationTags;
