import React, { ReactElement, ReactNode } from 'react';
import AppBanner, { AppBannerSize } from 'Components/AppBanner';

export type AppBannerOptions = {
    backgroundColor: string;
    color: string;
    enabled: boolean;
    size: AppBannerSize;
    text: string;
};

export type PublicConfig = {
    header: AppBannerOptions;
    footer: AppBannerOptions;
    loginNotice: string;
};

export type AppWrapperProps = {
    publicConfig: PublicConfig | undefined;
    children: ReactNode;
};

const AppWrapper = ({ publicConfig, children }: AppWrapperProps): ReactElement => {
    return (
        <div className="flex flex-col h-full">
            {publicConfig?.header?.enabled && (
                <AppBanner {...publicConfig.header} dataTestId="header-banner" />
            )}
            {children}
            {publicConfig?.footer?.enabled && (
                <AppBanner {...publicConfig.footer} dataTestId="footer-banner" />
            )}
        </div>
    );
};

export default AppWrapper;
