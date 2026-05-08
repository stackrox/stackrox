import type { ReactElement } from 'react';
import { Button, Flex, FlexItem } from '@patternfly/react-core';
import { SearchIcon } from '@patternfly/react-icons';

import LinkShim from 'Components/PatternFly/LinkShim';
import useIsRouteEnabled from 'hooks/useIsRouteEnabled';
import { searchPath } from 'routePaths';

import CLIDownloadMenu from './CLIDownloadMenu';
import ClusterStatusProblems from './ClusterStatusProblems';
import HelpMenu from './HelpMenu';
import ThemeToggleButton from './ThemeToggleButton';
import UserMenu from './UserMenu';

function MastheadToolbar(): ReactElement {
    const isRouteEnabled = useIsRouteEnabled();
    const isRouteEnabledForSearch = isRouteEnabled('search');

    // TODO: (PatternFly) need more robust mobile experience than just hiding tools
    // <PageHeaderToolsGroup visibility={{ default: 'hidden', md: 'visible' }}>
    return (
        <Flex spaceItems={{ default: 'spaceItemsSm' }}>
            {isRouteEnabledForSearch && (
                <FlexItem>
                    <Button
                        variant="plain"
                        component={LinkShim}
                        href={searchPath}
                        icon={<SearchIcon />}
                    >
                        Search
                    </Button>
                </FlexItem>
            )}
            <FlexItem>
                <CLIDownloadMenu />
            </FlexItem>
            <FlexItem>
                <ThemeToggleButton />
            </FlexItem>
            <FlexItem>
                <ClusterStatusProblems />
            </FlexItem>
            <FlexItem>
                <HelpMenu />
            </FlexItem>
            <FlexItem>
                <UserMenu />
            </FlexItem>
        </Flex>
    );
}

export default MastheadToolbar;
