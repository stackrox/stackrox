/* eslint-disable react/jsx-no-bind */
/* eslint-disable react/no-array-index-key */
import React, { ReactElement, useState } from 'react';
import { Button, Toolbar, ToolbarContent, ToolbarItem } from '@patternfly/react-core';

import { LabelSelector, LabelSelectorRequirement, LabelSelectorsKey } from 'services/RolesService';

import { LabelSelectorsEditingState, getLabelSelectorActivity } from './accessScopes.utils';
import LabelSelectorCard from './LabelSelectorCard';

export type LabelSelectorCardsProps = {
    labelSelectors: LabelSelector[];
    labelSelectorsKey: LabelSelectorsKey;
    hasAction: boolean;
    labelSelectorsEditingState: LabelSelectorsEditingState | null;
    setLabelSelectorsEditingState: (nextState: LabelSelectorsEditingState | null) => void;
    handleLabelSelectorsChange: (
        labelSelectorsKey: LabelSelectorsKey,
        labelSelectorsNext: LabelSelector[]
    ) => void;
};

function LabelSelectorCards({
    labelSelectors,
    labelSelectorsKey,
    hasAction,
    labelSelectorsEditingState,
    setLabelSelectorsEditingState,
    handleLabelSelectorsChange,
}: LabelSelectorCardsProps): ReactElement {
    const [labelSelectorsCancel, setLabelSelectorsCancel] = useState<LabelSelector[]>([]);
    const [indexRequirementActive, setIndexRequirementActive] = useState(-1);

    function handleRequirementsChange(
        indexLabelSelector: number,
        requirements: LabelSelectorRequirement[]
    ) {
        handleLabelSelectorsChange(
            labelSelectorsKey,
            labelSelectors.map((labelSelector, index) =>
                index === indexLabelSelector ? { requirements } : labelSelector
            )
        );
    }

    function handleLabelSelectorDelete(indexLabelSelector: number) {
        handleLabelSelectorsChange(
            labelSelectorsKey,
            labelSelectors.filter((_, i) => i !== indexLabelSelector)
        );
    }

    function handleLabelSelectorEdit(indexLabelSelector: number) {
        setLabelSelectorsCancel(labelSelectors);
        setLabelSelectorsEditingState({ labelSelectorsKey, indexLabelSelector });
    }

    function handleLabelSelectorOK() {
        setLabelSelectorsCancel([]);
        setIndexRequirementActive(-1);
        setLabelSelectorsEditingState(null);
    }

    function handleLabelSelectorCancel() {
        handleLabelSelectorsChange(labelSelectorsKey, labelSelectorsCancel);
        setLabelSelectorsCancel([]);
        setIndexRequirementActive(-1);
        setLabelSelectorsEditingState(null);
    }

    function onAddLabelSelector() {
        setLabelSelectorsCancel(labelSelectors);
        handleLabelSelectorsChange(labelSelectorsKey, [...labelSelectors, { requirements: [] }]);
        setLabelSelectorsEditingState({
            labelSelectorsKey,
            indexLabelSelector: labelSelectors.length,
        });
    }

    return (
        <ul>
            {labelSelectors.map((labelSelector, indexLabelSelector) => (
                <li key={indexLabelSelector} className="pf-u-pt-md">
                    <LabelSelectorCard
                        requirements={labelSelector.requirements}
                        labelSelectorsKey={labelSelectorsKey}
                        hasAction={hasAction}
                        indexRequirementActive={indexRequirementActive}
                        setIndexRequirementActive={setIndexRequirementActive}
                        activity={getLabelSelectorActivity(
                            labelSelectorsKey,
                            indexLabelSelector,
                            labelSelectorsEditingState
                        )}
                        handleLabelSelectorDelete={() =>
                            handleLabelSelectorDelete(indexLabelSelector)
                        }
                        handleLabelSelectorEdit={() => handleLabelSelectorEdit(indexLabelSelector)}
                        handleLabelSelectorOK={handleLabelSelectorOK}
                        handleLabelSelectorCancel={handleLabelSelectorCancel}
                        handleRequirementsChange={(requirements) =>
                            handleRequirementsChange(indexLabelSelector, requirements)
                        }
                    />
                </li>
            ))}
            {hasAction && (
                <li>
                    <Toolbar inset={{ default: 'insetNone' }}>
                        <ToolbarContent>
                            <ToolbarItem>
                                <Button
                                    variant="primary"
                                    className="pf-m-smaller"
                                    isDisabled={Boolean(labelSelectorsEditingState)}
                                    onClick={onAddLabelSelector}
                                >
                                    Add label selector
                                </Button>
                            </ToolbarItem>
                        </ToolbarContent>
                    </Toolbar>
                </li>
            )}
        </ul>
    );
}

export default LabelSelectorCards;
