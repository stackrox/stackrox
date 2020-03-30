import React from 'react';
import { AlertTriangle, Shield, MessageSquare, Tag } from 'react-feather';

import IconWithCount from './IconWithCount';

export default {
    title: 'IconWithCount',
    component: IconWithCount
};

export const basic = () => {
    return <IconWithCount Icon={AlertTriangle} count={15} />;
};

export const loading = () => {
    return <IconWithCount Icon={AlertTriangle} count={15} isLoading />;
};

export const multipleIcons = () => {
    return (
        <div className="flex">
            <IconWithCount Icon={AlertTriangle} count={150} />
            <IconWithCount Icon={Shield} count={50} />
            <IconWithCount Icon={MessageSquare} count={40} />
            <IconWithCount Icon={Tag} count={20} />
        </div>
    );
};
