import React, { useState, ReactElement } from 'react';
import { connect } from 'react-redux';
import { DownloadIcon } from '@patternfly/react-icons';
import {
    Dropdown,
    DropdownItem,
    DropdownList,
    Flex,
    FlexItem,
    MenuToggle,
} from '@patternfly/react-core';
import Raven from 'raven-js';

import { actions } from 'reducers/notifications';
import downloadCLI from 'services/CLIService';

const cliDownloadOptions = [
    { os: 'darwin-amd64', display: 'Mac x86_64' },
    { os: 'darwin-arm64', display: 'Mac arm_64' },
    { os: 'linux-amd64', display: 'Linux x86_64' },
    { os: 'linux-arm64', display: 'Linux arm_64' },
    { os: 'linux-ppc64le', display: 'Linux ppc64le' },
    { os: 'linux-s390x', display: 'Linux s390x' },
    { os: 'windows-amd64', display: 'Windows x86_64' },
] as const;

type AvailableOS = (typeof cliDownloadOptions)[number]['os'];

type CLIDownloadMenuProps = {
    addToast: (msg) => void;
    removeToast: () => void;
};

function CLIDownloadMenu({ addToast, removeToast }: CLIDownloadMenuProps): ReactElement {
    const [isCLIMenuOpen, setIsCLIMenuOpen] = useState(false);

    function handleDownloadCLI(os: AvailableOS) {
        return () => {
            setIsCLIMenuOpen(false);
            addToast(`Downloading roxctl for ${os}`);
            downloadCLI(os)
                .catch((err) => {
                    addToast(`Error while downloading roxctl for ${os}`);
                    removeToast();
                    Raven.captureException(err);
                })
                .finally(() => {
                    removeToast();
                });
        };
    }

    return (
        <Dropdown
            isOpen={isCLIMenuOpen}
            onOpenChange={(isOpen) => setIsCLIMenuOpen(isOpen)}
            onOpenChangeKeys={['Escape', 'Tab']}
            popperProps={{ position: 'right' }}
            toggle={(toggleRef) => (
                <MenuToggle
                    aria-label="CLI Download Menu"
                    ref={toggleRef}
                    variant="plain"
                    onClick={() => setIsCLIMenuOpen((wasOpen) => !wasOpen)}
                    isExpanded={isCLIMenuOpen}
                >
                    <Flex
                        alignItems={{ default: 'alignItemsCenter' }}
                        spaceItems={{ default: 'spaceItemsSm' }}
                    >
                        <FlexItem>
                            <DownloadIcon />
                        </FlexItem>
                        <FlexItem>CLI</FlexItem>
                    </Flex>
                </MenuToggle>
            )}
        >
            <DropdownList>
                {cliDownloadOptions.map(({ os, display }) => (
                    <DropdownItem key={os} onClick={handleDownloadCLI(os)}>
                        {display}
                    </DropdownItem>
                ))}
            </DropdownList>
        </Dropdown>
    );
}

const mapDispatchToProps = {
    addToast: actions.addNotification,
    removeToast: actions.removeOldestNotification,
};

export default connect(null, mapDispatchToProps)(CLIDownloadMenu);
