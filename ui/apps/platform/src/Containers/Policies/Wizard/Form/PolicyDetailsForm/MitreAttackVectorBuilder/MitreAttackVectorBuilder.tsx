import React, { ReactElement } from 'react';
import { FieldArray, FieldArrayFieldsProps } from 'redux-form';

import { MitreAttackVectorId } from 'services/MitreService';

import { Message } from '@stackrox/ui-components';
import ReduxSelectField from 'Components/forms/ReduxSelectField';
import NoResultsMessage from 'Components/NoResultsMessage';
import useFetchMitreAttackVectors from './useFetchMitreAttackVectors';
import MitreAttackVectorContainer from './MitreAttackVectorContainer';
import AddTacticButton from './AddTacticButton';
import Techniques from './Techniques';
import { FormSectionBody, FormSectionFooter } from '../FormSection';

export type MitreAttackVectorBuilderProps = {
    fields: FieldArrayFieldsProps<MitreAttackVectorId>;
    isReadOnly?: boolean;
};

function MitreAttackVectorBuilder({
    fields,
    isReadOnly = false,
}: MitreAttackVectorBuilderProps): ReactElement {
    const { mitreAttackVectors, isLoading } = useFetchMitreAttackVectors();

    // @TODO: filter available options based on tactics already selected
    const tacticOptions = mitreAttackVectors.map((mitreAttackVector) => ({
        label: `${mitreAttackVector.tactic.name} | ${mitreAttackVector.tactic.id}`,
        value: mitreAttackVector.tactic.id,
    }));

    function onAddTactic() {
        const newTactic: MitreAttackVectorId = {
            tactic: '',
            techniques: [],
        };
        fields.push(newTactic);
    }

    return (
        <>
            <FormSectionBody>
                {isReadOnly && (
                    <div className="mb-4">
                        <Message>
                            Editing MITRE ATT&CK is disabled for default system policies. Please
                            clone this system policy or create a new system policy to perform edits
                            to MITRE ATT&CK.
                        </Message>
                    </div>
                )}
                <div className="gap-4">
                    {fields.length === 0 ? (
                        <NoResultsMessage message="There are no MITRE ATT&CK vectors" />
                    ) : (
                        fields.map((field: string, index: number) => {
                            const tacticId = fields.get(index).tactic;
                            const tacticDetail = mitreAttackVectors.find((mitreAttackVector) => {
                                return mitreAttackVector.tactic.id === tacticId;
                            });

                            function onDeleteTactic() {
                                fields.remove(index);
                            }

                            return (
                                <MitreAttackVectorContainer
                                    headerText="Tactic"
                                    onDelete={onDeleteTactic}
                                    isReadOnly={isReadOnly}
                                >
                                    <div className="p-3">
                                        <ReduxSelectField
                                            name={`${field}.tactic`}
                                            options={tacticOptions}
                                            value={tacticId}
                                            disabled={isLoading || isReadOnly}
                                            placeholder="Select a tactic..."
                                        />
                                        <div className="mt-3">
                                            {tacticDetail?.tactic.description}
                                        </div>
                                    </div>
                                    {tacticDetail?.techniques && (
                                        <div className="border-t border-base-300">
                                            <FieldArray
                                                name={`${field}.techniques`}
                                                component={Techniques}
                                                rerenderOnEveryChange
                                                props={{
                                                    possibleTechniques: tacticDetail.techniques,
                                                    isReadOnly,
                                                }}
                                            />
                                        </div>
                                    )}
                                </MitreAttackVectorContainer>
                            );
                        })
                    )}
                </div>
            </FormSectionBody>
            {!isReadOnly && (
                <FormSectionFooter>
                    <AddTacticButton onClick={onAddTactic} />
                </FormSectionFooter>
            )}
        </>
    );
}

export default MitreAttackVectorBuilder;
