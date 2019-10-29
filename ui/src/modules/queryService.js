import pluralize from 'pluralize';
import entityTypes from 'constants/entityTypes';
import useCases from 'constants/useCaseTypes';
import { NODE_FRAGMENT } from 'queries/node';
import { DEPLOYMENT_FRAGMENT } from 'queries/deployment';
import { NAMESPACE_FRAGMENT } from 'queries/namespace';
import { SUBJECT_WITH_CLUSTER_FRAGMENT } from 'queries/subject';
import { ROLE_FRAGMENT } from 'queries/role';
import { SECRET_FRAGMENT } from 'queries/secret';
import { SERVICE_ACCOUNT_FRAGMENT } from 'queries/serviceAccount';
import { CONTROL_FRAGMENT } from 'queries/controls';
import { POLICY_FRAGMENT } from 'queries/policy';
import { IMAGE_FRAGMENT } from 'queries/image';
import { COMPONENT_LIST_FRAGMENT, CVE_LIST_FRAGMENT } from 'Containers/VulnMgmt/VulnMgmt.fragments';

function objectToWhereClause(query) {
    if (!query) return '';

    return Object.entries(query)
        .reduce((acc, entry) => {
            const [key, value] = entry;
            if (!key || !value) return acc;
            if (typeof value === 'undefined' || value === '') return acc;
            const flatValue = Array.isArray(value) ? value.join() : value;
            const needsExactMatch =
                key.toLowerCase().indexOf(' id') !== -1 && value.indexOf(',') === -1;
            const queryValue = needsExactMatch ? `"${flatValue}"` : flatValue;
            return `${acc}${key}:${queryValue}+`;
        }, '')
        .slice(0, -1);
}

function entityContextToQueryObject(entityContext) {
    if (!entityContext) return {};

    // TO DO: waiting for backend to use COMPONENT ID instead of NAME and VERSION. workaround for now
    return Object.keys(entityContext).reduce((acc, key) => {
        const entityQueryObj = {};
        if (key === entityTypes.IMAGE) {
            entityQueryObj[`${key} SHA`] = entityContext[key];
        } else if (key === entityTypes.COMPONENT) {
            const parsedComponentID = entityContext[key].split(':');
            [entityQueryObj[`${key} NAME`], entityQueryObj[`${key} VERSION`]] = parsedComponentID;
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

function getListFieldName(entityType, listType) {
    // TODO: Back end should rename these fields and these exceptions should be removed
    if (entityType === entityTypes.CLUSTER) {
        if (listType === entityTypes.ROLE) {
            return 'k8sroles';
        }
        if (listType === entityTypes.CONTROL) return 'complianceResults';
    }

    if (entityType === entityTypes.DEPLOYMENT) {
        if (listType === entityTypes.POLICY) {
            return 'failingPolicies';
        }
    }

    if (entityType === entityTypes.NAMESPACE) {
        if (listType === entityTypes.ROLE) {
            return 'k8sroles';
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
            return 'subjectWithClusterFields';
        case entityTypes.ROLE:
            return 'k8roleFields';
        case entityTypes.SECRET:
            return 'secretFields';
        case entityTypes.POLICY:
            return 'policyFields';
        case entityTypes.SERVICE_ACCOUNT:
            return 'serviceAccountFields';
        case entityTypes.CONTROL:
            return 'controlFields';
        case entityTypes.CVE:
            return 'cveListFields';
        case entityTypes.COMPONENT:
            return 'componentListFields';
        default:
            return '';
    }
}

function getFragment(entityType, appContext) {
    const standardFields = [
        useCases.CONFIG_MANAGEMENT,
        useCases.COMPLIANCE,
        useCases.VULN_MANAGEMENT
    ];

    if (standardFields.includes(appContext)) {
        switch (entityType) {
            case entityTypes.IMAGE:
                return IMAGE_FRAGMENT;
            case entityTypes.NODE:
                return NODE_FRAGMENT;
            case entityTypes.DEPLOYMENT:
                return DEPLOYMENT_FRAGMENT;
            case entityTypes.NAMESPACE:
                return NAMESPACE_FRAGMENT;
            case entityTypes.SUBJECT:
                return SUBJECT_WITH_CLUSTER_FRAGMENT;
            case entityTypes.ROLE:
                return ROLE_FRAGMENT;
            case entityTypes.SECRET:
                return SECRET_FRAGMENT;
            case entityTypes.POLICY:
                return POLICY_FRAGMENT;
            case entityTypes.SERVICE_ACCOUNT:
                return SERVICE_ACCOUNT_FRAGMENT;
            case entityTypes.CONTROL:
                return CONTROL_FRAGMENT;
            case entityTypes.CVE:
                return CVE_LIST_FRAGMENT;
            case entityTypes.COMPONENT:
                return COMPONENT_LIST_FRAGMENT;
            default:
                return '';
        }
    }

    throw new Error(`No fragment found for ${entityType} in app context ${appContext}`);
}

function getFragmentInfo(entityType, listType, appContext) {
    const listFieldName = getListFieldName(entityType, listType);
    const fragmentName = getFragmentName(listType);
    const fragment = getFragment(listType, appContext);

    return {
        listFieldName,
        fragmentName,
        fragment
    };
}

export default {
    objectToWhereClause,
    entityContextToQueryObject,
    entityContextToQueryString,
    getEntityWhereClause,
    getQueryBasedOnSearchContext,
    getFragmentInfo
};
