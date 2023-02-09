import React, { ReactElement, ReactNode } from 'react';

export type AppWrapperProps = {
    children: ReactNode;
};

const AppWrapper = ({ children }: AppWrapperProps): ReactElement => {
    return <div className="flex flex-col h-full">{children}</div>;
};

export default AppWrapper;
