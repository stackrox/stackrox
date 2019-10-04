import entityTypes from 'constants/entityTypes';
import { DEPLOYMENTS_QUERY } from 'queries/deployment';

const LIST_QUERIES = {
    [entityTypes.DEPLOYMENT]: DEPLOYMENTS_QUERY
};

function getListQuery(listType) {
    return LIST_QUERIES[listType];
}

export default {
    getListQuery
};
