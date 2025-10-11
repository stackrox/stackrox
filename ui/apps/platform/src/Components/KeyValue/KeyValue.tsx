import React from 'react';
import type { ReactElement } from 'react';

type KeyValueProps = {
    label: string;
    value: string;
    className?: string;
};

function KeyValue({ label, value, className = '' }: KeyValueProps): ReactElement {
    return (
        <div className={className}>
            <span className="font-700 capitalize">{label}</span> {value}
        </div>
    );
}

export default KeyValue;
