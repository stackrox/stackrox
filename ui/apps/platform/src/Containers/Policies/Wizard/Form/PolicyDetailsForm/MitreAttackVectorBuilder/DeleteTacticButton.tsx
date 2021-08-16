import React, { ReactElement } from 'react';
import { FieldArrayFieldsProps } from 'redux-form';
import { Trash2 } from 'react-feather';

import { MitreAttackVectorId } from 'services/MitreService';

export type DeleteTacticButtonProps = {
    fields: FieldArrayFieldsProps<MitreAttackVectorId>;
    index: number;
};

function DeleteTacticButton({ fields, index }: DeleteTacticButtonProps): ReactElement {
    function onDeleteTactic() {
        fields.remove(index);
    }

    return (
        <button type="button" className="p-3 hover:bg-alert-200" onClick={onDeleteTactic}>
            <Trash2 className="h-4 w-4 text-base-500" />
        </button>
    );
}
export default DeleteTacticButton;
