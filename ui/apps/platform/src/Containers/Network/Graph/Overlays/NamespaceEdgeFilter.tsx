import React, { ReactElement } from 'react';

import RadioButtonGroup from 'Components/RadioButtonGroup';

export type NamespaceEdgeFilterProps = {
    selectedState: 'show' | 'hide';
    setFilter: (string) => void;
};

function NamespaceEdgeFilter({ selectedState, setFilter }: NamespaceEdgeFilterProps): ReactElement {
    const buttons = [
        {
            text: 'Show',
            value: 'show',
        },
        {
            text: 'Hide',
            value: 'hide',
        },
    ];

    return (
        <div className="flex items-center" data-testid="namespace-flows-filter">
            <span className="text-base-500 font-700 mr-2">Namespace flows:</span>
            <div className="flex items-center">
                <RadioButtonGroup
                    buttons={buttons}
                    onClick={setFilter}
                    selected={selectedState}
                    groupClassName="h-auto w-24 my-1"
                />
            </div>
        </div>
    );
}

export default NamespaceEdgeFilter;
