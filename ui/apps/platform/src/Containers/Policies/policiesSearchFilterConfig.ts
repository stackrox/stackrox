import {
    Category,
    Description,
    Enforcement,
    LastUpdated,
    LifecycleStage,
    Name,
    Severity,
    SkipContainerType,
    SkipImageLayers,
    Status,
} from 'Components/CompoundSearchFilter/attributes/policy';
import type {
    CompoundSearchFilterAttribute,
    CompoundSearchFilterEntity,
} from 'Components/CompoundSearchFilter/types';

type FeatureFlags = {
    initContainerSupport: boolean;
    imageLayerFilter: boolean;
};

export function getPolicySearchFilterConfig(flags: FeatureFlags): CompoundSearchFilterEntity {
    const attributes: CompoundSearchFilterAttribute[] = [
        Category,
        Description,
        Enforcement,
        LastUpdated,
        LifecycleStage,
        Name,
        Severity,
        Status,
    ];

    if (flags.initContainerSupport) {
        attributes.push(SkipContainerType);
    }
    if (flags.imageLayerFilter) {
        attributes.push(SkipImageLayers);
    }

    return {
        displayName: 'Policy',
        searchCategory: 'POLICIES',
        attributes,
    };
}
