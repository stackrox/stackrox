import {
    Category,
    Description,
    Enforcement,
    LastUpdated,
    LifecycleStage,
    Name,
    Severity,
    Status,
} from 'Components/CompoundSearchFilter/attributes/policy';
import type { CompoundSearchFilterEntity } from 'Components/CompoundSearchFilter/types';

export const policySearchFilterConfig: CompoundSearchFilterEntity = {
    displayName: 'Policy',
    searchCategory: 'POLICIES',
    attributes: [
        Category,
        Description,
        Enforcement,
        LastUpdated,
        LifecycleStage,
        Name,
        Severity,
        Status,
    ],
};
