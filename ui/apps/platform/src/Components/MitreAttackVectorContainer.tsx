import React, { ReactElement, ReactNode } from 'react';
import { Trash2 } from 'react-feather';

export type MitreAttackVectorContainerProps = {
    headerText: string;
    children: ReactNode;
    onDelete?: () => void;
    isLight?: boolean;
    isReadOnly?: boolean;
};

function MitreAttackVectorContainer({
    headerText,
    children,
    onDelete,
    isLight = false,
    isReadOnly = false,
}: MitreAttackVectorContainerProps): ReactElement {
    return (
        <div
            className={`border border-base-400 mb-4 ${
                isLight ? 'bg-base-100' : 'bg-primary-100'
            } rounded`}
        >
            <div className="flex flex-1 items-center">
                <div className="flex flex-1 p-3 text-base-600 font-700">{headerText}</div>
                {onDelete && !isReadOnly && (
                    <div className="border-l border-base-400">
                        <button type="button" className="p-3 hover:bg-alert-200" onClick={onDelete}>
                            <Trash2 className="h-4 w-4 text-base-500" />
                        </button>
                    </div>
                )}
            </div>
            <div className="border-t border-base-400">{children}</div>
        </div>
    );
}

export default MitreAttackVectorContainer;
