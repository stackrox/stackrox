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
            key="app-launcher-item-cli-mac"
            component="button"
            onClick={handleDownloadCLI('darwin')}
        >
            Mac 64-bit
        </ApplicationLauncherItem>,
        <ApplicationLauncherItem
            key="app-launcher-item-cli-linux"
            component="button"
            onClick={handleDownloadCLI('linux')}
        >
            Linux 64-bit
        </ApplicationLauncherItem>,
        <ApplicationLauncherItem
            key="app-launcher-item-cli-windows"
            component="button"
            onClick={handleDownloadCLI('windows')}
        >
            Windows 64-bit
        </ApplicationLauncherItem>,
    ];

    function toggleCLIMenu() {
        setIsCLIMenuOpen(!isCLIMenuOpen);
    }

    // The className prop overrides `font-weight: 600` for button in ui-components.css file.
    const CLIDownloadIcon = (
        <Flex
            alignItems={{ default: 'alignItemsCenter' }}
            spaceItems={{ default: 'spaceItemsSm' }}
            className="pf-u-font-weight-normal"
        >
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
    // TODO: type redux props
    addToast: actions.addNotification,
    removeToast: actions.removeOldestNotification,
};

export default withRouter(connect(null, mapDispatchToProps)(CLIDownloadMenu));
