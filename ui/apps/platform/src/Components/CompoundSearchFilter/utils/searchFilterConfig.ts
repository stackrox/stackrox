export function getFilteredConfig<
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    T extends { attributes: Record<K, any> },
    K extends keyof T['attributes'],
>(
    config: T,
    selectedAttributes: K[]
): { [P in keyof T]: P extends 'attributes' ? Pick<T['attributes'], K> : T[P] } {
    const filteredAttributes = selectedAttributes.reduce((acc: Partial<T['attributes']>, key) => {
        const attribute = config.attributes[key];
        if (attribute) {
            acc[key] = attribute;
        }
        return acc;
    }, {});

    return {
        ...config,
        attributes: filteredAttributes,
    } as { [P in keyof T]: P extends 'attributes' ? Pick<T['attributes'], K> : T[P] };
}
