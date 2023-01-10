import { Button, FileUpload } from '@patternfly/react-core';
import { FileUploadIcon } from '@patternfly/react-icons';
import React from 'react';

type UploadYAMLButtonProps = {
    onFileInputChange: (
        _event: React.ChangeEvent<HTMLInputElement> | React.DragEvent<HTMLElement>,
        file: File
    ) => void;
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
            <div className="pf-u-hidden">
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
