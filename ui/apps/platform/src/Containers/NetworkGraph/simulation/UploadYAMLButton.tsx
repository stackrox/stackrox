import { Button, DropEvent, FileUpload } from '@patternfly/react-core';
import { FileUploadIcon } from '@patternfly/react-icons';
import React from 'react';

type UploadYAMLButtonProps = {
    onFileInputChange: (_event: DropEvent, file: File) => void;
};

function UploadYAMLButton({ onFileInputChange }: UploadYAMLButtonProps) {
    return (
        <>
            <Button
                variant="secondary"
                icon={<FileUploadIcon />}
                onClick={() => {
                    document.getElementById('upload-network-policy-browse-button')?.click();
                }}
            >
                Upload YAML
            </Button>
            <div className="pf-v5-u-hidden">
                <FileUpload
                    id="upload-network-policy"
                    filenamePlaceholder="Drag and drop a YAML or upload one"
                    onFileInputChange={onFileInputChange}
                    hideDefaultPreview
                    browseButtonText="Upload"
                />
            </div>
        </>
    );
}

export default UploadYAMLButton;
