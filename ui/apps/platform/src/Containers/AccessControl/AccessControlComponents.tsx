import React, { ReactElement } from 'react';

// Temporary file contains components that might move to Components folder.

export type PanelTitle2Props = {
    entityName: string;
    entityTypeLabel: string;
};

// eslint-disable-next-line import/prefer-default-export
export function PanelTitle2({ entityName, entityTypeLabel }: PanelTitle2Props): ReactElement {
    return (
        <div className="flex items-center leading-normal overflow-hidden px-4 text-base-600">
            <div className="flex flex-col">
                <span className="font-700">{entityName}</span>
                <span className="italic">{entityTypeLabel}</span>
            </div>
        </div>
    );
}
