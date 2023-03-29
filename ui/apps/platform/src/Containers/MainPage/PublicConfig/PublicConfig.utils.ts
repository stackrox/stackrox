import { CSSProperties } from 'react';

import { BannerConfig, BannerConfigSize } from 'types/config.proto';

const sizeVarMap: Record<BannerConfigSize, string> = {
    UNSET: 'var(--pf-global--FontSize--xs)',
    SMALL: 'var(--pf-global--FontSize--xs)',
    MEDIUM: 'var(--pf-global--FontSize--sm)',
    LARGE: 'var(--pf-global--FontSize--md)',
};

export function getPublicConfigStyle({
    backgroundColor,
    color,
    size,
}: BannerConfig): CSSProperties {
    return {
        '--pf-c-banner--BackgroundColor': backgroundColor,
        '--pf-c-banner--Color': color,
        '--pf-c-banner--FontSize': sizeVarMap[size],
    } as CSSProperties;
}
