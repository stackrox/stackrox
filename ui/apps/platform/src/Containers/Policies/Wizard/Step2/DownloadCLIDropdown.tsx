import React from 'react';
import { Dropdown, DropdownToggle, DropdownItem, Spinner, Bullseye } from '@patternfly/react-core';

import downloadCLI from 'services/CLIService';

function DownloadCLIDropdown({ hasBuild }) {
    const [isCLIDropdownOpen, setIsCLIDropdownOpen] = React.useState(false);
    const [isCLIDownloading, setIsCLIDownloading] = React.useState(false);

    function handleDownloadCLI(event) {
        setIsCLIDownloading(true);
        setIsCLIDropdownOpen(false);
        downloadCLI(event.target.value)
            .then(() => {
                // TODO: Show a success message
            })
            .catch(() => {
                // TODO: Show an error message
            })
            .finally(() => {
                setIsCLIDownloading(false);
            });
    }
    return (
        <Dropdown
            toggle={
                <DropdownToggle
                    isDisabled={!hasBuild || isCLIDownloading}
                    onToggle={() => setIsCLIDropdownOpen(!isCLIDropdownOpen)}
                >
                    {isCLIDownloading ? (
                        <Bullseye>
                            <Spinner isSVG size="md" />
                        </Bullseye>
                    ) : (
                        'Download CLI'
                    )}
                </DropdownToggle>
            }
            isOpen={isCLIDropdownOpen}
            onSelect={handleDownloadCLI}
            dropdownItems={[
                <DropdownItem value="darwin" component="button">
                    Mac 64-bit
                </DropdownItem>,
                <DropdownItem value="linux" component="button">
                    Linux 64-bit
                </DropdownItem>,
                <DropdownItem value="windows" component="button">
                    Windows 64-bit
                </DropdownItem>,
            ]}
        />
    );
}

export default DownloadCLIDropdown;
