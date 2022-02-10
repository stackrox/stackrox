import React, { ReactElement, ReactNode } from 'react';
import * as Icon from 'react-feather';

import useDetailsOverlayToggle from 'Containers/Network/SidePanel/useDetailsOverlayToggle';

type DetailsOverlayProps = {
    headerText: string;
    subHeaderText: string;
    tabHeaderComponents?: ReactElement[];
    children: ReactNode;
    dataTestId?: string;
};

function DetailsOverlay({
    headerText,
    subHeaderText,
    tabHeaderComponents = [],
    dataTestId = 'network-details-overlay',
    children,
}: DetailsOverlayProps): ReactElement {
    const { isContentHidden, toggleContentHidden } = useDetailsOverlayToggle();

    return (
        <div
            className="flex flex-1 flex-col text-sm max-h-minus-buttons rounded-bl-lg min-w-0"
            data-testid={dataTestId}
        >
            <div className="bg-primary-800 flex items-center m-2 min-w-108 rounded-lg shadow text-primary-100">
                <div
                    className="flex flex-1 items-center p-3 cursor-pointer"
                    role="button"
                    tabIndex={0}
                    onKeyUp={toggleContentHidden}
                    onClick={toggleContentHidden}
                >
                    <div className="mr-4">
                        {isContentHidden ? (
                            <Icon.ChevronRight className="h-4 w-4" />
                        ) : (
                            <Icon.ChevronDown className="h-4 w-4" />
                        )}
                    </div>
                    <div className="flex flex-1 flex-col" data-testid={`${dataTestId}-header`}>
                        <div className="text-base">{headerText}</div>
                        <div className="italic text-primary-200 text-xs capitalize">
                            {subHeaderText}
                        </div>
                    </div>
                </div>

                {!!tabHeaderComponents.length && (
                    <ul className="flex ml-8 items-center text-sm uppercase font-700 p-3 ">
                        {tabHeaderComponents}
                    </ul>
                )}
            </div>
            {!isContentHidden && (
                <div className="flex flex-1 m-2 pb-1 overflow-auto rounded">{children}</div>
            )}
        </div>
    );
}

export default DetailsOverlay;
