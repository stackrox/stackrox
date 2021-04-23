import React, { CSSProperties, ReactElement } from 'react';

export type AppBannerSize = 'UNSET' | 'SMALL' | 'MEDIUM' | 'LARGE';

export type AppBannerProps = {
    dataTestId: string;
    backgroundColor: string;
    color: string;
    size: AppBannerSize;
    text: string;
};

const sizeClassMap = {
    UNSET: 'var(--pf-global--FontSize--xs)',
    SMALL: 'var(--pf-global--FontSize--xs)',
    MEDIUM: 'var(--pf-global--FontSize--sm)',
    LARGE: 'var(--pf-global--FontSize--md)',
};

const AppBanner = ({
    dataTestId,
    text,
    color,
    size,
    backgroundColor,
}: AppBannerProps): ReactElement => {
    const style = {
        '--pf-c-banner--BackgroundColor': backgroundColor,
        '--pf-c-banner--Color': color,
        '--pf-c-banner--FontSize': sizeClassMap[size],
    } as CSSProperties;
    return (
        <div className="pf-c-banner pf-u-text-align-center" style={style} data-testid={dataTestId}>
            {text}
        </div>
    );
};

export default AppBanner;
