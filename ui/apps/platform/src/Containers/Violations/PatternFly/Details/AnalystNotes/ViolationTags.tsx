import React, { ReactElement } from 'react';

import ViolationTagsSearchAutoComplete from 'Containers/AnalystNotes/ViolationTags/ViolationTagsSearchAutoComplete';
import ViolationTagsCard from './ViolationTagsCard';

type ViolationTagsProps = {
    resourceId: string;
};

const ViolationTags = ({ resourceId }: ViolationTagsProps): ReactElement => {
    return (
        <div data-testid="violation-tags">
            <ViolationTagsSearchAutoComplete>
                {({ options, autoCompleteVariables, onInputChange }) => (
                    <ViolationTagsCard
                        resourceId={resourceId}
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
