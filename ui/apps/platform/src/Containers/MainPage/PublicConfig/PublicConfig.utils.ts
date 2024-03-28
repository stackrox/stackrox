import { CSSProperties } from 'react';

import { BannerConfig, BannerConfigSize } from 'types/config.proto';

const sizeVarMap: Record<BannerConfigSize, string> = {
    UNSET: 'var(--pf-v5-global--FontSize--xs)',
    SMALL: 'var(--pf-v5-global--FontSize--xs)',
    MEDIUM: 'var(--pf-v5-global--FontSize--sm)',
    LARGE: 'var(--pf-v5-global--FontSize--md)',
};

export function getPublicConfigStyle({
    backgroundColor,
    color,
    size,
}: BannerConfig): CSSProperties {
    return {
        '--pf-v5-c-banner--BackgroundColor': backgroundColor,
        '--pf-v5-c-banner--Color': color,
        '--pf-v5-c-banner--FontSize': sizeVarMap[size],
    } as CSSProperties;
}
