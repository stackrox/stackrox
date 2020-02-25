import { format } from 'date-fns';

export function getLatestDatedItemByKey(key, list = []) {
    if (!key || !list.length || !list[0][key]) return null;

    return list.reduce((acc, item) => {
        const nextDate = item[key] && Date.parse(item[key]);

        if (!acc || nextDate > Date.parse(acc[key])) return item;

        return acc;
    }, null);
}

export function addBrandedTimestampToString(str) {
    return `StackRox:${str}-${format(new Date(), 'MM/DD/YYYY')}`;
}

export default {
    getLatestDatedItemByKey,
    addBrandedTimestampToString
};
