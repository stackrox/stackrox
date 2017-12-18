import React from 'react';

const TabContent = ({ children, active, name }) => (
    <div className={(active === name) ? 'flex flex-col h-full transition' : 'hidden'}>
        {children}
    </div>
);

export default TabContent;
