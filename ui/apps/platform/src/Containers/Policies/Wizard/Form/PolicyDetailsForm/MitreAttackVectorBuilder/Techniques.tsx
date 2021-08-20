import React, { ReactElement } from 'react';
import { FieldArrayFieldsProps } from 'redux-form';

import { MitreAttackVector } from 'services/MitreService';

import ReduxSelectField from 'Components/forms/ReduxSelectField';
import MitreAttackVectorContainer from 'Components/MitreAttackVectorContainer';
import AddTechniqueButton from './AddTechniqueButton';

export type TechniqueProps = {
    fields: FieldArrayFieldsProps<string>;
    possibleTechniques: MitreAttackVector['techniques'];
    isReadOnly?: boolean;
};

function Techniques({
    fields,
    possibleTechniques,
    isReadOnly = false,
}: TechniqueProps): ReactElement {
    // @TODO: filter available options based on techniques already selected
    const techniqueOptions = possibleTechniques.map((technique) => ({
        label: `${technique.name} | ${technique.id}`,
        value: technique.id,
    }));

    function onAddTechnique() {
        fields.push('');
    }

    return (
        <>
            {fields.map((field: string, index: number) => {
                const techniqueId = fields.get(index);
                const technique = possibleTechniques.find((possibleTechnique) => {
                    return possibleTechnique.id === techniqueId;
                });
                // subtechniques have a "." in the id
                const headerText = techniqueId.includes('.') ? 'Subtechnique' : 'Technique';

                function onDeleteTechnique() {
                    fields.remove(index);
                }

                return (
                    <MitreAttackVectorContainer
                        headerText={headerText}
                        onDelete={onDeleteTechnique}
                        isLight
                        isReadOnly={isReadOnly}
                    >
                        <div className="p-3 space-y-3">
                            <ReduxSelectField
                                name={field}
                                options={techniqueOptions}
                                value={techniqueId}
                                placeholder="Select a technique..."
                                disabled={isReadOnly}
                            />
                            <div>{technique?.description}</div>
                        </div>
                    </MitreAttackVectorContainer>
                );
            })}
            {!isReadOnly && <AddTechniqueButton onClick={onAddTechnique} />}
        </>
    );
}

export default Techniques;
