import { PrivateConfig, PublicConfig } from 'types/config.proto';
import { PlatformComponentsConfigRules } from '../configUtils';

export type Values = {
    privateConfig: PrivateConfig;
    publicConfig: PublicConfig;
    platformComponentsConfigRules: PlatformComponentsConfigRules;
};
