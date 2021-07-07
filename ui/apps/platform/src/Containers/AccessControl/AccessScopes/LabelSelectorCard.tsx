/* eslint-disable react/jsx-no-bind */
/* eslint-disable react/no-array-index-key */
import React, { ReactElement, useState } from 'react';
import {
    Badge,
    Button,
    Card,
    CardActions,
    CardBody,
    CardHeader,
    CardTitle,
    Flex,
    FlexItem,
    Toolbar,
    ToolbarContent,
    ToolbarGroup,
    ToolbarItem,
    Tooltip,
} from '@patternfly/react-core';
import { OutlinedQuestionCircleIcon, PlusCircleIcon } from '@patternfly/react-icons';
import { TableComposable, Tbody, Th, Thead, Tr } from '@patternfly/react-table';

import {
    LabelSelectorOperator,
    LabelSelectorRequirement,
    LabelSelectorsKey,
    getIsValidRequirements,
} from 'services/RolesService';

import { Activity, getRequirementActivity } from './accessScopes.utils';
import RequirementRow from './RequirementRow';
import RequirementRowAddKey from './RequirementRowAddKey';

const labelIconLabelSelector = (
    <Tooltip
        content={
            <div>
                A label selector card has <strong>requirement</strong> rows
                <br />
                All requirements must be satisfied (r1 and r2 and r3)
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

const infoValues = {
    ariaLabel: 'in: key has one of the values; not in: key does not have any of the values',
    tooltip: (
        <div>
            <strong>in</strong>: key does have one of the values
            <br />
            <strong>not in</strong>: key does not have any of the values
        </div>
    ),
    tooltipProps: {
        isContentLeftAligned: true,
        maxWidth: '24rem',
    },
};

export type LabelSelectorCardProps = {
    requirements: LabelSelectorRequirement[];
    labelSelectorsKey: LabelSelectorsKey;
    hasAction: boolean;
    indexRequirementActive: number;
    setIndexRequirementActive: (indexRequirement: number) => void;
    activity: Activity;
    handleLabelSelectorDelete: () => void;
    handleLabelSelectorEdit: () => void;
    handleLabelSelectorOK: () => void;
    handleLabelSelectorCancel: () => void;
    handleRequirementsChange: (requirements: LabelSelectorRequirement[]) => void;
};

function LabelSelectorCard({
    requirements,
    labelSelectorsKey,
    hasAction,
    indexRequirementActive,
    setIndexRequirementActive,
    activity,
    handleLabelSelectorDelete,
    handleLabelSelectorEdit,
    handleLabelSelectorOK,
    handleLabelSelectorCancel,
    handleRequirementsChange,
}: LabelSelectorCardProps): ReactElement {
    const [requirementsCancel, setRequirementsCancel] = useState<LabelSelectorRequirement[]>([]);
    const [hasAddKey, setHasAddKey] = useState(false);

    const title =
        labelSelectorsKey === 'namespaceLabelSelectors'
            ? 'Namespace label selector'
            : 'Cluster label selector';

    const isLabelSelectorActive = activity === 'ACTIVE';

    function handleRequirementChange(
        indexRequirement: number,
        requirementChange: LabelSelectorRequirement
    ) {
        handleRequirementsChange(
            requirements.map((requirement, index) =>
                index === indexRequirement ? requirementChange : requirement
            )
        );
    }

    function onAddRequirement() {
        // Render an active extra row to enter the key.
        setIndexRequirementActive(requirements.length);
        setHasAddKey(true);
    }

    function handleRequirementKeyOK(key) {
        // Append requirement and render it as the last ordinary row.
        handleRequirementsChange([...requirements, { key, op: 'IN', values: [] }]);

        // The added requirement remains active,
        // just as if editing an existing requirement.
        // Because it has no values yet:
        // its OK button is disabled initially
        // getIsValidRules in AccessScopeForm short-circuits computeeffectiveaccessscopes request
        setRequirementsCancel(requirements);
        setHasAddKey(false);
    }

    function handleRequirementKeyCancel() {
        setIndexRequirementActive(-1);
        setHasAddKey(false);
    }

    function handleRequirementDelete(indexRequirement: number) {
        handleRequirementsChange(requirements.filter((_, index) => index !== indexRequirement));
    }

    function handleRequirementEdit(indexRequirement: number) {
        setRequirementsCancel(requirements);
        setIndexRequirementActive(indexRequirement);
    }

    function handleRequirementOK() {
        setIndexRequirementActive(-1);
    }

    function handleRequirementCancel() {
        handleRequirementsChange(requirementsCancel);
        setRequirementsCancel([]);
        setIndexRequirementActive(-1);
    }

    function handleOperatorSelect(indexRequirement: number, op: LabelSelectorOperator) {
        const { key, values } = requirements[indexRequirement];
        handleRequirementChange(indexRequirement, {
            key,
            op,
            values,
        });
    }

    function handleValueAdd(indexRequirement: number, value: string) {
        const { key, op, values } = requirements[indexRequirement];
        handleRequirementChange(indexRequirement, {
            key,
            op,
            values: [...values, value],
        });
    }

    function handleValueDelete(indexRequirement: number, indexValue: number) {
        const { key, op, values } = requirements[indexRequirement];
        handleRequirementChange(indexRequirement, {
            key,
            op,
            values: values.filter((_, index) => index !== indexValue),
        });
    }

    return (
        <Card isCompact isFlat>
            <CardHeader>
                <CardTitle className="pf-u-font-size-sm">
                    {title}
                    {labelIconLabelSelector}
                </CardTitle>
                {hasAction && (
                    <CardActions>
                        <Button
                            variant="danger"
                            className="pf-m-smaller"
                            isDisabled={activity !== 'ENABLED'}
                            onClick={handleLabelSelectorDelete}
                        >
                            Delete label selector
                        </Button>
                    </CardActions>
                )}
            </CardHeader>
            <CardBody>
                <Flex spaceItems={{ default: 'spaceItemsSm' }} className="pf-u-pb-sm">
                    <FlexItem>
                        <strong>Requirements</strong>
                    </FlexItem>
                    <FlexItem>
                        <Badge isRead>{requirements.length}</Badge>
                    </FlexItem>
                </Flex>
                {(requirements.length !== 0 || hasAddKey) && (
                    <TableComposable variant="compact">
                        <Thead>
                            <Tr>
                                <Th modifier="breakWord">Key</Th>
                                <Th modifier="fitContent">Operator</Th>
                                <Th modifier="breakWord" info={infoValues}>
                                    Values
                                </Th>
                                {isLabelSelectorActive && <Th modifier="fitContent">Action</Th>}
                            </Tr>
                        </Thead>
                        <Tbody
                            className={
                                labelSelectorsKey === 'namespaceLabelSelectors'
                                    ? 'pf-u-background-color-200'
                                    : ''
                            }
                        >
                            {requirements.map((requirement, indexRequirement) => (
                                <RequirementRow
                                    key={`${requirement.key} ${requirement.op}`}
                                    requirement={requirement}
                                    requirements={requirements}
                                    hasAction={isLabelSelectorActive}
                                    activity={getRequirementActivity(
                                        indexRequirement,
                                        indexRequirementActive
                                    )}
                                    handleRequirementDelete={() =>
                                        handleRequirementDelete(indexRequirement)
                                    }
                                    handleRequirementEdit={() =>
                                        handleRequirementEdit(indexRequirement)
                                    }
                                    handleRequirementOK={handleRequirementOK}
                                    handleRequirementCancel={handleRequirementCancel}
                                    handleOperatorSelect={(op: LabelSelectorOperator) =>
                                        handleOperatorSelect(indexRequirement, op)
                                    }
                                    handleValueAdd={(value: string) =>
                                        handleValueAdd(indexRequirement, value)
                                    }
                                    handleValueDelete={(indexValue: number) =>
                                        handleValueDelete(indexRequirement, indexValue)
                                    }
                                />
                            ))}
                            {hasAddKey && (
                                <RequirementRowAddKey
                                    handleRequirementKeyOK={handleRequirementKeyOK}
                                    handleRequirementKeyCancel={handleRequirementKeyCancel}
                                />
                            )}
                        </Tbody>
                    </TableComposable>
                )}
                {hasAction && (
                    <Toolbar className="pf-u-pb-0" inset={{ default: 'insetNone' }}>
                        {isLabelSelectorActive ? (
                            <ToolbarContent>
                                <ToolbarItem>
                                    <Button
                                        key="Add requirement"
                                        variant="link"
                                        isInline
                                        icon={<PlusCircleIcon className="pf-u-mr-sm" />}
                                        onClick={onAddRequirement}
                                        isDisabled={indexRequirementActive !== -1}
                                    >
                                        Add requirement
                                    </Button>
                                </ToolbarItem>
                                <ToolbarGroup alignment={{ default: 'alignRight' }}>
                                    <ToolbarItem>
                                        <Button
                                            variant="primary"
                                            className="pf-m-smaller"
                                            onClick={handleLabelSelectorOK}
                                            isDisabled={
                                                indexRequirementActive !== -1 ||
                                                !getIsValidRequirements(requirements)
                                            }
                                        >
                                            OK
                                        </Button>
                                    </ToolbarItem>
                                    <ToolbarItem>
                                        <Button
                                            variant="tertiary"
                                            className="pf-m-smaller"
                                            onClick={handleLabelSelectorCancel}
                                        >
                                            Cancel
                                        </Button>
                                    </ToolbarItem>
                                </ToolbarGroup>
                            </ToolbarContent>
                        ) : (
                            <ToolbarContent>
                                <ToolbarItem>
                                    <Button
                                        key="Edit label selector"
                                        variant="primary"
                                        className="pf-m-smaller"
                                        isDisabled={activity === 'DISABLED'}
                                        onClick={handleLabelSelectorEdit}
                                    >
                                        Edit label selector
                                    </Button>
                                </ToolbarItem>
                            </ToolbarContent>
                        )}
                    </Toolbar>
                )}
            </CardBody>
        </Card>
    );
}

export default LabelSelectorCard;
