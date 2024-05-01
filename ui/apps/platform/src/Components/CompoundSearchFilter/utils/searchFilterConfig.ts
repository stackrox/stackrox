import { EntitySearchFilterConfig } from '../types';

export function getFilteredConfig<
    T extends EntitySearchFilterConfig,
    K extends keyof T['attributes'],
>(
    config: T,
    selectedAttributes: K[]
): Omit<T, 'attributes'> & { attributes: Pick<T['attributes'], K> } {
    const attributes = {} as Pick<T['attributes'], K>;

    selectedAttributes.forEach((key) => {
        attributes[key] = config.attributes[key as string];
    });

    return {
        ...config,
        attributes,
    };
}
