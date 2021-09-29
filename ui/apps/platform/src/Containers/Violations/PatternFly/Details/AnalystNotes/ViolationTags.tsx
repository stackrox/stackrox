import React, { ReactElement } from 'react';

import ViolationTagsSearchAutoComplete from 'Containers/AnalystNotes/ViolationTags/ViolationTagsSearchAutoComplete';
import AnalystTags from 'Containers/AnalystNotes/AnalystTags';
import ANALYST_NOTES_TYPES from 'constants/analystnotes';

type ViolationTagsProps = {
    resourceId: string;
};

const ViolationTags = ({ resourceId }: ViolationTagsProps): ReactElement => {
    return (
        <div data-testid="violation-tags">
            <ViolationTagsSearchAutoComplete>
                {({ options, autoCompleteVariables, onInputChange }) => (
                    <AnalystTags
                        type={ANALYST_NOTES_TYPES.VIOLATION}
                        variables={{ resourceId }}
                        autoComplete={options}
                        autoCompleteVariables={autoCompleteVariables}
                        onInputChange={onInputChange}
                    />
                )}
            </ViolationTagsSearchAutoComplete>
        </div>
    );
};

export default ViolationTags;
