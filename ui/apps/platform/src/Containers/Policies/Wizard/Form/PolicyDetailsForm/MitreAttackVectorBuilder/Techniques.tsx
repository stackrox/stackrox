import React, { ReactElement } from 'react';
import { FieldArrayFieldsProps } from 'redux-form';

import { MitreAttackVector } from 'services/MitreService';

import ReduxSelectField from 'Components/forms/ReduxSelectField';
import MitreAttackVectorContainer from './MitreAttackVectorContainer';
import AddTechniqueButton from './AddTechniqueButton';

export type TechniqueProps = {
    fields: FieldArrayFieldsProps<string>;
    possibleTechniques: MitreAttackVector['techniques'];
};

function Techniques({ fields, possibleTechniques }: TechniqueProps): ReactElement {
    // @TODO: filter available options based on techniques already selected
    const techniqueOptions = possibleTechniques.map((technique) => ({
        label: `${technique.name} | ${technique.id}`,
        value: technique.id,
    }));

    function onAddTechnique() {
        fields.push('');
    }

    return (
        <div className="p-3">
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
                    >
                        <div className="p-3">
                            <ReduxSelectField
                                name={field}
                                options={techniqueOptions}
                                value={techniqueId}
                                placeholder="Select a technique..."
                            />
                            <div className="mt-3">{technique?.description}</div>
                        </div>
                    </MitreAttackVectorContainer>
                );
            })}
            <AddTechniqueButton onClick={onAddTechnique} />
        </div>
    );
}

export default Techniques;
