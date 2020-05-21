/* eslint-disable react-hooks/rules-of-hooks */
import React from 'react';
import { Power, Bell, BellOff } from 'react-feather';

import IconWithState from './IconWithState';

export default {
    title: 'IconWithState',
    component: IconWithState,
};

export const disabled = () => {
    return <IconWithState Icon={Power} enabled={false} />;
};

export const enabled = () => {
    return <IconWithState Icon={Power} enabled />;
};

export const multipleIcons = () => {
    return (
        <div className="flex">
            <IconWithState Icon={Power} enabled={false} />
            <IconWithState Icon={Power} enabled />
            <IconWithState Icon={BellOff} enabled={false} />
            <IconWithState Icon={Bell} enabled />
        </div>
    );
};
