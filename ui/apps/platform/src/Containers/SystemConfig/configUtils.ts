import { PlatformComponentRule, PlatformComponentsConfig } from 'types/config.proto';

export type PlatformComponentsConfigRules = {
    coreSystemRule: PlatformComponentRule;
    redHatLayeredProductsRule: PlatformComponentRule;
    customRules: PlatformComponentRule[];
};

// This rule should theoretically always be set. This is to safeguard against any issues with the API
const defaultSystemRule = {
    name: 'system rule',
    namespaceRule: {
        regex: '',
    },
};

// This rule should theoretically always be set. This is to safeguard against any issues with the API
const defaultRedHayLayeredProductsRule = {
    name: 'red hat layered products',
    namespaceRule: {
        regex: '',
    },
};

/**
 * Transforms the platform components configuration into a structure compatible with Formik.
 *
 * @param platformComponentsConfig - This reprsents the platform components config retrieved from the API
 * @returns - A structured set of rules organized for use in Formik
 */
export function getPlatformComponentsConfigRules(
    platformComponentsConfig: PlatformComponentsConfig | undefined
): PlatformComponentsConfigRules {
    const platformComponentsConfigRules: PlatformComponentsConfigRules = {
        coreSystemRule: defaultSystemRule,
        redHatLayeredProductsRule: defaultRedHayLayeredProductsRule,
        customRules: [],
    };

    platformComponentsConfig?.rules.forEach((rule) => {
        if (rule.name === 'system rule') {
            platformComponentsConfigRules.coreSystemRule = rule;
        } else if (rule.name === 'red hat layered products') {
            platformComponentsConfigRules.redHatLayeredProductsRule = rule;
        } else {
            platformComponentsConfigRules.customRules.push(rule);
        }
    });

    return platformComponentsConfigRules;
}
