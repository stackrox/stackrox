import React from 'react';

import { Trash2 } from 'react-feather';
import RowActionButton from './RowActionButton';

export default {
    title: 'RowActionButton',
    component: RowActionButton
};

function fn() {
    // eslint-disable-next-line no-alert
    alert('hi');
}

export const basicRowActionButton = () => (
    <RowActionButton text="Snooze CVE" onClick={fn} icon={<Trash2 className="my-1 h-4 w-4" />} />
);

export const customRowActionButton = () => (
    <RowActionButton
        text="Delete policy"
        onClick={fn}
        className="hover:bg-alert-200 text-alert-600 hover:text-alert-700"
        icon={<Trash2 className="my-1 h-4 w-4" />}
    />
);

export const multipleRowActionButtons = () => (
    <div className="flex border-2 border-r-2 border-base-400 bg-base-100 float-left">
        <RowActionButton
            text="Add To Policy"
            onClick={fn}
            icon={<Trash2 className="my-1 h-4 w-4" />}
        />
        <RowActionButton
            text="Snooze CVE"
            onClick={fn}
            border="border-l-2 border-base-400"
            icon={<Trash2 className="my-1 h-4 w-4" />}
        />
    </div>
);
