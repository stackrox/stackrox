import React, { ReactElement } from 'react';
import { FieldArrayFieldsProps } from 'redux-form';
import { PlusCircle } from 'react-feather';

import { MitreAttackVectorId } from 'services/MitreService';

export type AddTacticButtonProps = {
    fields: FieldArrayFieldsProps<MitreAttackVectorId>;
};

function AddTacticButton({ fields }: AddTacticButtonProps): ReactElement {
    function onAddTactic() {
        const newTactic: MitreAttackVectorId = {
            tactic: '',
            techniques: [],
        };
        fields.push(newTactic);
    }

    return (
        <button
            type="button"
            className="flex flex-1 justify-center p-3 w-full border-dashed border border-base-500 hover:bg-primary-100"
            onClick={onAddTactic}
        >
            <PlusCircle className="h-4 w-4 text-base-500 mr-4" />
            Add tactic
        </button>
    );
}
export default AddTacticButton;
