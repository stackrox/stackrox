import React, { ReactElement } from 'react';
import { Plus } from 'react-feather';
import pluralize from 'pluralize';

import { Tooltip, TooltipOverlay } from '@stackrox/ui-components';
import PanelButton from 'Components/PanelButton';
import URLSearchInput from 'Components/URLSearchInput';
import { AccessControlEntityType } from 'constants/entityTypes';
import { accessControlLabels } from 'messages/common';

export type AccessControlListHeaderProps = {
    entityCount: number;
    entityType: AccessControlEntityType;
    isEntity: boolean;
    onClickNew: () => void;
};

function AccessControlListHeader({
    entityCount = 0,
    entityType,
    isEntity,
    onClickNew,
}: AccessControlListHeaderProps): ReactElement {
    const entityLabel = accessControlLabels[entityType];
    const textHeader = `0 ${pluralize(entityLabel, entityCount)}`;
    const textNew = `New ${entityLabel}`;

    // TODO search filter is not yet interactive
    const searchOptions = [];
    const availableCategories = [];
    const autoFocusSearchInput = true;

    // TODO factor out divs as presentation component?
    return (
        <div className="flex w-full min-h-14 border-b border-base-400">
            <div
                className="overflow-hidden mx-4 flex text-base-600 items-center tracking-wide leading-normal font-700 uppercase"
                data-testid="main-panel-header"
            >
                <Tooltip content={<TooltipOverlay>{textHeader}</TooltipOverlay>}>
                    <div className="line-clamp break-all">{textHeader}</div>
                </Tooltip>
            </div>
            <div className="flex items-center justify-end relative flex-1 px-3">
                <div className="flex flex-1 justify-start">
                    <URLSearchInput
                        className="w-full"
                        categoryOptions={searchOptions}
                        categories={availableCategories}
                        autoFocus={autoFocusSearchInput}
                    />
                </div>
                <div className="ml-2 flex">
                    <PanelButton
                        icon={<Plus className="h-4 w-4 ml-1" />}
                        tooltip={textNew}
                        className="btn btn-base ml-2"
                        onClick={onClickNew}
                        disabled={isEntity}
                    >
                        {textNew}
                    </PanelButton>
                </div>
            </div>
        </div>
    );
}

export default AccessControlListHeader;
