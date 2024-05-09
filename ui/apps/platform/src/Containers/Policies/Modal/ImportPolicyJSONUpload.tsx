import React, { ReactElement, useState } from 'react';
import pluralize from 'pluralize';
import {
    Button,
    FileUpload,
    Title,
    Flex,
    FlexItem,
    ModalBoxFooter,
    ModalBoxBody,
} from '@patternfly/react-core';

import { ListPolicy } from 'types/policy.proto';

type ImportPolicyJSONUploadProps = {
    cancelModal: () => void;
    startImportPolicies: (policiesList) => void;
    setPolicies: (policies) => void;
    policies: ListPolicy[];
};

function ImportPolicyJSONUpload({
    cancelModal,
    startImportPolicies,
    setPolicies,
    policies,
}: ImportPolicyJSONUploadProps): ReactElement {
    const [fileContent, setFileContent] = useState<File>();
    const [filename, setFilename] = useState('');
    const [isFileLoading, setIsFileLoading] = useState(false);

    function handleCancelModal() {
        setFileContent(undefined);
        setFilename('');
        cancelModal();
    }

    async function handleFileChange(e, newFileContent) {
        setFileContent(newFileContent);
        setFilename(newFileContent.name);
        if (newFileContent) {
            const jsonFile = await newFileContent.text();
            const jsonObj = JSON.parse(jsonFile);
            if (jsonObj?.policies) {
                setPolicies(jsonObj.policies);
            }
        }
    }

    function handleTextOrDataChange(e, value: string) {
        setFileContent(value as unknown as File);
    }

    function handleFileReadStarted() {
        setIsFileLoading(true);
    }

    function handleFileReadFinished() {
        setIsFileLoading(false);
    }

    function handleClear() {
        setFileContent(undefined);
        setFilename('');
    }

    return (
        <>
            <ModalBoxBody>
                Upload a policy JSON file to import a previously exported security policy
                <FileUpload
                    id="policies-json-import"
                    type="text"
                    className="pf-v5-u-mt-md"
                    value={fileContent}
                    filename={filename}
                    filenamePlaceholder="Drag and drop a file or upload one"
                    onFileInputChange={handleFileChange}
                    onDataChange={handleTextOrDataChange}
                    onTextChange={handleTextOrDataChange}
                    onReadStarted={handleFileReadStarted}
                    onReadFinished={handleFileReadFinished}
                    onClearClick={handleClear}
                    isLoading={isFileLoading}
                    browseButtonText="Upload"
                    dropzoneProps={{
                        accept: { 'application/json': ['.json'] },
                    }}
                />
                {policies?.length > 0 && fileContent && (
                    <Flex direction={{ default: 'column' }} className="pf-v5-u-mt-md">
                        <FlexItem>
                            <Title headingLevel="h3">
                                The following {`${pluralize('policy', policies.length)}`} will be
                                imported:
                            </Title>
                        </FlexItem>
                        <FlexItem data-testid="policies-to-import">
                            {policies.map(({ id, name }, idx) => (
                                <div key={id} className={idx === 0 ? '' : 'pf-v5-u-pt-sm'}>
                                    {name}
                                </div>
                            ))}
                        </FlexItem>
                    </Flex>
                )}
            </ModalBoxBody>
            <ModalBoxFooter>
                <Button
                    key="import"
                    variant="primary"
                    onClick={startImportPolicies}
                    isDisabled={policies.length === 0}
                >
                    Begin import
                </Button>
                <Button key="cancel" variant="link" onClick={handleCancelModal}>
                    Cancel
                </Button>
            </ModalBoxFooter>
        </>
    );
}

export default ImportPolicyJSONUpload;
