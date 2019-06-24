import qs from 'qs';
import pageTypes from 'constants/pageTypes';
import contextTypes from 'constants/contextTypes';
import { generatePath } from 'react-router-dom';
import { entityParamNames, listParamNames } from 'constants/url';
import entityTypes from 'constants/entityTypes';
import { nestedPaths, riskPath, secretsPath, urlEntityListTypes, urlEntityTypes } from 'routePaths';

export function getTypeKeyFromParamValue(value, listOnly) {
    const listMatch = Object.entries(urlEntityListTypes).find(entry => entry[1] === value);
    const entityMatch = Object.entries(urlEntityTypes).find(entry => entry[1] === value);
    const match = listOnly ? listMatch : listMatch || entityMatch;
    return match ? match[0] : null;
}

function isListType(value) {
    return Object.values(urlEntityListTypes).includes(value);
}

function getEntityTypeKeyFromValue(entityTypeValue) {
    const match = Object.entries(urlEntityListTypes).find(entry => entry[1] === entityTypeValue);
    return match ? match[0] : null;
}

function getPageType(urlParams) {
    if (urlParams.pageEntityListType) {
        return pageTypes.LIST;
    }
    if (urlParams.pageEntityType) {
        return pageTypes.ENTITY;
    }
    return pageTypes.DASHBOARD;
}

function getPath(urlParams) {
    const pageType = getPageType(urlParams);
    const { context } = urlParams;

    const defaultPathMap = {
        [pageTypes.DASHBOARD]: nestedPaths.DASHBOARD,
        [pageTypes.ENTITY]: nestedPaths.ENTITY,
        [pageTypes.LIST]: nestedPaths.LIST
    };

    const legacyPathMap = {
        [contextTypes.RISK]: {
            [pageTypes.ENTITY]: riskPath,
            [pageTypes.LIST]: '/main/risk'
        },
        [contextTypes.SECRET]: {
            [pageTypes.ENTITY]: secretsPath,
            [pageTypes.LIST]: '/main/secrets'
        }
    };

    const contextData = legacyPathMap[context] || defaultPathMap;
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
    params.entityType2 =
        urlEntityTypes[params.entityType2] || urlEntityListTypes[params.entityListType2];
    params.pageEntityType = urlEntityTypes[params.pageEntityType];
    params.entityListType1 = urlEntityListTypes[params.entityListType1];

    return generatePath(path, params);
}

function getParams(match, location) {
    if (!match) return {};
    const newParams = { ...match.params };

    // Mapping from url to entity types
    if (newParams.pageEntityListType)
        newParams.pageEntityListType = getTypeKeyFromParamValue(newParams.pageEntityListType);
    if (newParams.entityListType2) {
        newParams.entityListType2 = getTypeKeyFromParamValue(newParams.entityListType2, true);
    } else if (newParams.entityType2) {
        if (isListType(newParams.entityType2)) {
            newParams.entityListType2 = getTypeKeyFromParamValue(newParams.entityType2, true);
            delete newParams.entityType2;
        } else {
            newParams.entityType2 = getTypeKeyFromParamValue(newParams.entityType2);
        }
    }

    if (newParams.pageEntityType)
        newParams.pageEntityType = getTypeKeyFromParamValue(newParams.pageEntityType);
    if (newParams.entityListType1)
        newParams.entityListType1 = getTypeKeyFromParamValue(newParams.entityListType1);

    return {
        ...newParams,
        query:
            location && location.search
                ? qs.parse(location.search, { ignoreQueryPrefix: true })
                : {}
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
    if (!paramName) return false;
    return paramName.toLowerCase().includes('entityid');
}

const pageTypesToParamNames = {
    [pageTypes.ENTITY]: entityParamNames,
    [pageTypes.LIST]: listParamNames
};

function getNextEmptyParamName(urlParams) {
    const propNames = Object.values(pageTypesToParamNames[getPageType(urlParams)]);
    if (urlParams.entityListType2) {
        propNames[propNames.indexOf('entityType2')] = 'entityListType2';
    }
    let i = 0;
    for (; i < propNames.length; i += 1) {
        const propName = propNames[i];
        if (!urlParams[propName]) {
            return propName;
        }
    }

    return null;
}

function getLastUsedParamName(urlParams) {
    const pageType = getPageType(urlParams);
    if (!pageType) return null;
    const paramTypes = pageTypesToParamNames[pageType];
    if (!paramTypes) return null;
    const propNames = Object.values(paramTypes).reverse();
    if (urlParams.entityListType2) {
        propNames[propNames.indexOf('entityType2')] = 'entityListType2';
    }
    for (let i = 0; i < propNames.length; i += 1) {
        const propName = propNames[i];
        if (urlParams[propName]) return propName;
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

    push(val, val2) {
        const { urlParams } = this;
        let newParams;

        const isType = !!entityTypes[val];

        // Not pushing a value, return
        if (!val) {
            return this;
        }

        // Pushing initial values, use base instead
        if (!urlParams.pageEntityListType && !urlParams.pageEntityType) {
            return this.base(val, val2);
        }

        let emptyParamName = getNextEmptyParamName(urlParams);
        emptyParamName =
            emptyParamName === 'entityType2' && !val2 ? 'entityListType2' : emptyParamName;
        const replaceParamName = getLastUsedParamName(urlParams);

        if (emptyParamName) {
            newParams = { ...urlParams };
            if (isIdParam(emptyParamName) === !isType) {
                // Next empty param type matches the val type, push it.
                newParams[emptyParamName] = val;
                if (emptyParamName === 'entityType2') {
                    newParams.entityId2 = val2;
                }
            } else {
                // next empty param type is different than input type, replace last used param instead of push
                newParams[replaceParamName] = val;
            }
        } else if (isType) {
            newParams = {
                context: urlParams.context,
                pageEntityType: urlParams.entityType2 || urlParams.entityListType2,
                pageEntityId: urlParams.entityId2,
                entityListType1: val
            };
            if (val2) newParams.entityId1 = val2;
        }

        this.urlParams = newParams;
        return this;
    }

    pop() {
        const { urlParams } = this;
        const paramName = getLastUsedParamName(urlParams);
        if (paramName) delete urlParams[paramName];

        if (paramName === 'entityId2') {
            delete urlParams.entityType2;
        }
        if (paramName === 'pageEntityId') delete urlParams.pageEntityType;

        return this;
    }

    set(paramName, value) {
        const { urlParams } = this;
        urlParams[paramName] = value;
        return this;
    }

    query(queryChanges) {
        const newQuery = { ...this.q, ...queryChanges };
        this.q = newQuery;
        return this;
    }

    clearSidePanelParams() {
        const p = this.urlParams;
        delete p.entityId1;
        delete p.entityType2;
        delete p.entityListType2;
        delete p.entityId2;
        delete p.entityListType1;

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
