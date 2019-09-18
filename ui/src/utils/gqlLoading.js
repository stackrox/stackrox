import isEmpty from 'lodash/isEmpty';

function isGQLLoading(loading, data) {
    return loading && isEmpty(data);
}

export default isGQLLoading;
