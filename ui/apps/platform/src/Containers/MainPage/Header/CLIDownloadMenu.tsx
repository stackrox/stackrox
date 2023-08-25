import React, { useState, ReactElement } from 'react';
import { withRouter } from 'react-router-dom';
import { connect } from 'react-redux';
import { DownloadIcon } from '@patternfly/react-icons';
import {
    ApplicationLauncher,
    ApplicationLauncherItem,
    Flex,
    FlexItem,
} from '@patternfly/react-core';
import Raven from 'raven-js';

import { actions } from 'reducers/notifications';
import downloadCLI from 'services/CLIService';

type CLIDownloadMenuProps = {
    addToast: (msg) => void;
    removeToast: () => void;
};

function CLIDownloadMenu({ addToast, removeToast }: CLIDownloadMenuProps): ReactElement {
    const [isCLIMenuOpen, setIsCLIMenuOpen] = useState(false);

    function handleDownloadCLI(os: string) {
        return () => {
            downloadCLI(os)
                .then(() => {
                    setIsCLIMenuOpen(false);
                })
                .catch((err) => {
                    addToast(`Error while downloading roxctl for ${os}`);
                    removeToast();
                    Raven.captureException(err);
                });
        };
    }

    const appLauncherItems = [
        <ApplicationLauncherItem
            key="app-launcher-item-cli-mac-amd64"
            component="button"
            onClick={handleDownloadCLI('darwin-amd64')}
        >
            Mac x86_64
        </ApplicationLauncherItem>,
        <ApplicationLauncherItem
            key="app-launcher-item-cli-linux-amd64"
            component="button"
            onClick={handleDownloadCLI('linux-amd64')}
        >
            Linux x86_64
        </ApplicationLauncherItem>,
        <ApplicationLauncherItem
            key="app-launcher-item-cli-linux-ppc64le"
            component="button"
            onClick={handleDownloadCLI('linux-ppc64le')}
        >
            Linux ppc64le
        </ApplicationLauncherItem>,
        <ApplicationLauncherItem
            key="app-launcher-item-cli-linux-s390x"
            component="button"
            onClick={handleDownloadCLI('linux-s390x')}
        >
            Linux s390x
        </ApplicationLauncherItem>,
        <ApplicationLauncherItem
            key="app-launcher-item-cli-windows-amd64"
            component="button"
            onClick={handleDownloadCLI('windows-amd64')}
        >
            Windows x86_64
        </ApplicationLauncherItem>,
    ];

    function toggleCLIMenu() {
        setIsCLIMenuOpen(!isCLIMenuOpen);
    }

    const CLIDownloadIcon = (
        <Flex alignItems={{ default: 'alignItemsCenter' }} spaceItems={{ default: 'spaceItemsSm' }}>
            <FlexItem>
                <DownloadIcon alt="" />
            </FlexItem>
            <FlexItem>CLI</FlexItem>
        </Flex>
    );

    return (
        <ApplicationLauncher
            aria-label="CLI Download Menu"
            onToggle={toggleCLIMenu}
            isOpen={isCLIMenuOpen}
            items={appLauncherItems}
            position="right"
            data-quickstart-id="qs-masthead-climenu"
            toggleIcon={CLIDownloadIcon}
        />
    );
}

const mapDispatchToProps = {
    addToast: actions.addNotification,
    removeToast: actions.removeOldestNotification,
};

export default withRouter(connect(null, mapDispatchToProps)(CLIDownloadMenu));
