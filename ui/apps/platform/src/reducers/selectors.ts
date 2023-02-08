import { Metadata } from 'types/metadataService.proto';
import { BannerConfig, LoginNotice } from 'types/config.proto';

import { selectors } from './index';

// metadata

export const metadataSelector = selectors.getMetadata as (state: unknown) => Metadata;

// serverError

export type ServerStatus = 'RESURRECTED' | 'UNREACHABLE' | 'UP' | null | undefined;

export const serverStatusSelector = selectors.getServerState as (state: unknown) => ServerStatus;

// systemConfig

export const publicConfigFooterSelector = selectors.getPublicConfigFooter as (
    state: unknown
) => BannerConfig | null;

export const publicConfigHeaderSelector = selectors.getPublicConfigHeader as (
    state: unknown
) => BannerConfig | null;

export const publicConfigLoginNoticeSelector = selectors.getPublicConfigLoginNotice as (
    state: unknown
) => LoginNotice | null;
