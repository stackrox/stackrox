import React from 'react';

import { pagingParams, sortParams } from 'constants/searchParams';

const configMgmtPaginationContext = React.createContext();

export default configMgmtPaginationContext;

export const MAIN_PAGINATION_PARAMS = {
    sortParam: sortParams.page,
    pageParam: pagingParams.page,
};

export const SIDEPANEL_PAGINATION_PARAMS = {
    sortParam: sortParams.sidePanel,
    pageParam: pagingParams.sidePanel,
};
