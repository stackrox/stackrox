import React, { useState } from 'react';
import type { ReactElement } from 'react';
import {
    Badge,
    Button,
    Card,
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
import { Table, Tbody, Th, Thead, Tr } from '@patternfly/react-table';

import type { LabelSelectorRequirement, LabelSelectorsKey } from 'services/AccessScopesService';

import { getIsValidRequirements, getRequirementActivity } from './accessScopes.utils';
import type { Activity } from './accessScopes.utils';
import RequirementRow from './RequirementRow';
import RequirementRowAddKey from './RequirementRowAddKey';

const labelIconClusterLabelSelector = (
    <Tooltip
        content={<div>Provide access to new and existing clusters using label selection rules</div>}
        isContentLeftAligned
        maxWidth="24rem"
    >
        <div className="pf-v5-c-button pf-m-plain pf-m-smallest pf-v5-u-ml-sm">
            <OutlinedQuestionCircleIcon />
        </div>
    </Tooltip>
);

const labelIconNamespaceLabelSelector = (
    <Tooltip
        content={
            <div>Provide access to new and existing namespaces using label selection rules</div>
        }
        isContentLeftAligned
        maxWidth="24rem"
    >
        <div className="pf-v5-c-button pf-m-plain pf-m-smallest pf-v5-u-ml-sm">
            <OutlinedQuestionCircleIcon />
        </div>
    </Tooltip>
);

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
    const labelIconLabelSelector =
        labelSelectorsKey === 'namespaceLabelSelectors'
            ? labelIconNamespaceLabelSelector
            : labelIconClusterLabelSelector;

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
            <CardHeader
                {...(hasAction && {
                    actions: {
                        actions: (
                            <>
                                <Button
                                    variant="danger"
                                    className="pf-m-smaller"
                                    isDisabled={activity !== 'ENABLED'}
                                    onClick={handleLabelSelectorDelete}
                                >
                                    Delete label selector
                                </Button>
                            </>
                        ),
                        hasNoOffset: false,
                        className: undefined,
                    },
                })}
            >
                <CardTitle className="pf-v5-u-font-size-sm">
                    {title}
                    {labelIconLabelSelector}
                </CardTitle>
            </CardHeader>
            <CardBody>
                <Flex spaceItems={{ default: 'spaceItemsSm' }} className="pf-v5-u-pb-sm">
                    <FlexItem>
                        <strong>Rules</strong>
                    </FlexItem>
                    <FlexItem>
                        <Badge isRead>{requirements.length}</Badge>
                    </FlexItem>
                </Flex>
                {(requirements.length !== 0 || hasAddKey) && (
                    <Table variant="compact">
                        <Thead>
                            <Tr>
                                <Th width={40}>Key</Th>
                                <Th>
                                    <span className="pf-v5-screen-reader">Operator</span>
                                </Th>
                                <Th width={40}>Values</Th>
                                {isLabelSelectorActive && <Th modifier="fitContent">Action</Th>}
                            </Tr>
                        </Thead>
                        <Tbody
                            className={
                                labelSelectorsKey === 'namespaceLabelSelectors'
                                    ? 'pf-v5-u-background-color-200'
                                    : ''
                            }
                        >
                            {requirements.map((requirement, indexRequirement) => (
                                <RequirementRow
                                    key={`${requirement.key} ${requirement.op}`}
                                    requirement={requirement}
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
                    </Table>
                )}
                {hasAction && (
                    <Toolbar className="pf-v5-u-pb-0" inset={{ default: 'insetNone' }}>
                        {isLabelSelectorActive ? (
                            <ToolbarContent>
                                <ToolbarItem>
                                    <Button
                                        key="Add rule"
                                        variant="link"
                                        isInline
                                        icon={<PlusCircleIcon className="pf-v5-u-mr-sm" />}
                                        onClick={onAddRequirement}
                                        isDisabled={indexRequirementActive !== -1}
                                    >
                                        Add rule
                                    </Button>
                                </ToolbarItem>
                                <ToolbarGroup align={{ default: 'alignRight' }}>
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
