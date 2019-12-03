import React from 'react';

import { Trash2 } from 'react-feather';
import PanelButton from './PanelButton';

export default {
    title: 'PanelButton',
    component: PanelButton
};

function fn() {
    // eslint-disable-next-line no-alert
    alert('hi');
}

export const basicPanelButton = () => (
    <PanelButton
        icon={<Trash2 className="h-4 w-4 ml-1" />}
        className="btn btn-tertiary ml-2"
        onClick={fn}
    >
        Delete Cluster
    </PanelButton>
);

export const disabledPanelButton = () => (
    <PanelButton
        icon={<Trash2 className="h-4 w-4 ml-1" />}
        className="btn btn-tertiary ml-2"
        onClick={fn}
        disabled
    >
        Delete Cluster
    </PanelButton>
);
