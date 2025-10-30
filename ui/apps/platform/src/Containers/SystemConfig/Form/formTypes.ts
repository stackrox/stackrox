import type { PrivateConfig, PublicConfig } from 'types/config.proto';
import type { PlatformComponentsConfigRules } from '../configUtils';

export type Values = {
    privateConfig: PrivateConfig;
    publicConfig: PublicConfig;
    platformComponentConfigRules: PlatformComponentsConfigRules;
};
