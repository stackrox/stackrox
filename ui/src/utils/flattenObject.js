const flattenObject = (ob) => {
    if (!ob) return ob;

    const toReturn = {};

    Object.keys(ob).forEach((i) => {
        if (Object.prototype.hasOwnProperty.call(ob, i)) {
            if (typeof ob[i] === 'object' && !Array.isArray(ob[i])) {
                const flatObject = flattenObject(ob[i]);
                if (flatObject) {
                    Object.keys(flatObject).forEach((x) => {
                        if (Object.prototype.hasOwnProperty.call(flatObject, x)) {
                            toReturn[`${i}.${x}`] = flatObject[x];
                        }
                    });
                }
            } else {
                toReturn[i] = ob[i];
            }
        }
    });

    return toReturn;
};

export default flattenObject;
