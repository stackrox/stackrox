const getUserAttributeMap = (attrs) => {
    const attributeMap = {};
    attrs.forEach(({ key, values }) => {
        // eslint-disable-next-line prefer-destructuring
        attributeMap[key] = values[0];
    });
    return attributeMap;
};

export default getUserAttributeMap;
