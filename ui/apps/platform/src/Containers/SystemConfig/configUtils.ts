import type { PlatformComponentRule, PlatformComponentsConfig } from 'types/config.proto';

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
const defaultRedHatLayeredProductsRule = {
    name: 'red hat layered products',
    namespaceRule: {
        regex: '',
    },
};

/**
 * Transforms the platform components configuration into a structure compatible with Formik.
 *
 * @param platformComponentConfig - This reprsents the platform components config retrieved from the API
 * @returns - A structured set of rules organized for use in Formik
 */
export function getPlatformComponentsConfigRules(
    platformComponentConfig: PlatformComponentsConfig | undefined
): PlatformComponentsConfigRules {
    const platformComponentConfigRules: PlatformComponentsConfigRules = {
        coreSystemRule: defaultSystemRule,
        redHatLayeredProductsRule: defaultRedHatLayeredProductsRule,
        customRules: [],
    };

    platformComponentConfig?.rules.forEach((rule) => {
        if (rule.name === 'system rule') {
            platformComponentConfigRules.coreSystemRule = rule;
        } else if (rule.name === 'red hat layered products') {
            platformComponentConfigRules.redHatLayeredProductsRule = rule;
        } else {
            platformComponentConfigRules.customRules.push(rule);
        }
    });

    return platformComponentConfigRules;
}
