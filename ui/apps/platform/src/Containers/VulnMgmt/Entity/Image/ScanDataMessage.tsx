import React, { ReactElement } from 'react';

import Message from 'Components/Message';
import getImageScanMessages from 'Containers/VulnMgmt/VulnMgmt.utils/getImageScanMessages';

function ScanDataMessage({ imageNotes = [], scanNotes = [] }): ReactElement | null {
    const imageScanMessages = getImageScanMessages(imageNotes || [], scanNotes || []);

    return Object.keys(imageScanMessages).length > 0 ? (
        <div className="px-4 pt-4">
            <Message
                type="error"
                message={
                    <div className="w-full">
                        <header className="text-lg pb-2 border-b border-alert-300 mb-2 w-full">
                            <h2 className="mb-1 font-700 uppercase">CVE Data May Be Inaccurate</h2>
                            <span>{imageScanMessages.header}</span>
                        </header>
                        <p>
                            <span>{imageScanMessages.body}</span> {imageScanMessages.extra}
                        </p>
                    </div>
                }
            />
        </div>
    ) : null;
}

export default ScanDataMessage;
