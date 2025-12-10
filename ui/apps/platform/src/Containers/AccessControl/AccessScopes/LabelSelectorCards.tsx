/* eslint-disable react/no-array-index-key */
import { useState } from 'react';
import type { ReactElement } from 'react';
import { Button, Label, Toolbar, ToolbarContent, ToolbarItem } from '@patternfly/react-core';

import type {
    LabelSelector,
    LabelSelectorRequirement,
    LabelSelectorsKey,
} from 'services/AccessScopesService';

import { getIsEditingLabelSelectorOnTab, getLabelSelectorActivity } from './accessScopes.utils';
import type { LabelSelectorsEditingState } from './accessScopes.utils';
import LabelSelectorCard from './LabelSelectorCard';

export type LabelSelectorCardsProps = {
    labelSelectors: LabelSelector[];
    labelSelectorsKey: LabelSelectorsKey;
    hasAction: boolean;
    labelSelectorsEditingState: LabelSelectorsEditingState;
    setLabelSelectorsEditingState: (nextState: LabelSelectorsEditingState) => void;
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
        setLabelSelectorsEditingState({
            ...labelSelectorsEditingState,
            [labelSelectorsKey]: indexLabelSelector,
        });
    }

    function handleLabelSelectorOK() {
        setLabelSelectorsCancel([]);
        setIndexRequirementActive(-1);
        setLabelSelectorsEditingState({
            ...labelSelectorsEditingState,
            [labelSelectorsKey]: -1,
        });
    }

    function handleLabelSelectorCancel() {
        handleLabelSelectorsChange(labelSelectorsKey, labelSelectorsCancel);
        setLabelSelectorsCancel([]);
        setIndexRequirementActive(-1);
        setLabelSelectorsEditingState({
            ...labelSelectorsEditingState,
            [labelSelectorsKey]: -1,
        });
    }

    function onAddLabelSelector() {
        setLabelSelectorsCancel(labelSelectors);
        handleLabelSelectorsChange(labelSelectorsKey, [...labelSelectors, { requirements: [] }]);
        setLabelSelectorsEditingState({
            ...labelSelectorsEditingState,
            [labelSelectorsKey]: labelSelectors.length,
        });
    }

    return (
        <ul>
            {labelSelectors.map((labelSelector, indexLabelSelector) => (
                <li key={indexLabelSelector} className="pf-v5-u-pt-md">
                    {indexLabelSelector !== 0 && (
                        <div className="pf-v5-u-mb-md pf-v5-u-text-align-center">
                            <Label variant="outline" className="pf-v5-u-px-md">
                                or
                            </Label>
                        </div>
                    )}
                    <LabelSelectorCard
                        requirements={labelSelector.requirements}
                        labelSelectorsKey={labelSelectorsKey}
                        hasAction={hasAction}
                        indexRequirementActive={indexRequirementActive}
                        setIndexRequirementActive={setIndexRequirementActive}
                        activity={getLabelSelectorActivity(
                            labelSelectorsEditingState,
                            labelSelectorsKey,
                            indexLabelSelector
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
                                    isDisabled={getIsEditingLabelSelectorOnTab(
                                        labelSelectorsEditingState,
                                        labelSelectorsKey
                                    )}
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
