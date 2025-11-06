import { groupBy } from 'lodash';
import { Divider, Flex, Title } from '@patternfly/react-core';

import { policyCriteriaCategories } from 'messages/common';
import type { PolicyCriteriaCategoryKey } from 'messages/common';
import type { PolicyEventSource } from 'types/policy.proto';

import PolicyCriteriaCategory from './PolicyCriteriaCategory';
import type { Descriptor } from './policyCriteriaDescriptors';

type CriteriaDomain =
    | 'Image criteria'
    | 'Workload configuration'
    | 'Workload activity'
    | 'Kubernetes resource operations'
    | 'Node level events';

const criteriaDomains: Partial<Record<PolicyCriteriaCategoryKey, CriteriaDomain>> = {
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

const nodeEventSourceCriteriaDomains: Partial<Record<PolicyCriteriaCategoryKey, CriteriaDomain>> = {
    [policyCriteriaCategories.FILE_ACTIVITY]: 'Node level events',
} as const;

function getCriteriaDomains(
    keys: Descriptor[],
    eventSource: PolicyEventSource
): Partial<Record<CriteriaDomain, Record<PolicyCriteriaCategoryKey, Descriptor[]>>> {
    const keysByDomain = groupBy(keys, ({ category }) =>
        eventSource === 'NODE_EVENT'
            ? nodeEventSourceCriteriaDomains[category]
            : criteriaDomains[category]
    );
    const domains = {};

    Object.entries(keysByDomain).forEach(([domain, keys]) => {
        domains[domain] = groupBy(keys, 'category');
    });

    return domains;
}

type PolicyCriteriaKeysProps = {
    keys: Descriptor[];
    eventSource: PolicyEventSource;
};

function PolicyCriteriaKeys({ keys, eventSource }: PolicyCriteriaKeysProps) {
    const domains = getCriteriaDomains(keys, eventSource);
    const showDomainHeading = eventSource !== 'NODE_EVENT' && eventSource !== 'AUDIT_LOG_EVENT';

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
                    {showDomainHeading && domain && <Title headingLevel="h3">{domain}</Title>}
                    <Flex
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
