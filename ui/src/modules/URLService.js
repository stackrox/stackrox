import qs from 'qs';
import pageTypes from 'constants/pageTypes';
import contextTypes from 'constants/contextTypes';
import { generatePath } from 'react-router-dom';
import { entityParamNames, listParamNames } from 'constants/url';
import entityTypes from 'constants/entityTypes';
import merge from 'deepmerge';
import configMgmtEntityRelationship from 'Containers/ConfigManagement/entityTabRelationships';

import {
    nestedPaths,
    riskPath,
    secretsPath,
    urlEntityListTypes,
    urlEntityTypes,
    policiesPath
} from '../routePaths';

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

function getTabsPerEntity(entityType, context) {
    const contextRelationships = {
        [contextTypes.CONFIG_MANAGEMENT]: configMgmtEntityRelationship
    };
    if (!contextRelationships[context] || !contextRelationships[context][entityType]) return [];
    return contextRelationships[context][entityType];
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
            [pageTypes.LIST]: '/main/risk',
            [pageTypes.DASHBOARD]: '/main/risk'
        },
        [contextTypes.SECRET]: {
            [pageTypes.ENTITY]: secretsPath,
            [pageTypes.LIST]: '/main/secrets',
            [pageTypes.DASHBOARD]: '/main/secrets'
        },
        [contextTypes.POLICY]: {
            [pageTypes.ENTITY]: policiesPath,
            [pageTypes.LIST]: '/main/policies',
            [pageTypes.DASHBOARD]: '/main/policies'
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
    params.entityType1 =
        urlEntityTypes[params.entityType1] || urlEntityListTypes[params.entityListType1];
    params.pageEntityType = urlEntityTypes[params.pageEntityType];

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
    if (newParams.entityListType1) {
        newParams.entityListType1 = getTypeKeyFromParamValue(newParams.entityListType1);
    } else if (newParams.entityType1) {
        if (isListType(newParams.entityType1)) {
            newParams.entityListType1 = getTypeKeyFromParamValue(newParams.entityType1, true);
            delete newParams.entityType1;
        } else {
            newParams.entityType1 = getTypeKeyFromParamValue(newParams.entityType1);
        }
    }
    if (newParams.pageEntityType)
        newParams.pageEntityType = getTypeKeyFromParamValue(newParams.pageEntityType);

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
    const search = query
        ? qs.stringify(query, {
              addQueryPrefix: true,
              arrayFormat: 'repeat',
              encodeValuesOnly: true
          })
        : '';

    return {
        pathname,
        search,
        url: pathname + search
    };
}

const pageTypesToParamNames = {
    [pageTypes.ENTITY]: entityParamNames,
    [pageTypes.LIST]: listParamNames
};

function getLastUsedParamName(urlParams) {
    const pageType = getPageType(urlParams);
    if (!pageType) return null;
    const paramTypes = pageTypesToParamNames[pageType];
    if (!paramTypes) return null;
    const propNames = Object.values(paramTypes).reverse();
    if (urlParams.entityListType2) {
        propNames[propNames.indexOf('entityType2')] = 'entityListType2';
    }
    if (urlParams.entityListType1) {
        propNames[propNames.indexOf('entityType1')] = 'entityListType1';
    }

    for (let i = 0; i < propNames.length; i += 1) {
        const propName = propNames[i];
        if (urlParams[propName]) return propName;
    }
    return null;
}

function isIdParamName(param) {
    if (!param) return false;
    return param.toLowerCase().includes('entityid');
}

function isType(input) {
    return !!entityTypes[input];
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
        const { urlParams, q } = this;
        let newParams = { ...urlParams };
        let newQuery = { ...q };
        const lastUsedParamName = getLastUsedParamName(urlParams);
        const lastUsedParamIsId = isIdParamName(lastUsedParamName);
        const pageType = urlParams.pageEntityListType || urlParams.pageEntityType;
        const entityType1 = urlParams.entityType1 || urlParams.entityListType1;
        const entityType2 = urlParams.entityType2 || urlParams.entityListType2;
        const isListPath = !!urlParams.pageEntityListType;
        const tabs = getTabsPerEntity(entityType2, urlParams.context);
        // Not pushing a value, return
        if (!val) {
            return this;
        }

        // Pushing initial values, use base instead
        if (!pageType) {
            return this.base(val, val2);
        }

        // replacement: if pushing type or id onto a stack the ends in type or id, replace instead of push
        if (isType(urlParams[lastUsedParamName]) === isType(val)) {
            newParams[lastUsedParamName] = val;
        }

        // Entity push: if pushing both a type and id at the same time, then entity <> entity
        else if (val && val2) {
            if (!lastUsedParamIsId) {
                throw new Error({
                    message: `Can't push an entity type and id onto a list. Use push(id) instead of push(type,id)`
                });
            }
            if (!isListPath && !urlParams.entityId1) {
                newParams.entityType1 = val;
                newParams.entityId1 = val2;
            } else if (!urlParams.entityType2 && !urlParams.entityListType2) {
                newParams.entityType2 = val;
                newParams.entityId2 = val2;
            } else {
                newParams = tabs.includes(val)
                    ? {
                          context: urlParams.context,
                          pageEntityType: entityType2,
                          pageEntityId: urlParams.entityId2,
                          entityListType1: val,
                          entityId1: val2
                      }
                    : (newParams = {
                          context: urlParams.context,
                          pageEntityType: val,
                          pageEntityId: val2
                      });
                newQuery = null;
            }
        }

        // Id push: pushing an id value alone
        else if (!isType(val)) {
            if (!urlParams.entityId1) {
                newParams.entityId1 = val;
            } else if (!urlParams.entityId2) {
                newParams.entityId2 = val;
            } else {
                // overflow:
                throw new Error(`can't push id onto UI ${this.url()}`);
            }
        }

        // Type push: pushing a type value alone
        else if (!isListPath && !entityType1) {
            newParams.entityListType1 = val;
        } else if (!entityType2) {
            newParams.entityListType2 = val;
        } else {
            // overflow, preserve last entity context
            newParams = tabs.includes(val)
                ? {
                      context: urlParams.context,
                      pageEntityType: entityType2,
                      pageEntityId: urlParams.entityId2,
                      entityListType1: val
                  }
                : {
                      context: urlParams.context,
                      pageEntityListType: val
                  };
            newQuery = null;
        }

        this.urlParams = newParams;
        this.q = newQuery;
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
        if (paramName === 'entityListType2') delete this.q.s2;

        return this;
    }

    set(paramName, value) {
        const { urlParams } = this;
        urlParams[paramName] = value;
        return this;
    }

    query(queryChanges) {
        if (!queryChanges) this.q = {};
        else this.q = merge(this.q, queryChanges);

        return this;
    }

    clearSidePanelParams() {
        const p = this.urlParams;

        // if in an entity page overview, reset url to overview on sidepanel close
        if (p.pageEntityType && !p.entityListType1) {
            delete p.entityType1;
        }

        delete p.entityId1;
        delete p.entityType2;
        delete p.entityListType2;
        delete p.entityId2;
        delete this.q.s2;

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
