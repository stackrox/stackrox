import type { ReactNode } from 'react';
import pluralize from 'pluralize';
import { ArrowRightCircle } from 'react-feather';

import Popper from 'Components/Popper';

type KeyValuePair = {
    key: string;
    value: string | number | boolean;
};

function renderKeyValuePairs(data: KeyValuePair[]): ReactNode {
    return data.map(({ key, value }) => (
        <div className="mt-2" key={key}>
            {key} : {String(value)}
        </div>
    ));
}

type ResourceCountPopperProps = {
    data: KeyValuePair[];
    label: string;
    renderContent?: (data: KeyValuePair[]) => ReactNode;
};

function ResourceCountPopper({
    data,
    label,
    renderContent = renderKeyValuePairs,
}: ResourceCountPopperProps) {
    const { length } = data;
    const buttonContent = (
        <div className="flex justify-between items-center">
            <span className="pr-2">{`${length} ${pluralize(label, length)}`}</span>
            <ArrowRightCircle size={12} />
        </div>
    );

    return (
        <Popper
            disabled={!length}
            placement="bottom"
            buttonClass={`w-full rounded border border-base-400 bg-base-100 p-1 px-2 text-left text-sm ${
                length ? 'hover:bg-base-200' : ''
            }`}
            buttonContent={buttonContent}
            popperContent={
                <div className="border border-base-300 p-4 shadow bg-base-100 whitespace-nowrap">
                    {renderContent(data)}
                </div>
            }
        />
    );
}

export default ResourceCountPopper;
