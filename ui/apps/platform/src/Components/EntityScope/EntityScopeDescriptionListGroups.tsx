import type { ReactElement } from 'react';
import {
    DescriptionListDescription,
    DescriptionListGroup,
    DescriptionListTerm,
    Flex,
    FlexItem,
} from '@patternfly/react-core';

import type { EntityScope } from 'services/ReportsService.types';

import { ruleEntityFieldText, ruleValueText } from './entityScopeText';

export type EntityScopeDescriptionListGroupsProps = {
    entityScope: EntityScope;
};

// For maximum composability and reusability:
// Render description list groups instead of description list.
// If rules array is empty, caller is responsible for conditional rendering, like a warning alert.
function EntityScopeDescriptionListGroups({
    entityScope,
}: EntityScopeDescriptionListGroupsProps): ReactElement {
    /* eslint-disable react/no-array-index-key */
    return (
        <>
            {entityScope.rules.map((rule, indexOfRule) => {
                return (
                    <DescriptionListGroup key={indexOfRule}>
                        <DescriptionListTerm>{ruleEntityFieldText(rule)}</DescriptionListTerm>
                        <DescriptionListDescription>
                            <Flex direction={{ default: 'column' }}>
                                {rule.values.map((ruleValue, indexOfValue) => (
                                    <FlexItem key={indexOfValue}>
                                        {ruleValueText(ruleValue)}
                                    </FlexItem>
                                ))}
                            </Flex>
                        </DescriptionListDescription>
                    </DescriptionListGroup>
                );
            })}
        </>
    );
    /* eslint-enable react/no-array-index-key */
}

export default EntityScopeDescriptionListGroups;
