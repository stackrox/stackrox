import React, { ReactElement, useState, useEffect } from 'react';
import { Alert, Button, Flex } from '@patternfly/react-core';
import { TrashIcon } from '@patternfly/react-icons';
import { useFormikContext } from 'formik';

import { fetchMitreAttackVectors } from 'services/MitreService';
import { MitreAttackVector } from 'types/mitre.proto';
import { Policy } from 'types/policy.proto';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';

import MitreTacticSelect from './MitreTacticSelect';
import MitreTechniqueSelect from './MitreTechniqueSelect';
import {
    addPolicyAttackVector,
    addPolicyTechnique,
    deletePolicyAttackVector,
    deletePolicyTechnique,
    getMitreAttackVector,
    getMitreTechnique,
    replacePolicyAttackVector,
    replacePolicyTechnique,
    sortMitreAttackVectors,
} from './mitreAttackVectors.utils';

/*
 * The application-specific layout is specified in style rules,
 * instead of combining the effects of PatternFly props and style rules.
 * Except layout of id select and delete button is via className prop.
 */
import './MitreAttackVectorsFormSection.css';

const fieldKey = 'mitreAttackVectors';

function MitreAttackVectorsFormSection(): ReactElement {
    // PolicyMitreAttackVector: tactic and techniques are ids.
    const { setFieldValue, values } = useFormikContext<Policy>();
    const { mitreAttackVectors: policyAttackVectors } = values;

    // MitreAttackVector: tactic and techniques are objects.
    const [mitreAttackVectors, setMitreAttackVectors] = useState<MitreAttackVector[]>([]);
    const [mitreAttackVectorsError, setMitreAttackVectorsError] = useState('');

    useEffect(() => {
        fetchMitreAttackVectors()
            .then((data) => {
                setMitreAttackVectors(sortMitreAttackVectors(data));
            })
            .catch((error) => {
                setMitreAttackVectorsError(getAxiosErrorMessage(error));
            });

        return () => {
            setMitreAttackVectors([]);
            setMitreAttackVectorsError('');
        };
    }, []);

    // Prevent multiple selections of a tactic.
    function getPolicyHasTactic(optionId: string) {
        return policyAttackVectors.some(({ tactic: tacticId }) => tacticId === optionId);
    }

    function handleAddTactic(tacticId: string) {
        setFieldValue(fieldKey, addPolicyAttackVector(policyAttackVectors, tacticId));
    }

    function handleDeleteTactic(tacticId: string) {
        setFieldValue(fieldKey, deletePolicyAttackVector(policyAttackVectors, tacticId));
    }

    function handleReplaceTactic(tacticIdPrev: string, tacticIdNext: string) {
        setFieldValue(
            fieldKey,
            replacePolicyAttackVector(policyAttackVectors, tacticIdPrev, tacticIdNext)
        );
    }

    function handleAddTechnique(tacticId: string, techniqueId: string) {
        setFieldValue(fieldKey, addPolicyTechnique(policyAttackVectors, tacticId, techniqueId));
    }

    function handleDeleteTechnique(tacticId: string, techniqueId: string) {
        setFieldValue(fieldKey, deletePolicyTechnique(policyAttackVectors, tacticId, techniqueId));
    }

    function handleReplaceTechnique(
        tacticId: string,
        techniqueIdPrev: string,
        techniqueIdNext: string
    ) {
        setFieldValue(
            fieldKey,
            replacePolicyTechnique(policyAttackVectors, tacticId, techniqueIdPrev, techniqueIdNext)
        );
    }

    /*
     * Render HTML elements with PatternFly TreeView classes, especially because of Delete button:
     * Avoid accessibility problem: Delete button within a TreeView node button,
     * for collapse/expand which seems like unnecessary additional clicks in the form.
     * Align Select element and Delete button, which has vertical-align center as an action prop,
     * therefore alignment is indeterminate at the right of the description paragraph.
     */
    return (
        <div
            id="mitre-attack-vectors-form-section"
            className="pf-c-tree-view pf-m-compact pf-m-no-background"
        >
            {mitreAttackVectorsError && (
                <Alert className="pf-u-my-md" title="Request failed" variant="warning" isInline>
                    {mitreAttackVectorsError}
                </Alert>
            )}
            <ul className="pf-c-tree-view__list mitre-tactics-list">
                {policyAttackVectors.map(({ tactic: tacticId, techniques: techniqueIds }) => {
                    const {
                        tactic: { description: tacticDescription },
                        techniques: mitreTechniques,
                    } = getMitreAttackVector(mitreAttackVectors, tacticId);

                    // Prevent multiple selections of a technique for a tactic.
                    function getPolicyTacticHasTechnique(optionId: string) {
                        return techniqueIds.some((techniqueId) => techniqueId === optionId);
                    }

                    /*
                     * Cannot replace a tactic which has techniques,
                     * because the techniques selected for the previous tactic
                     * might not be relevant for the next tactic.
                     * Instead, delete the tactic, and then add a different tactic.
                     */
                    const isDisabledTactic = techniqueIds.length !== 0;

                    return (
                        <li key={tacticId} className="pf-c-tree-view__list-item mitre-tactic-item">
                            <TreeViewContent>
                                <Flex flexWrap={{ default: 'nowrap' }}>
                                    <MitreTacticSelect
                                        className="pf-u-flex-grow-1 pf-u-flex-shrink-1"
                                        getIsDisabledOption={getPolicyHasTactic}
                                        handleSelectOption={(tacticIdNext) => {
                                            handleReplaceTactic(tacticId, tacticIdNext);
                                        }}
                                        isDisabled={isDisabledTactic}
                                        label="Replace tactic"
                                        mitreAttackVectors={mitreAttackVectors}
                                        tacticId={tacticId}
                                    />
                                    <Button
                                        aria-label="Delete tactic"
                                        className="pf-u-flex-shrink-0"
                                        onClick={() => handleDeleteTactic(tacticId)}
                                        variant="plain"
                                    >
                                        <TrashIcon />
                                    </Button>
                                </Flex>
                                <p className="description">{tacticDescription}</p>
                                <ul className="pf-c-tree-view__list mitre-techniques-list">
                                    {techniqueIds.map((techniqueId) => {
                                        const { description: techniqueDescription } =
                                            getMitreTechnique(mitreTechniques, techniqueId);

                                        return (
                                            <li
                                                key={techniqueId}
                                                className="pf-c-tree-view__list-item mitre-technique-item"
                                            >
                                                <TreeViewContent>
                                                    <Flex flexWrap={{ default: 'nowrap' }}>
                                                        <MitreTechniqueSelect
                                                            className="pf-u-flex-grow-1 pf-u-flex-shrink-1"
                                                            getIsDisabledOption={
                                                                getPolicyTacticHasTechnique
                                                            }
                                                            handleSelectOption={(
                                                                techniqueIdNext
                                                            ) => {
                                                                handleReplaceTechnique(
                                                                    tacticId,
                                                                    techniqueId,
                                                                    techniqueIdNext
                                                                );
                                                            }}
                                                            label="Replace technique"
                                                            mitreTechniques={mitreTechniques}
                                                            techniqueId={techniqueId}
                                                        />
                                                        <Button
                                                            aria-label="Delete technique"
                                                            className="pf-u-flex-shrink-0"
                                                            onClick={() => {
                                                                handleDeleteTechnique(
                                                                    tacticId,
                                                                    techniqueId
                                                                );
                                                            }}
                                                            variant="plain"
                                                        >
                                                            <TrashIcon />
                                                        </Button>
                                                    </Flex>
                                                    <p className="description">
                                                        {techniqueDescription}
                                                    </p>
                                                </TreeViewContent>
                                            </li>
                                        );
                                    })}
                                    <li
                                        key="Add technique"
                                        className="pf-c-tree-view__list-item mitre-technique-item"
                                    >
                                        <TreeViewContent>
                                            <Flex flexWrap={{ default: 'nowrap' }}>
                                                <MitreTechniqueSelect
                                                    className="pf-u-flex-grow-1 pf-u-flex-shrink-1"
                                                    getIsDisabledOption={
                                                        getPolicyTacticHasTechnique
                                                    }
                                                    handleSelectOption={(techniqueId) => {
                                                        handleAddTechnique(tacticId, techniqueId);
                                                    }}
                                                    label="Add technique"
                                                    mitreTechniques={mitreTechniques}
                                                    techniqueId=""
                                                />
                                            </Flex>
                                        </TreeViewContent>
                                    </li>
                                </ul>
                            </TreeViewContent>
                        </li>
                    );
                })}
                <li key="Add tactic" className="pf-c-tree-view__list-item mitre-tactic-item">
                    <TreeViewContent>
                        <Flex flexWrap={{ default: 'nowrap' }}>
                            <MitreTacticSelect
                                className="pf-u-flex-grow-1 pf-u-flex-shrink-1"
                                getIsDisabledOption={getPolicyHasTactic}
                                handleSelectOption={(tacticId) => {
                                    handleAddTactic(tacticId);
                                }}
                                isDisabled={false}
                                label="Add tactic"
                                mitreAttackVectors={mitreAttackVectors}
                                tacticId=""
                            />
                        </Flex>
                    </TreeViewContent>
                </li>
            </ul>
        </div>
    );
}

function TreeViewContent({ children }) {
    return (
        <div className="pf-c-tree-view__content">
            <div className="pf-c-tree-view__node">
                <div className="pf-c-tree-view__node-container">
                    <div className="pf-c-tree-view__node-content">{children}</div>
                </div>
            </div>
        </div>
    );
}

export default MitreAttackVectorsFormSection;
