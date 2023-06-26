import { matchPath } from 'react-router-dom';
import qs, { ParsedQs } from 'qs';
import { Location, LocationState } from 'history';

import useCases from 'constants/useCaseTypes';
import { searchParams, sortParams, pagingParams } from 'constants/searchParams';
import WorkflowEntity from './WorkflowEntity';
import { WorkflowState } from './WorkflowState';
import {
    workflowPaths,
    urlEntityListTypes,
    urlEntityTypes,
    clustersPathWithParam,
    riskPath,
    violationsPath,
    policiesPath,
    userRolePath,
    accessControlPath,
} from '../routePaths';

type ParamsWithContext = {
    context: string;
    [key: string]: string;
};

const nonWorkflowUseCasePathEntries = Object.entries({
    CLUSTERS: clustersPathWithParam,
    RISK: riskPath,
    VIOLATIONS: violationsPath,
    POLICIES: policiesPath,
    USER: userRolePath, // however, it matches workflow list path
    ACCESS_CONTROL: accessControlPath,
});

function getNonWorkflowParams(pathname): ParamsWithContext {
    for (let i = 0; i < nonWorkflowUseCasePathEntries.length; i += 1) {
        const [useCaseKey, path] = nonWorkflowUseCasePathEntries[i];
        const matchedPath = matchPath(pathname, {
            path,
            exact: true,
        });

        if (matchedPath?.params) {
            const { params } = matchedPath;
            return {
                ...(params as Record<string, string>),
                context: useCases[useCaseKey],
            };
        }
    }

    return { context: '' };
}

function getParams(pathname): ParamsWithContext {
    // The type casts assert that workflow paths include a :context param.

    const matchedEntityPath = matchPath(pathname, {
        path: workflowPaths.ENTITY,
    });
    if (matchedEntityPath?.params) {
        return matchedEntityPath.params as ParamsWithContext;
    }

    const matchedListPath = matchPath(pathname, {
        path: workflowPaths.LIST,
    });
    if (matchedListPath?.params) {
        return matchedListPath.params as ParamsWithContext;
    }

    const matchedDashboardPath = matchPath(pathname, {
        path: workflowPaths.DASHBOARD,
        exact: true,
    });
    if (matchedDashboardPath?.params) {
        return matchedDashboardPath.params as ParamsWithContext;
    }

    return getNonWorkflowParams(pathname);
}

function getTypeKeyFromParamValue(value: string, listOnly = false): string | null {
    const listMatch = Object.entries(urlEntityListTypes).find((entry) => entry[1] === value);
    const entityMatch = Object.entries(urlEntityTypes).find((entry) => entry[1] === value);
    const match = listOnly ? listMatch : listMatch || entityMatch;
    return match ? match[0] : null;
}

function getEntityFromURLParam(type: string, id?: string): WorkflowEntity {
    return new WorkflowEntity(getTypeKeyFromParamValue(type), id);
}

function paramsToStateStack(params): WorkflowEntity[] {
    const { pageEntityListType, pageEntityType, pageEntityId, entityId1, entityId2 } = params;
    const { entityType1: urlEntityType1, entityType2: urlEntityType2 } = params;
    const entityListType1 = getTypeKeyFromParamValue(urlEntityType1, true);
    const entityListType2 = getTypeKeyFromParamValue(urlEntityType2, true);
    const entityType1 = getTypeKeyFromParamValue(urlEntityType1);
    const entityType2 = getTypeKeyFromParamValue(urlEntityType2);
    const stateArray: WorkflowEntity[] = [];
    if (!pageEntityListType && !pageEntityType) {
        return stateArray;
    }

    // List
    if (pageEntityListType) {
        stateArray.push(getEntityFromURLParam(pageEntityListType));

        if (entityId1) {
            stateArray.push(getEntityFromURLParam(pageEntityListType, entityId1));
        }
    } else {
        stateArray.push(getEntityFromURLParam(pageEntityType, pageEntityId));
        if (entityListType1) {
            stateArray.push(new WorkflowEntity(entityListType1));
        }
        if (entityType1 && entityId1) {
            stateArray.push(new WorkflowEntity(entityType1, entityId1));
        }
    }

    if (entityListType2) {
        stateArray.push(new WorkflowEntity(entityListType2));
    }
    if (entityType2 && entityId2) {
        stateArray.push(new WorkflowEntity(entityType2, entityId2));
    }

    return stateArray;
}

function formatSort(sort?: ParsedQs | ParsedQs[]): Record<string, unknown>[] | null {
    if (!sort) {
        return null;
    }

    let sorts: ParsedQs[];
    if (!Array.isArray(sort)) {
        sorts = [sort];
    } else {
        sorts = [...sort];
    }

    return sorts.map(({ id, desc }) => {
        return {
            id,
            desc: JSON.parse(desc as string),
        } as Record<string, unknown>;
    });
}

// Convert URL to workflow state and search objects
// note: this will read strictly from 'location' as 'match' is relative to the closest Route component
function parseURL(location: Location<LocationState>): WorkflowState {
    if (!location) {
        // TODO: be more specific, it could be an exception instead of a dummy object
        return new WorkflowState();
    }

    const { pathname, search } = location;

    const rawParams = getParams(pathname);
    const params = {
        ...rawParams,
        pageEntityId: rawParams.pageEntityId
            ? decodeURIComponent(rawParams.pageEntityId)
            : undefined,
    };

    const queryStr = search ? qs.parse(search, { ignoreQueryPrefix: true }) : {};

    const stateStackFromURLParams = paramsToStateStack(params) || [];

    const {
        [searchParams.page]: pageSearch,
        [searchParams.sidePanel]: sidePanelSearch,
        [sortParams.page]: pageSort,
        [sortParams.sidePanel]: sidePanelSort,
        [pagingParams.page]: pagePaging,
        [pagingParams.sidePanel]: sidePanelPaging,
    } = queryStr;

    const queryWorkflowState = queryStr.workflowState || [];
    const stateStackFromQueryString = !Array.isArray(queryWorkflowState)
        ? [queryWorkflowState as ParsedQs]
        : (queryWorkflowState as ParsedQs[]);
    const stateStack = stateStackFromQueryString.map(({ t, i }) => new WorkflowEntity(t, i));

    const workflowState = new WorkflowState(
        params.context,
        [...stateStackFromURLParams, ...stateStack],
        {
            [searchParams.page]: pageSearch || null,
            [searchParams.sidePanel]: sidePanelSearch || null,
        },
        {
            [sortParams.page]: formatSort(pageSort as ParsedQs | ParsedQs[]),
            [sortParams.sidePanel]: formatSort(sidePanelSort as ParsedQs | ParsedQs[]),
        },
        {
            [pagingParams.page]: parseInt((pagePaging as string) ?? '0', 10),
            [pagingParams.sidePanel]: parseInt((sidePanelPaging as string) ?? '0', 10),
        }
    );

    return workflowState;
}

export default parseURL;
