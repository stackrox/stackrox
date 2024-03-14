import { vulnerabilitiesNodeCvesPath } from 'routePaths';
import { SearchFilter } from 'types/search';
import { getQueryString } from 'utils/queryStringUtils';

import { NodeEntityTab } from '../../types';

type NodeCvesSearch = {
    entityTab?: NodeEntityTab;
    s?: SearchFilter;
};

export function getOverviewCvesPath(nodeCvesSearch: NodeCvesSearch): string {
    return `${vulnerabilitiesNodeCvesPath}${getQueryString(nodeCvesSearch)}`;
}
