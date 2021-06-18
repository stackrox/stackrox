/* eslint-disable react/jsx-no-bind */
/* eslint-disable react/no-array-index-key */
import React, { ReactElement } from 'react';
import {
    Button,
    Card,
    CardActions,
    CardBody,
    CardHeader,
    CardTitle,
    Flex,
    FlexItem,
    Tooltip,
} from '@patternfly/react-core';
import { OutlinedQuestionCircleIcon } from '@patternfly/react-icons';

import { LabelSelector, LabelSelectorRequirement, LabelSelectorsKey } from 'services/RolesService';

import { LabelSelectorCreatable, LabelSelectorUpdatable } from './LabelSelector';

// Form group label icon style rule in AccessScopes.css mimics info prop in table head cells.

const labelIconLabelSelector = (
    <Tooltip
        content={
            <div>
                If a label selector has multiple requirements,
                <br />
                then all of them must be satisfied
            </div>
        }
        isContentLeftAligned
        maxWidth="24rem"
    >
        <div className="pf-c-button pf-m-plain pf-m-smallest pf-u-ml-sm">
            <OutlinedQuestionCircleIcon />
        </div>
    </Tooltip>
);

export type LabelSelectorsProps = {
    labelSelectors: LabelSelector[];
    labelSelectorsKey: LabelSelectorsKey;
    hasAction: boolean;
    handleChangeLabelSelectors: (
        labelSelectorsKey: LabelSelectorsKey,
        labelSelectorsNext: LabelSelector[]
    ) => void;
};

function LabelSelectors({
    labelSelectors,
    labelSelectorsKey,
    hasAction,
    handleChangeLabelSelectors,
}: LabelSelectorsProps): ReactElement {
    const titleLabelSelector =
        labelSelectorsKey === 'namespaceLabelSelectors'
            ? 'Namespace label selector'
            : 'Cluster label selector';

    function handleChangeLabelSelectorRequirements(
        indexLabelSelector: number,
        requirements: LabelSelectorRequirement[]
    ) {
        handleChangeLabelSelectors(
            labelSelectorsKey,
            labelSelectors.map((labelSelector, index) =>
                index === indexLabelSelector ? { requirements } : labelSelector
            )
        );
    }

    function handleDeleteLabelSelector(indexLabelSelector: number) {
        handleChangeLabelSelectors(
            labelSelectorsKey,
            labelSelectors.filter((_, i) => i !== indexLabelSelector)
        );
    }

    return (
        <Flex direction={{ default: 'column' }}>
            {labelSelectors.map((labelSelector, indexLabelSelector) => (
                <FlexItem key={indexLabelSelector} className="pf-u-pt-md">
                    <Card isCompact isFlat>
                        <CardHeader>
                            <CardTitle className="pf-u-font-size-sm">
                                {titleLabelSelector}
                                {labelIconLabelSelector}
                            </CardTitle>
                            {hasAction && (
                                <CardActions>
                                    <Button
                                        variant="danger"
                                        isSmall
                                        onClick={() =>
                                            handleDeleteLabelSelector(indexLabelSelector)
                                        }
                                    >
                                        Delete label selector
                                    </Button>
                                </CardActions>
                            )}
                        </CardHeader>
                        <CardBody>
                            <LabelSelectorUpdatable
                                labelSelectorsKey={labelSelectorsKey}
                                requirements={labelSelector.requirements}
                                hasAction={hasAction}
                                handleChangeRequirements={(requirements) =>
                                    handleChangeLabelSelectorRequirements(
                                        indexLabelSelector,
                                        requirements
                                    )
                                }
                            />
                        </CardBody>
                    </Card>
                </FlexItem>
            ))}
            {hasAction && (
                <FlexItem key={labelSelectors.length} className="pf-u-pt-md">
                    <Card isCompact isFlat>
                        <CardHeader>
                            <CardTitle className="pf-u-font-size-sm">
                                {titleLabelSelector}
                                {labelIconLabelSelector}
                            </CardTitle>
                        </CardHeader>
                        <CardBody>
                            <LabelSelectorCreatable />
                        </CardBody>
                    </Card>
                </FlexItem>
            )}
        </Flex>
    );
}

export default LabelSelectors;
