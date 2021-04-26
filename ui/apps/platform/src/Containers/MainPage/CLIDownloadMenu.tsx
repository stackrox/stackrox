/* eslint-disable react/jsx-no-bind */
import React, { useState, ReactElement } from 'react';
import { withRouter } from 'react-router-dom';
import { connect } from 'react-redux';
import { DownloadIcon } from '@patternfly/react-icons';
import { ApplicationLauncher, ApplicationLauncherItem } from '@patternfly/react-core';
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

    const appLauncherItems: ReactElement[] = [];
    appLauncherItems.push(
        <>
            <ApplicationLauncherItem
                key="app-launcher-item-cli-mac"
                component="button"
                onClick={handleDownloadCLI('darwin')}
            >
                Mac 64-bit
            </ApplicationLauncherItem>
            <ApplicationLauncherItem
                key="app-launcher-item-cli-linux"
                component="button"
                onClick={handleDownloadCLI('linux')}
            >
                Linux 64-bit
            </ApplicationLauncherItem>
            <ApplicationLauncherItem
                key="app-launcher-item-cli-windows"
                component="button"
                onClick={handleDownloadCLI('windows')}
            >
                Windows 64-bit
            </ApplicationLauncherItem>
        </>
    );

    function toggleCLIMenu() {
        setIsCLIMenuOpen(!isCLIMenuOpen);
    }

    const CLIDownloadIcon = (
        <div className="flex items-center pt-1">
            <DownloadIcon alt="" />
            <span className="pl-1">CLI</span>
        </div>
    );

    return (
        <ApplicationLauncher
            key="cli-download-menu"
            aria-label="CLI Download Menu"
            className="co-app-launcher"
            onSelect={() => {}}
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
    // eslint-disable-next-line @typescript-eslint/no-unsafe-return
    addToast: actions.addNotification,
    removeToast: actions.removeOldestNotification,
};

export default withRouter(connect(null, mapDispatchToProps)(CLIDownloadMenu));
