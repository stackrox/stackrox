import React, { ReactElement } from 'react';

type KeyValueProps = {
    label: string;
    value: string;
};

function KeyValue({ label, value }: KeyValueProps): ReactElement {
    return (
        <div>
            <span className="font-700 capitalize">{label}</span> {value}
        </div>
    );
}

export default KeyValue;
