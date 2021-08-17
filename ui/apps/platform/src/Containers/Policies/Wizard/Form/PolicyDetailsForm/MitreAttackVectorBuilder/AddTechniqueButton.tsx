import React, { ReactElement } from 'react';
import { PlusCircle } from 'react-feather';

export type AddTechniqueButtonProps = {
    onClick: () => void;
};

function AddTechniqueButton({ onClick }: AddTechniqueButtonProps): ReactElement {
    return (
        <button
            type="button"
            className="flex flex-1 justify-center p-3 w-full hover:bg-primary-100"
            onClick={onClick}
        >
            <PlusCircle className="h-4 w-4 text-base-500 mr-4" />
        </button>
    );
}
export default AddTechniqueButton;
