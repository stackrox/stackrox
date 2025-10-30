import React, { useState } from 'react';
import { DropdownItem } from '@patternfly/react-core';

import downloadCLI from 'services/CLIService';
import MenuDropdown from 'Components/PatternFly/MenuDropdown';

function DownloadCLIDropdown({ hasBuild }) {
    const [isCLIDownloading, setIsCLIDownloading] = useState(false);

    // TODO: Show a success and error message
    async function handleDownloadCLI(_, value) {
        setIsCLIDownloading(true);
        await downloadCLI(value);
        setIsCLIDownloading(false);
    }

    return (
        <MenuDropdown
            isDisabled={!hasBuild || isCLIDownloading}
            isLoading={isCLIDownloading}
            toggleText="Download CLI"
            onSelect={handleDownloadCLI}
        >
            <DropdownItem key="darwin-amd64" value="darwin-amd64">
                Mac x86_64
            </DropdownItem>
            <DropdownItem key="darwin-arm64" value="darwin-arm64">
                Mac arm_64
            </DropdownItem>
            <DropdownItem key="linux-amd64" value="linux-amd64">
                Linux x86_64
            </DropdownItem>
            <DropdownItem key="linux-arm64" value="linux-arm64">
                Linux arm_64
            </DropdownItem>
            <DropdownItem key="linux-ppc64le" value="linux-ppc64le">
                Linux ppc64le
            </DropdownItem>
            <DropdownItem key="linux-s390x" value="linux-s390x">
                Linux s390x
            </DropdownItem>
            <DropdownItem key="windows-amd64" value="windows-amd64">
                Windows x86_64
            </DropdownItem>
        </MenuDropdown>
    );
}

export default DownloadCLIDropdown;
