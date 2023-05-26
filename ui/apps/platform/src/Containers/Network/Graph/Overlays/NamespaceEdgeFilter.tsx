import React, { ReactElement } from 'react';

import RadioButtonGroup from 'Components/RadioButtonGroup';

export type NamespaceEdgeFilterState = 'show' | 'hide';

export type NamespaceEdgeFilterProps = {
    selectedState: NamespaceEdgeFilterState;
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
            <span className="text-sm mr-2">Namespace flows:</span>
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
