import React, { ReactElement } from 'react';

import CloseButton from 'Components/CloseButton';
import { PanelHead, PanelHeadEnd } from 'Components/Panel';
import { accessControlLabels } from 'messages/common';
import { AccessControlEntityType } from 'constants/entityTypes';

// Temporary file contains components that might move to Components folder.

type PanelTitle2Props = {
    entityName: string;
    entityType: AccessControlEntityType;
};

function PanelTitle2({ entityName, entityType }: PanelTitle2Props): ReactElement {
    return (
        <div className="flex items-center leading-normal overflow-hidden px-4 text-base-600">
            <div className="flex flex-col">
                <span className="font-700" data-testid="entity-name">
                    {entityName}
                </span>
                <span className="italic" data-testid="entity-type">
                    {accessControlLabels[entityType]}
                </span>
            </div>
        </div>
    );
}

export type AccessControlSidePanelHeadProps = {
    entityType: AccessControlEntityType;
    isEditable: boolean;
    isEditing: boolean;
    name: string;
    onClickCancel: () => void;
    onClickClose: () => void;
    onClickEdit: () => void;
    onClickSave: () => void;
};

export function AccessControlSidePanelHead({
    entityType,
    isEditable,
    isEditing,
    name,
    onClickCancel,
    onClickClose,
    onClickEdit,
    onClickSave,
}: AccessControlSidePanelHeadProps): ReactElement {
    // TODO Save is disabled if form values are not valid.
    return (
        <PanelHead>
            <PanelTitle2 entityName={name} entityType={entityType} />
            {isEditing ? (
                <PanelHeadEnd>
                    <button type="button" className="btn btn-success mr-4" onClick={onClickSave}>
                        Save
                    </button>
                    <button type="button" className="btn btn-base mr-4" onClick={onClickCancel}>
                        Cancel
                    </button>
                </PanelHeadEnd>
            ) : (
                <PanelHeadEnd>
                    {isEditable && (
                        <button type="button" className="btn btn-base mr-4" onClick={onClickEdit}>
                            Edit
                        </button>
                    )}
                    <CloseButton onClose={onClickClose} className="border-base-400 border-l" />
                </PanelHeadEnd>
            )}
        </PanelHead>
    );
}

export const labelClassName = 'block pb-2 font-700';
export const inputTextClassName =
    'bg-base-100 border-2 border-base-300 hover:border-base-400 font-600 leading-normal p-2 rounded disabled:bg-base-200 w-full';

// The select element base style includes: pr-8 w-full
export const selectElementClassName =
    'bg-base-100 block border-base-300 focus:border-base-500 p-2 text-base-600 z-1';
export const selectWrapperClassName =
    'bg-base-100 border-2 border-base-300 hover:border-base-400 font-600 leading-normal rounded text-base-600 w-full';
export const selectTriggerClassName = 'border-l border-base-300';
