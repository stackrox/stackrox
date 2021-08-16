import React, { ReactElement } from 'react';
import { FieldArrayFieldsProps } from 'redux-form';

import { MitreAttackVectorId } from 'services/MitreService';

import ReduxSelectField from 'Components/forms/ReduxSelectField';
import AddTacticButton from './AddTacticButton';
import DeleteTacticButton from './DeleteTacticButton';
import useFetchMitreAttackVectors from './useFetchMitreAttackVectors';
import { FormSectionBody, FormSectionFooter } from '../FormSection';

export type MitreAttackVectorBuilderProps = {
    fields: FieldArrayFieldsProps<MitreAttackVectorId>;
};

function MitreAttackVectorBuilder({ fields }: MitreAttackVectorBuilderProps): ReactElement {
    const { mitreAttackVectors, isLoading } = useFetchMitreAttackVectors();

    // @TODO: filter available options based on tactics already selected
    const tacticOptions = mitreAttackVectors.map((mitreAttackVector) => ({
        label: `${mitreAttackVector.tactic.name} | ${mitreAttackVector.tactic.id}`,
        value: mitreAttackVector.tactic.id,
    }));

    return (
        <>
            <FormSectionBody>
                <div className="gap-4">
                    {fields.map((field: string, index: number) => {
                        const tacticId = fields.get(index).tactic;
                        const tacticDetail = mitreAttackVectors.find((mitreAttackVector) => {
                            return mitreAttackVector.tactic.id === tacticId;
                        });
                        // @TODO: Extract out into it's own MitreAttackTactic component
                        // @TODO: Add techniques/subtechniques
                        return (
                            <div
                                className="border border-base-400 mb-4 bg-primary-100 rounded"
                                key={tacticId}
                            >
                                <div className="flex flex-1 items-center">
                                    <div className="flex flex-1 p-3">Tactic: </div>
                                    <div className="border-l border-base-400">
                                        <DeleteTacticButton fields={fields} index={index} />
                                    </div>
                                </div>
                                <div className="p-3 border-t border-base-400">
                                    <ReduxSelectField
                                        name={`${field}.tactic`}
                                        options={tacticOptions}
                                        value={tacticId}
                                        disabled={isLoading}
                                        placeholder="Select a tactic..."
                                    />
                                    <div className="py-3">{tacticDetail?.tactic.description}</div>
                                </div>
                            </div>
                        );
                    })}
                </div>
            </FormSectionBody>
            <FormSectionFooter>
                <AddTacticButton fields={fields} />
            </FormSectionFooter>
        </>
    );
}

export default MitreAttackVectorBuilder;
