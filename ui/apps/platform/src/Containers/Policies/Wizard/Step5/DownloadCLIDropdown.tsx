import React from 'react';
import { Spinner, Bullseye } from '@patternfly/react-core';
import { Dropdown, DropdownToggle, DropdownItem } from '@patternfly/react-core/deprecated';

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
                            <Spinner size="md" />
                        </Bullseye>
                    ) : (
                        'Download CLI'
                    )}
                </DropdownToggle>
            }
            isOpen={isCLIDropdownOpen}
            onSelect={handleDownloadCLI}
            dropdownItems={[
                <DropdownItem key="darwin-amd64" value="darwin-amd64" component="button">
                    Mac x86_64
                </DropdownItem>,
                <DropdownItem key="darwin-arm64" value="darwin-arm64" component="button">
                    Mac arm_64
                </DropdownItem>,
                <DropdownItem key="linux-amd64" value="linux-amd64" component="button">
                    Linux x86_64
                </DropdownItem>,
                <DropdownItem key="linux-arm64" value="linux-arm64" component="button">
                    Linux arm_64
                </DropdownItem>,
                <DropdownItem key="linux-ppc64le" value="linux-ppc64le" component="button">
                    Linux ppc64le
                </DropdownItem>,
                <DropdownItem key="linux-s390x" value="linux-s390x" component="button">
                    Linux s390x
                </DropdownItem>,
                <DropdownItem key="windows-amd64" value="windows-amd64" component="button">
                    Windows x86_64
                </DropdownItem>,
            ]}
        />
    );
}

export default DownloadCLIDropdown;
