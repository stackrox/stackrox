import React from 'react';
import { groupBy } from 'lodash';
import { Title, Divider, Flex, Text } from '@patternfly/react-core';

import { policyCriteriaCategories, type PolicyCriteriaCategoryKey } from 'messages/common';

import PolicyCriteriaCategory from './PolicyCriteriaCategory';
import { Descriptor } from './policyCriteriaDescriptors';

type CriteriaDomain =
    | 'Image criteria'
    | 'Workload criteria'
    | 'Deployment events'
    | 'Audit log events';

const criteriaDomains: Record<PolicyCriteriaCategoryKey, CriteriaDomain> = {
    [policyCriteriaCategories.IMAGE_REGISTRY]: 'Image criteria',
    [policyCriteriaCategories.IMAGE_CONTENTS]: 'Image criteria',
    [policyCriteriaCategories.CONTAINER_CONFIGURATION]: 'Workload criteria',
    [policyCriteriaCategories.DEPLOYMENT_METADATA]: 'Workload criteria',
    [policyCriteriaCategories.STORAGE]: 'Workload criteria',
    [policyCriteriaCategories.NETWORKING]: 'Workload criteria',
    [policyCriteriaCategories.DEPLOYMENT_ACCESS_CONTROL]: 'Workload criteria',
    [policyCriteriaCategories.PROCESS_ACTIVITY]: 'Deployment events',
    [policyCriteriaCategories.BASELINE_DEVIATION]: 'Deployment events',
    [policyCriteriaCategories.USER_ISSUED_CONTAINER_COMMANDS]: 'Deployment events',
    [policyCriteriaCategories.AUDIT_LOG]: 'Audit log events',
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
                    {/* If there is only one category, don't show an extra level domain */}
                    {Object.keys(categories).length > 1 && <Text component="h3">{domain}</Text>}
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
