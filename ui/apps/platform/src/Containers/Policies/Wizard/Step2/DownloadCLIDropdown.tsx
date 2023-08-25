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
                <DropdownItem value="darwin-amd64" component="button">
                    Mac x86_64
                </DropdownItem>,
                <DropdownItem value="linux-amd64" component="button">
                    Linux x86_64
                </DropdownItem>,
                <DropdownItem value="linux-ppc64le" component="button">
                    Linux ppc64le
                </DropdownItem>,
                <DropdownItem value="linux-s390x" component="button">
                    Linux s390x
                </DropdownItem>,
                <DropdownItem value="windows-amd64" component="button">
                    Windows x86_64
                </DropdownItem>,
            ]}
        />
    );
}

export default DownloadCLIDropdown;
