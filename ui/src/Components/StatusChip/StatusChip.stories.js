/* eslint-disable no-use-before-define */
import React from 'react';

import StatusChip from './StatusChip';

export default {
    title: 'StatusChip',
    component: StatusChip
};

export const passingStatusChip = () => {
    const goodStatusChip = 'pass';

    return <StatusChip status={goodStatusChip} />;
};

export const failingStatusChip = () => {
    const badStatusChip = 'fail';

    return <StatusChip status={badStatusChip} />;
};

export const activeStatusChip = () => {
    const activeStatus = 'active';

    return <StatusChip status={activeStatus} />;
};

export const inactiveStatusChip = () => {
    const inactiveStatus = 'inactive';

    return <StatusChip status={inactiveStatus} />;
};

export const unknownStatusChip = () => {
    const whatStatusChip = 'foo';

    return <StatusChip status={whatStatusChip} />;
};
