import pluralize from 'pluralize';

import entityTypes from 'constants/entityTypes';
import useCases from 'constants/useCaseTypes';
import decodeBase64 from 'utils/decodeBase64/decodeBase64';
import { NODE_FRAGMENT } from 'queries/node';
import { DEPLOYMENT_FRAGMENT } from 'queries/deployment';
import { NAMESPACE_FRAGMENT, CONFIG_NAMESPACE_FRAGMENT } from 'queries/namespace';
import { SUBJECT_WITH_CLUSTER_FRAGMENT, SUBJECT_FRAGMENT } from 'queries/subject';
import { K8S_ROLE_FRAGMENT } from 'queries/role';
import { SECRET_FRAGMENT } from 'queries/secret';
import { SERVICE_ACCOUNT_FRAGMENT } from 'queries/serviceAccount';
import { CONTROL_FRAGMENT } from 'queries/controls';
import { POLICY_FRAGMENT } from 'queries/policy';
import { IMAGE_FRAGMENT } from 'queries/image';
import {
    VULN_COMPONENT_LIST_FRAGMENT,
    VULN_CVE_LIST_FRAGMENT,
    IMAGE_LIST_FRAGMENT as VULN_IMAGE_LIST_FRAGMENT,
    CLUSTER_LIST_FRAGMENT as VULN_CLUSTER_LIST_FRAGMENT,
    DEPLOYMENT_LIST_FRAGMENT as VULN_DEPLOYMENT_LIST_FRAGMENT,
    NAMESPACE_LIST_FRAGMENT as VULN_NAMESPACE_LIST_FRAGMENT,
    POLICY_LIST_FRAGMENT as VULN_POLICY_LIST_FRAGMENT,
} from 'Containers/VulnMgmt/VulnMgmt.fragments';
import { DEFAULT_PAGE_SIZE } from 'Components/Table';

function objectToWhereClause(query, delimiter = '+') {
    if (!query) {
        return '';
    }

    return Object.entries(query)
        .reduce((acc, entry) => {
            const [key, value] = entry;
            if (!key || !value) {
                return acc;
            }
            if (typeof value === 'undefined' || value === '') {
                return acc;
            }
            const flatValue = Array.isArray(value) ? value.join() : value;
            const needsExactMatch =
                key.toLowerCase().indexOf(' id') !== -1 && value.indexOf(',') === -1;
            const queryValue = needsExactMatch ? `"${flatValue}"` : flatValue;
            return `${acc}${key}:${queryValue}${delimiter}`;
        }, '')
        .slice(0, -delimiter.length);
}

function entityContextToQueryObject(entityContext) {
    if (!entityContext) {
        return {};
    }

    // TODO: waiting for backend to use COMPONENT ID instead of NAME and VERSION. workaround for now
    return Object.keys(entityContext).reduce((acc, key) => {
        const entityQueryObj = {};
        if (key === entityTypes.IMAGE) {
            entityQueryObj[`${key} SHA`] = entityContext[key];
        } else if (key === entityTypes.COMPONENT) {
            const parsedComponentID = entityContext[key].split(':').map(decodeBase64);
            [entityQueryObj[`${key}`], entityQueryObj[`${key} VERSION`]] = parsedComponentID;
        } else if (key === entityTypes.CVE) {
            entityQueryObj[key] = entityContext[key];
        } else {
            entityQueryObj[`${key} ID`] = entityContext[key];
        }
        return { ...acc, ...entityQueryObj };
    }, {});
}

function entityContextToQueryString(entityContext) {
    const queryObject = entityContextToQueryObject(entityContext);
    return objectToWhereClause(queryObject);
}

function getEntityWhereClause(search, entityContext) {
    return objectToWhereClause({ ...search, ...entityContextToQueryObject(entityContext) });
}

function getQueryBasedOnSearchContext(query, searchParam) {
    return searchParam && query && query[searchParam] ? query[searchParam] : query || {};
}

function getListFieldName(entityType, listType, useCase) {
    // TODO: Back end should rename these fields and these exceptions should be removed
    if (entityType === entityTypes.COMPONENT) {
        if (listType === entityTypes.CVE) {
            return 'vulns';
        }
    }

    if (entityType === entityTypes.IMAGE) {
        if (listType === entityTypes.CVE) {
            return 'vulns';
        }
    }

    if (entityType === entityTypes.CLUSTER) {
        if (listType === entityTypes.CVE) {
            return 'vulns';
        }

        if (listType === entityTypes.ROLE) {
            return 'k8sRoles';
        }
        if (listType === entityTypes.CONTROL) {
            return 'complianceResults';
        }
    }

    if (entityType === entityTypes.NODE) {
        if (listType === entityTypes.CVE) {
            return 'vulns';
        }
    }

    if (entityType === entityTypes.DEPLOYMENT) {
        if (listType === entityTypes.CVE) {
            return 'vulns';
        }

        if (listType === entityTypes.POLICY) {
            if (useCase === useCases.VULN_MANAGEMENT) {
                return 'policies';
            }
            return 'failingPolicies';
        }
    }

    if (entityType === entityTypes.NAMESPACE) {
        if (listType === entityTypes.CVE) {
            return 'vulns';
        }

        if (listType === entityTypes.ROLE) {
            return 'k8sRoles';
        }
    }

    if (entityType === entityTypes.SERVICE_ACCOUNT) {
        if (listType === entityTypes.ROLE) {
            return 'k8sRoles';
        }
    }

    const name = pluralize(listType.toLowerCase());
    const parts = name.split('_');
    for (let i = 1; i < parts.length; i += 1) {
        parts[i] = parts[i].charAt(0).toUpperCase() + parts[i].slice(1);
    }

    return parts.join('');
}

function getFragmentName(entityType) {
    switch (entityType) {
        case entityTypes.IMAGE:
            return 'imageFields';
        case entityTypes.NODE:
            return 'nodeFields';
        case entityTypes.DEPLOYMENT:
            return 'deploymentFields';
        case entityTypes.NAMESPACE:
            return 'namespaceFields';
        case entityTypes.SUBJECT:
            return 'subjectFields';
        case entityTypes.ROLE:
            return 'k8RoleFields';
        case entityTypes.SECRET:
            return 'secretFields';
        case entityTypes.POLICY:
            return 'policyFields';
        case entityTypes.SERVICE_ACCOUNT:
            return 'serviceAccountFields';
        case entityTypes.CONTROL:
            return 'controlFields';
        case entityTypes.CVE:
            return 'cveFields';
        case entityTypes.COMPONENT:
            return 'componentFields';
        default:
            return '';
    }
}

function getFragment(entityType, useCase) {
    const defaultFragments = {
        [entityTypes.IMAGE]: IMAGE_FRAGMENT,
        [entityTypes.NODE]: NODE_FRAGMENT,
        [entityTypes.DEPLOYMENT]: DEPLOYMENT_FRAGMENT,
        [entityTypes.NAMESPACE]: NAMESPACE_FRAGMENT,
        [entityTypes.SUBJECT]: SUBJECT_WITH_CLUSTER_FRAGMENT,
        [entityTypes.ROLE]: K8S_ROLE_FRAGMENT,
        [entityTypes.SECRET]: SECRET_FRAGMENT,
        [entityTypes.POLICY]: POLICY_FRAGMENT,
        [entityTypes.SERVICE_ACCOUNT]: SERVICE_ACCOUNT_FRAGMENT,
        [entityTypes.CONTROL]: CONTROL_FRAGMENT,
    };

    const fragmentsByUseCase = {
        [useCases.CONFIG_MANAGEMENT]: {
            ...defaultFragments,
            [entityTypes.NAMESPACE]: CONFIG_NAMESPACE_FRAGMENT,
            [entityTypes.SUBJECT]: SUBJECT_FRAGMENT,
        },
        [useCases.VULN_MANAGEMENT]: {
            ...defaultFragments,
            [entityTypes.COMPONENT]: VULN_COMPONENT_LIST_FRAGMENT,
            [entityTypes.CVE]: VULN_CVE_LIST_FRAGMENT,
            [entityTypes.IMAGE]: VULN_IMAGE_LIST_FRAGMENT,
            [entityTypes.CLUSTER]: VULN_CLUSTER_LIST_FRAGMENT,
            [entityTypes.NAMESPACE]: VULN_NAMESPACE_LIST_FRAGMENT,
            [entityTypes.POLICY]: VULN_POLICY_LIST_FRAGMENT,
            [entityTypes.DEPLOYMENT]: VULN_DEPLOYMENT_LIST_FRAGMENT,
        },
    };

    const fragmentMap = fragmentsByUseCase[useCase] || defaultFragments;

    return fragmentMap[entityType];
}

function getFragmentInfo(entityType, listType, useCase) {
    const listFieldName = getListFieldName(entityType, listType, useCase);
    const fragmentName = getFragmentName(listType);
    const fragment = getFragment(listType, useCase);

    return {
        listFieldName,
        fragmentName,
        fragment,
    };
}

function getPagination(sort, page, pageSize = DEFAULT_PAGE_SIZE) {
    const sortObj = Array.isArray(sort) ? sort[0] : sort; // Back end can't support multiple sort right now, so just taking first sort

    if (!sortObj) {
        return null;
    }
    const offset = page * pageSize;
    const limit = pageSize;
    const paginationObj = {
        offset,
        limit,
    };

    if (!sortObj.id) {
        return paginationObj;
    }

    paginationObj.sortOption = {
        field: sortObj.id,
        reversed: sortObj.desc,
    };
    return paginationObj;
}

export default {
    objectToWhereClause,
    entityContextToQueryObject,
    entityContextToQueryString,
    getEntityWhereClause,
    getQueryBasedOnSearchContext,
    getFragmentInfo,
    getPagination,
};
