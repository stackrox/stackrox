import React from 'react';
import { groupBy } from 'lodash';
import { Title, Divider, Flex, Text } from '@patternfly/react-core';

import { policyCriteriaCategories, type PolicyCriteriaCategoryKey } from 'messages/common';

import PolicyCriteriaCategory from './PolicyCriteriaCategory';
import { Descriptor } from './policyCriteriaDescriptors';

type CriteriaDomain =
    | 'Image criteria'
    | 'Workload configuration'
    | 'Workload activity'
    | 'Kubernetes resource operations';

const criteriaDomains: Record<PolicyCriteriaCategoryKey, CriteriaDomain> = {
    [policyCriteriaCategories.IMAGE_REGISTRY]: 'Image criteria',
    [policyCriteriaCategories.IMAGE_CONTENTS]: 'Image criteria',
    [policyCriteriaCategories.IMAGE_SCANNING]: 'Image criteria',
    [policyCriteriaCategories.CONTAINER_CONFIGURATION]: 'Workload configuration',
    [policyCriteriaCategories.DEPLOYMENT_METADATA]: 'Workload configuration',
    [policyCriteriaCategories.STORAGE]: 'Workload configuration',
    [policyCriteriaCategories.NETWORKING]: 'Workload configuration',
    [policyCriteriaCategories.ACCESS_CONTROL]: 'Workload configuration',
    [policyCriteriaCategories.PROCESS_ACTIVITY]: 'Workload activity',
    [policyCriteriaCategories.BASELINE_DEVIATION]: 'Workload activity',
    [policyCriteriaCategories.USER_ISSUED_CONTAINER_COMMANDS]: 'Workload activity',
    [policyCriteriaCategories.RESOURCE_OPERATION]: 'Kubernetes resource operations',
    [policyCriteriaCategories.RESOURCE_ATTRIBUTES]: 'Kubernetes resource operations',
} as const;

function getCriteriaDomains(
    keys: Descriptor[]
): Partial<Record<CriteriaDomain, Record<PolicyCriteriaCategoryKey, Descriptor[]>>> {
    const keysByDomain = groupBy(keys, ({ category }) => criteriaDomains[category]);
    const domains = {};

    Object.entries(keysByDomain).forEach(([domain, keys]) => {
        domains[domain] = groupBy(keys, 'category');
    });

    return domains;
}

type PolicyCriteriaKeysProps = {
    keys: Descriptor[];
};

function PolicyCriteriaKeys({ keys }: PolicyCriteriaKeysProps) {
    const domains = getCriteriaDomains(keys);

    return (
        <Flex direction={{ default: 'column' }} spaceItems={{ default: 'spaceItemsMd' }}>
            <Title headingLevel="h2">Drag out policy fields</Title>
            <Divider component="div" />
            {Object.entries(domains).map(([domain, categories]) => (
                <Flex
                    key={domain}
                    direction={{ default: 'column' }}
                    spaceItems={{ default: 'spaceItemsXs' }}
                >
                    <Text component="h3">{domain}</Text>
                    <Flex
                        key={domain}
                        direction={{ default: 'column' }}
                        spaceItems={{ default: 'spaceItemsNone' }}
                    >
                        {Object.entries(categories).map(([category, keys]) => (
                            <PolicyCriteriaCategory
                                key={category}
                                category={category}
                                keys={keys}
                                isOpenDefault={false}
                            />
                        ))}
                    </Flex>
                </Flex>
            ))}
        </Flex>
    );
}

export default PolicyCriteriaKeys;
