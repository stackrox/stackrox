import type { CSSProperties } from 'react';

import type { BannerConfig, BannerConfigSize } from 'types/config.proto';

const sizeVarMap: Record<BannerConfigSize, string> = {
    UNSET: 'var(--pf-t--global--font--size--xs)',
    SMALL: 'var(--pf-t--global--font--size--xs)',
    MEDIUM: 'var(--pf-t--global--font--size--sm)',
    LARGE: 'var(--pf-t--global--font--size--md)',
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
