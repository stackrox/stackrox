import React from 'react';

import HelpIcon from './HelpIcon';

export default {
    title: 'HelpIcon',
    component: HelpIcon,
};

export const withDescription = () => {
    return <HelpIcon description="Remember to wash your hands" />;
};
