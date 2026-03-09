import { Button, FileUpload } from '@patternfly/react-core';
import type { DropEvent } from '@patternfly/react-core';
import { FileUploadIcon } from '@patternfly/react-icons';

const UPLOAD_BUTTON_TEXT = 'Upload YAML';

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
                    // We search for the hidden file upload button here because the `<FileUpload>` component
                    // does not support rendering an icon within the internal `<Button>` component.
                    const fileUploadButtons = document.querySelectorAll<HTMLButtonElement>(
                        '.pf-v6-c-file-upload button'
                    );
                    Array.from(fileUploadButtons ?? [])
                        .find((button) => button.textContent === UPLOAD_BUTTON_TEXT)
                        ?.click();
                }}
            >
                Upload YAML
            </Button>
            <div className="pf-v6-u-hidden">
                <FileUpload
                    id="upload-network-policy"
                    filenamePlaceholder="Drag and drop a YAML or upload one"
                    onFileInputChange={onFileInputChange}
                    hideDefaultPreview
                    browseButtonText={UPLOAD_BUTTON_TEXT}
                />
            </div>
        </>
    );
}

export default UploadYAMLButton;
