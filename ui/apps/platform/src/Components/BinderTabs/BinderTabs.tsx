import React, { ReactElement } from 'react';

import useTabs from 'hooks/useTabs';

type BinderTabHeaderProps = {
    title: string;
    dataTestId?: string;
    isActive: boolean;
    onSelectTab: () => void;
};

function BinderTabHeader({
    title,
    isActive,
    onSelectTab,
    dataTestId = 'tab',
}: BinderTabHeaderProps): ReactElement {
    const className = 'border-base-400 border'; // 400 instead of 300 to contrast with bg-base-200
    const buttonClassName = `${isActive ? 'bg-primary-200' : 'bg-base-100'} text-base-600 p-3`;

    return (
        <li key={title} className={className} data-testid={dataTestId}>
            <button type="button" className={buttonClassName} onClick={onSelectTab}>
                {title}
            </button>
        </li>
    );
}

export type BinderTabsProps = {
    children: (ReactElement | null)[];
};

function BinderTabs({ children }: BinderTabsProps): ReactElement {
    const { tabHeaders, activeTabContent } = useTabs(children);

    const tabHeaderComponents = tabHeaders.map(({ title, isActive, onSelectTab, dataTestId }) => {
        return (
            <BinderTabHeader
                key={title}
                title={title}
                dataTestId={dataTestId}
                isActive={isActive}
                onSelectTab={onSelectTab}
            />
        );
    });

    return (
        <div className="flex flex-1 flex-col">
            <ul className="flex items-center" data-testid="tabs">
                {tabHeaderComponents}
            </ul>
            <div className="bg-primary-100 rounded-b rounded-tr-lg shadow flex flex-1">
                {activeTabContent}
            </div>
        </div>
    );
}

export default BinderTabs;
