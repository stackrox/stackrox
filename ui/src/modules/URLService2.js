import qs from 'qs';
import pageTypes from 'constants/pageTypes';
import contextTypes from 'constants/contextTypes';
import { generatePath } from 'react-router-dom';
import {
    nestedCompliancePaths,
    urlEntityListTypes,
    urlEntityTypes,
    riskPath,
    secretsPath,
    configManagementPath,
    nestedPaths
} from '../routePaths';

const entityPropNames = [
    'pageEntityType',
    'pageEntityId',
    'entityListType1',
    'entityId1',
    'entityType2',
    'entityId2'
];
const listPropNames = ['pageEntityListType', 'entityId1', 'entityType2', 'entityId2'];

function getTypeKeyFromParamValue(value) {
    const match =
        Object.entries(urlEntityListTypes).find(entry => entry[1] === value) ||
        Object.entries(urlEntityTypes).find(entry => entry[1] === value);
    return match ? match[0] : null;
}

function getEntityTypeKeyFromValue(entityTypeValue) {
    const match = Object.entries(urlEntityListTypes).find(entry => entry[1] === entityTypeValue);
    return match ? match[0] : null;
}

function getPath(urlParams) {
    let pageType = pageTypes.DASHBOARD;
    if (urlParams.pageEntityType) pageType = pageTypes.ENTITY;
    else if (urlParams.pageEntityListType) pageType = pageTypes.LIST;
    const { context } = urlParams;

    const pathMap = {
        [contextTypes.CONFIG_MANAGEMENT]: {
            [pageTypes.DASHBOARD]: configManagementPath,
            [pageTypes.ENTITY]: `${configManagementPath}${nestedPaths.ENTITY}`,
            [pageTypes.LIST]: `${configManagementPath}${nestedPaths.LIST}`
        },
        [contextTypes.COMPLIANCE]: {
            [pageTypes.DASHBOARD]: nestedCompliancePaths.DASHBOARD,
            [pageTypes.ENTITY]: nestedCompliancePaths.ENTITY,
            [pageTypes.LIST]: nestedCompliancePaths.LIST
        },
        [contextTypes.RISK]: {
            [pageTypes.ENTITY]: riskPath,
            [pageTypes.LIST]: '/main/risk'
        },
        [contextTypes.SECRET]: {
            [pageTypes.ENTITY]: secretsPath,
            [pageTypes.LIST]: '/main/secrets'
        }
    };

    const contextData = pathMap[context];
    if (!contextData) return null;

    const path = contextData[pageType];
    if (!path) return null;

    const params = { ...urlParams };

    // Patching url params for legacy contexts
    if (context === contextTypes.SECRET) {
        params.secretId = params.pageEntityId;
    } else if (context === contextTypes.RISK) {
        params.deploymentId = params.pageEntityId;
    }

    // Mapping from entity types to url entityTypes
    params.pageEntityListType = urlEntityListTypes[params.pageEntityListType];
    params.entityType2 = params.entityId2
        ? urlEntityTypes[params.entityType2]
        : urlEntityListTypes[params.entityType2];
    params.pageEntityType = urlEntityTypes[params.pageEntityType];
    params.entityListType1 = urlEntityListTypes[params.entityListType1];
    return generatePath(path, params);
}

function getParams(match, location) {
    if (!match) return {};
    const newParams = { ...match.params };
    // Mapping from url to entity types
    newParams.pageEntityListType = getTypeKeyFromParamValue(newParams.pageEntityListType);
    newParams.entityType2 = getTypeKeyFromParamValue(newParams.entityType2);
    newParams.pageEntityType = getTypeKeyFromParamValue(newParams.pageEntityType);
    newParams.entityListType1 = getTypeKeyFromParamValue(newParams.entityListType1);

    return {
        ...newParams,
        query: qs.parse(location.search, { ignoreQueryPrefix: true })
    };
}

function getLinkTo(params) {
    const { query, ...urlParams } = params;
    const pathname = getPath(urlParams);
    const search = query ? qs.stringify(query, { addQueryPrefix: true }) : '';

    return {
        pathname,
        search,
        url: pathname + search
    };
}

function isIdParam(paramName) {
    return paramName.toLowerCase().includes('entityid');
}

function getNextEmptyParamName(urlParams) {
    const propNames = urlParams.pageEntityListType ? listPropNames : entityPropNames;
    let i = 0;
    for (; i < propNames.length; i += 1) {
        const propName = propNames[i];
        if (!urlParams[propName]) return propName;
    }
    return null;
}

function getLastUsedParamName(urlParams) {
    const propNames = (urlParams.pageEntityListType ? listPropNames : entityPropNames).reverse();
    for (let i = 0; i < propNames.length; i += 1) {
        if (urlParams[propNames[i]]) return propNames[i];
    }
    return null;
}

class URL {
    constructor(match, location) {
        const { query, ...urlParams } = getParams(match, location);
        this.q = query;
        this.urlParams = urlParams;
    }

    base(type, id, context) {
        const params = { context: context || this.urlParams.context };
        if (id) {
            // Entity path
            params.pageEntityType = type;
            params.pageEntityId = id;
        } else if (type) {
            // List path
            params.pageEntityListType = type;
        }

        this.urlParams = params;

        return this;
    }

    push(type, id) {
        const newParams = { ...this.urlParams };

        // Dashboard
        if (!newParams.pageEntityListType && !newParams.pageEntityType) {
            return this.base(type, id);
        }

        const emptyParamName = getNextEmptyParamName(newParams);

        if (emptyParamName) {
            if (isIdParam(emptyParamName)) {
                newParams[emptyParamName] = id;
            } else if (emptyParamName === 'entityType2') {
                newParams.entityType2 = type;
                newParams.entityId2 = id;
            } else {
                newParams[emptyParamName] = type;
            }
        }

        this.urlParams = newParams;
        return this;
    }

    clearSidePanelParams() {
        const p = this.urlParams;
        delete p.entityId1;
        delete p.entityType2;
        delete p.entityId2;
        delete p.entityListType1;

        return this;
    }

    query(queryChanges) {
        const newQuery = { ...this.q, ...queryChanges };
        this.q = newQuery;
        return this;
    }

    pop() {
        const { urlParams } = this;
        const paramName = getLastUsedParamName(urlParams);
        if (paramName) delete urlParams[paramName];

        return this;
    }

    url() {
        const { q: query, urlParams } = this;
        return getLinkTo({ query, ...urlParams }).url;
    }
}

function getURL(match, location) {
    return new URL(match, location);
}
export default {
    getParams,
    getLinkTo,
    getEntityTypeKeyFromValue,
    getURL
};
