import React from 'react';

import HeaderWithSubText from './HeaderWithSubText';

export default {
    title: 'HeaderWithSubText',
    component: HeaderWithSubText
};

export const withData = () => {
    const header = 'hello-world';
    const subText = '12/08/2019 | 9:51:52PM';

    return <HeaderWithSubText header={header} subText={subText} />;
};
