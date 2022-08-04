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
    const [fileContent, setFileContent] = useState('');
    const [filename, setFilename] = useState('');
    const [isFileLoading, setIsFileLoading] = useState(false);

    function handleCancelModal() {
        setFileContent('');
        setFilename('');
        cancelModal();
    }

    function handleFileChange(newFileContent, newFilename) {
        setFileContent(newFileContent);
        setFilename(newFilename);
        if (newFileContent) {
            const jsonObj = JSON.parse(newFileContent);
            if (jsonObj?.policies) {
                setPolicies(jsonObj.policies);
            }
        }
    }

    function handleFileReadStarted() {
        setIsFileLoading(true);
    }

    function handleFileReadFinished() {
        setIsFileLoading(false);
    }

    return (
        <>
            <ModalBoxBody>
                Upload a policy JSON file to import a previously exported security policy
                <FileUpload
                    id="policies-json-import"
                    type="text"
                    className="pf-u-mt-md"
                    value={fileContent}
                    filename={filename}
                    filenamePlaceholder="Drag and drop a file or upload one"
                    onChange={handleFileChange}
                    onReadStarted={handleFileReadStarted}
                    onReadFinished={handleFileReadFinished}
                    isLoading={isFileLoading}
                    browseButtonText="Upload"
                    dropzoneProps={{
                        accept: '.json',
                    }}
                />
                {policies?.length > 0 && fileContent !== '' && (
                    <Flex direction={{ default: 'column' }} className="pf-u-mt-md">
                        <FlexItem>
                            <Title headingLevel="h3">
                                The following {`${pluralize('policy', policies.length)}`} will be
                                imported:
                            </Title>
                        </FlexItem>
                        <FlexItem data-testid="policies-to-import">
                            {policies.map(({ id, name }, idx) => (
                                <div key={id} className={idx === 0 ? '' : 'pf-u-pt-sm'}>
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
