import React, { ReactElement } from 'react';

import useTabs from 'hooks/useTabs';

type BinderTabHeaderProps = {
    title: string;
    isActive: boolean;
    onSelectTab: () => void;
};

function BinderTabHeader({ title, isActive, onSelectTab }: BinderTabHeaderProps): ReactElement {
    const className = `${
        isActive ? 'bg-primary-300' : 'bg-primary-100'
    } rounded-tr-none first:rounded-tl-lg last:rounded-tr-lg border-b border-primary-300 border-r border-t shadow`;
    const buttonClassName = `${isActive ? 'text-primary-700' : ''} p-3`;

    return (
        <li key={title} className={className}>
            <button type="button" className={buttonClassName} onClick={onSelectTab}>
                {title}
            </button>
        </li>
    );
}

export type BinderTabsProps = {
    children: ReactElement[];
};

function BinderTabs({ children }: BinderTabsProps): ReactElement {
    const { tabHeaders, activeTabContent } = useTabs(children);

    const tabHeaderComponents = tabHeaders.map(({ title, isActive, onSelectTab }) => {
        return (
            <BinderTabHeader
                key={title}
                title={title}
                isActive={isActive}
                onSelectTab={onSelectTab}
            />
        );
    });

    return (
        <div className="flex flex-1 flex-col">
            <ul className="flex font-700 items-center text-sm uppercase" data-testid="tabs">
                {tabHeaderComponents}
            </ul>
            <div className="bg-primary-100 rounded-b rounded-tr-lg shadow flex flex-1">
                {activeTabContent}
            </div>
        </div>
    );
}

export default BinderTabs;
